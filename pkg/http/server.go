package http

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	. "github.com/macq/flocons/pkg/error"
	"github.com/macq/flocons/pkg/file"
	"github.com/macq/flocons/pkg/flocons"
	"github.com/macq/flocons/pkg/storage"
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
	httpHandler.HandleFunc(FILES_PREFIX+"/", func(w http.ResponseWriter, r *http.Request) {
		mutex := sync.Mutex{}
		barrier := sync.NewCond(&mutex)
		mutex.Lock()
		server.fileJobs <- serverJob{writer: w, request: r, barrier: barrier}
		barrier.Wait()
		mutex.Unlock()
	})
	httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Served uri " + r.URL.String())
		w.WriteHeader(400)
	})

	for i := 0; i < FILE_WORKER_POOL_SIZE; i++ {
		go func() {
			server.waitForFileWork()
		}()
	}

	go func() {
		if httpServer.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	return &server, nil
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

func returnError(err error, w http.ResponseWriter) {
	w.WriteHeader(errorToHttpStatus(err))
	w.Write([]byte(err.Error()))
}
