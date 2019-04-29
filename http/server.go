package http

import (
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/macq/flocons"
	. "github.com/macq/flocons/error"
	"github.com/macq/flocons/file"
	"github.com/macq/flocons/storage"
)

const FILE_WORKER_POOL_SIZE int = 10

type Server struct {
	config     *flocons.Config
	storage    *storage.Storage
	httpServer *http.Server
	fileJobs   chan serverJob
}

type serverJob struct {
	writer  http.ResponseWriter
	request *http.Request
	barrier *sync.Cond
}

func NewServer(config *flocons.Config) (*Server, error) {
	if config.Node.Port == 0 {
		return nil, NewInternalError("No port configured")
	}
	storage, err := storage.NewStorage(config)
	if err != nil {
		return nil, err
	}

	httpHandler := http.NewServeMux() // Don't use default handler because we would want several servers in parralel for tests
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(config.Node.Port), Handler: httpHandler}
	server := Server{
		config:     config,
		storage:    storage,
		httpServer: httpServer,
		fileJobs:   make(chan serverJob),
	}
	server.start()
	return &server, nil
}

func (s *Server) start() {
	httpHandler, _ := s.httpServer.Handler.(*http.ServeMux)
	httpHandler.HandleFunc(FILES_PREFIX+"/", func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("Handle file request %s on ressource %s\n", r.Method, r.URL.Path)
		mutex := sync.Mutex{}
		barrier := sync.NewCond(&mutex)
		mutex.Lock()
		s.fileJobs <- serverJob{writer: w, request: r, barrier: barrier}
		barrier.Wait()
		mutex.Unlock()
	})
	httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Warnf("Unhandled URL request %s\n", r.URL.Path)
		w.WriteHeader(400)
	})

	logger.Info("Start workers")
	for i := 0; i < FILE_WORKER_POOL_SIZE; i++ {
		go func() {
			s.waitForFileWork()
		}()
	}

	go func() {
		logger.Info("Start http server")
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Panicf("Http server failed %s\n", err)
		}
	}()
}

func (s *Server) waitForFileWork() {
	for {
		select {
		case job, ok := <-s.fileJobs:
			if !ok {
				return // channel closed
			}
			job.barrier.L.Lock()
			s.ServeFile(job.writer, job.request)
			job.barrier.Broadcast()
			job.barrier.L.Unlock()
		}
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
		returnError(err, w)
		return
	}
	fileInfoToHeader(fi, w.Header())
}

func (s *Server) GetFileWithData(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[len(FILES_PREFIX):]
	fi, err := s.storage.GetFile(p)
	if err != nil {
		returnError(err, w)
		return
	}
	var data []byte
	if fi.Mode().IsRegular() {
		storageFileInfo, _ := fi.(*file.FileInfo)
		data, err = storageFileInfo.Data()
		if err != nil {
			returnError(err, w)
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
