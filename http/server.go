package http

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/macq/flocons/cluster"
	"github.com/macq/flocons/config"
	. "github.com/macq/flocons/error"
	"github.com/macq/flocons/file"
	"github.com/macq/flocons/storage"
)

const FILE_WORKER_POOL_SIZE int = 10

type Server struct {
	config         *config.Config
	storage        *storage.Storage
	topologyClient cluster.TopologyClient
	httpServer     *http.Server
	fileJobs       chan serverJob
}

type serverJob struct {
	writer  http.ResponseWriter
	request *http.Request
	barrier *sync.Cond
}

func NewServer(config *config.Config, storage *storage.Storage, topologyClient cluster.TopologyClient) (*Server, error) {
	if config.Node.Port == 0 {
		return nil, NewInternalError("No port configured")
	}
	if config == nil {
		logger.Fatalf("Tried to create a new http server without config")
	}
	if storage == nil {
		logger.Fatalf("Tried to create a new http server without storage")
	}
	if topologyClient == nil {
		logger.Fatalf("Tried to create a new http server without topology client")
	}

	httpHandler := http.NewServeMux() // Don't use default handler because we would want several servers in parralel for tests
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(config.Node.Port), Handler: httpHandler}
	server := Server{
		config:         config,
		storage:        storage,
		topologyClient: topologyClient,
		httpServer:     httpServer,
		fileJobs:       make(chan serverJob),
	}
	server.start()
	return &server, nil
}

func (s *Server) start() {
	httpHandler, _ := s.httpServer.Handler.(*http.ServeMux)
	httpHandler.HandleFunc(FILES_PREFIX+"/", func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("Handle file request %s on node %s for ressource %s", r.Method, s.config.Node.Name, r.URL.Path)
		mutex := sync.Mutex{}
		barrier := sync.NewCond(&mutex)
		mutex.Lock()
		s.fileJobs <- serverJob{writer: w, request: r, barrier: barrier}
		barrier.Wait()
		mutex.Unlock()
	})
	httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Warnf("Unhandled URL request %s", r.URL.Path)
		w.WriteHeader(400)
	})

	logger.Info("Start workers")
	for i := 0; i < FILE_WORKER_POOL_SIZE; i++ {
		go s.waitForFileWork()
	}

	go func() {
		logger.Infof("Start http server on port %d", s.config.Node.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Http server failed %s", err)
		}
	}()
}

func (s *Server) waitForFileWork() {
	for job := range s.fileJobs {
		job.barrier.L.Lock()
		s.ServeFile(job.writer, job.request)
		job.barrier.Broadcast()
		job.barrier.L.Unlock()
	}
}

func (s *Server) ServeFile(w http.ResponseWriter, r *http.Request) {
	mimeType := r.Header.Get(CONTENT_TYPE)
	if mimeType == "" {
		mimeType = file.DEFAULT_FILE_MIME_TYPE
	}
	method := r.Method

	switch {
	case method == "HEAD":
		s.GetFile(w, r)
	case method == "GET":
		s.GetFileWithData(w, r)
	case method == "POST" && mimeType == file.DIRECTORY_MIME_TYPE:
		s.CreateDirectory(w, r)
	case method == "POST":
		s.CreateRegularFile(w, r)
	}
}

func (s *Server) CreateDirectory(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[len(FILES_PREFIX):]
	mode := headerToFileMode(r.Header)
	fi, err := s.storage.CreateDirectory(p, mode)
	if err != nil {
		returnError(err, w)
		return
	}
	fileInfoToHeader(fi, w.Header())
}

func (s *Server) CreateRegularFile(w http.ResponseWriter, r *http.Request) {
	if s.distributeRequestIfPossible(w, r) {
		return
	}
	p := r.URL.Path[len(FILES_PREFIX):]
	mode := headerToFileMode(r.Header)
	size := r.ContentLength
	buffer := make([]byte, size)
	if size > 0 {
		_, err := r.Body.Read(buffer)
		if err != nil && err != io.EOF {
			w.WriteHeader(400)
			return
		}
	}

	fi, err := s.storage.CreateRegularFile(p, mode, buffer)
	if err != nil {
		returnError(err, w)
		return
	}
	fileInfoToHeader(fi, w.Header())
}

func (s *Server) GetFile(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[len(FILES_PREFIX):]
	fi, err := s.storage.GetFile(p)
	if err != nil {
		// We didn't find the file, maybe it is not still synchronized but
		// if it was created, it is certainly on the node reponsible for it
		// let's try to dispatch the request
		if !s.distributeRequestIfPossible(w, r) {
			returnError(err, w)
		}
		return
	}
	fileInfoToHeader(fi, w.Header())
}

func (s *Server) GetFileWithData(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[len(FILES_PREFIX):]
	fi, err := s.storage.GetFile(p)
	if err != nil {
		// We didn't find the file, maybe it is not still synchronized but
		// if it was created, it is certainly on the node reponsible for it
		// let's try to dispatch the request
		if !s.distributeRequestIfPossible(w, r) {
			returnError(err, w)
		}
		return
	}
	var data []byte
	if fi.Mode().IsRegular() {
		storageFileInfo, _ := fi.(*file.FileInfo)
		data, err = storageFileInfo.Data()
		if err != nil {
			// We don't have the data, let's try to redirect to the node responsible
			// or any other node in the same shard
			if !s.tryRedirectToNode(w, r, storageFileInfo.Node(), storageFileInfo.Shard()) {
				returnError(err, w)
			}
			return
		}
	} else {
		files, err := s.storage.ReadDir(p)
		if err != nil {
			returnError(err, w)
			return
		}
		data, err = filesInfoToCsv(files)
		if err != nil {
			returnError(err, w)
			return
		}
	}
	fileInfoToHeader(fi, w.Header())
	w.Header().Set(CONTENT_LENGTH, strconv.FormatInt((int64)(len(data)), 10))
	w.Write(data)
}

func (s *Server) distributeRequestIfPossible(w http.ResponseWriter, r *http.Request) bool {
	if _, alreadyTraversed := r.URL.Query()[TRAVERSED_NODE_PARAMETER]; alreadyTraversed {
		return false
	}
	p := r.URL.Path[len(FILES_PREFIX):]
	node := s.topologyClient.GetNodeForObject(p)
	if node == nil || node.Name == s.config.Node.Name {
		return false
	}
	s.redirectToNode(w, r, node)
	return true
}

func (s *Server) tryRedirectToNode(w http.ResponseWriter, r *http.Request, nodeName string, shard string) bool {
	logger.Debugf("Try to redirect query to node %s of shard %s", nodeName, shard)
	var node *cluster.NodeInfo
	traversedNodes := []string{s.config.Node.Name}
	query := r.URL.Query()
	if nodes, ok := query[TRAVERSED_NODE_PARAMETER]; ok {
		traversedNodes = append(traversedNodes, nodes...)
	}
	nodeAlreadyTraversed := sort.SearchStrings(traversedNodes, nodeName) != len(traversedNodes)
	if !nodeAlreadyTraversed {
		logger.Debugf("Not not %s yed traversed, let's try to find info online", nodeName)
		// We didn't tried this node yet, let's see if it is online
		if nodeInfo, found := s.topologyClient.Nodes()[nodeName]; found {
			node = nodeInfo
		}
	}
	if node == nil {
		// We already tried this node or it is not online, let's look for another node in the same shard
		logger.Debugf("Look in shard %s for node not in %v", shard, traversedNodes)
		for _, nodeInfo := range s.topologyClient.Nodes() {
			if nodeInfo.Shard == shard && sort.SearchStrings(traversedNodes, nodeInfo.Name) == len(traversedNodes) {
				node = nodeInfo
				break
			}
		}
	}
	if node != nil {
		s.redirectToNode(w, r, node)
		return true
	}
	return false
}

func (s *Server) redirectToNode(w http.ResponseWriter, r *http.Request, node *cluster.NodeInfo) {
	uri := r.URL.String()
	uri = node.Address + uri[strings.Index(uri, FILES_PREFIX):]
	if strings.Index(uri, "?") == -1 {
		uri += "?"
	} else {
		uri += "&"
	}
	uri += TRAVERSED_NODE_PARAMETER + "=" + s.config.Node.Name

	logger.Debugf("Redirect to URL %s", uri)
	w.Header().Set(LOCATION, uri)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (s *Server) Close() {
	s.httpServer.Close()
	close(s.fileJobs)
}

func (s *Server) CloseAndDestroyStorage() error {
	s.Close()
	return s.storage.Destroy()
}

func returnError(err error, w http.ResponseWriter) {
	w.WriteHeader(errorToHttpStatus(err))
	w.Write([]byte(err.Error()))
}
