package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/t-mind/flocons/config"
	. "github.com/t-mind/flocons/error"

	"github.com/golang/groupcache/lru"
)

const DIRECTORY_CACHE_SIZE int = 1000

type Storage struct {
	path             string
	config           *config.Config
	directoryCache   *lru.Cache
	updateCacheMutex *sync.Mutex
}

type DirectoryCacheEntry struct {
	writeContainer            *RegularFileContainer
	containers                map[string]*RegularFileContainer
	containersUpdateMutex     sync.Mutex
	writeContainerUpdateMutex sync.Mutex
}

type regularFileContainerWalker struct {
	storage    *Storage
	directory  string
	cacheEntry *DirectoryCacheEntry

	cacheKeys    []string
	currentIndex int
	files        []os.FileInfo
}

func NewStorage(config *config.Config) (*Storage, error) {
	if config.Storage.Path == "" {
		return nil, NewInternalError("Tried to initialize storage with no configured path")
	}
	if config.Node.Name == "" {
		return nil, NewInternalError("Tried to initialize storage with no configured node name")
	}
	s := Storage{
		path:             config.Storage.Path,
		config:           config,
		directoryCache:   lru.New(DIRECTORY_CACHE_SIZE),
		updateCacheMutex: &sync.Mutex{},
	}
	s.directoryCache.OnEvicted = func(key lru.Key, value interface{}) {
		cacheEntry, _ := value.(*DirectoryCacheEntry)
		cacheEntry.containersUpdateMutex.Lock()
		defer cacheEntry.containersUpdateMutex.Unlock()
		cacheEntry.writeContainerUpdateMutex.Lock()
		defer cacheEntry.writeContainerUpdateMutex.Unlock()
		if cacheEntry.writeContainer != nil {
			cacheEntry.writeContainer.Close()
		}
	}

	fi, err := os.Stat(s.path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, NewIsNotDirError(s.path)
	}
	testPath := s.MakeAbsolute("flocons-test")
	if file, err := os.Create(testPath); err == nil {
		file.Close()
	} else {
		return nil, os.ErrPermission
	}
	if os.Remove(testPath) != nil {
		return nil, os.ErrPermission
	}
	return &s, nil
}

func (s *Storage) MakeAbsolute(p string) string {
	if !strings.HasPrefix(p, s.path) {
		return filepath.Join(s.path, p)
	}
	return p
}

func (s *Storage) CreateDirectory(p string, mode os.FileMode) (os.FileInfo, error) {
	mode |= 0700 // be sure that we will whatever be able to interact with this directory
	fullPath := s.MakeAbsolute(p)
	logger.Debugf("create directory %s with mode %o\n", fullPath, mode)
	if err := os.Mkdir(fullPath, mode); err != nil {
		return nil, err
	}
	return os.Stat(fullPath)
}

func (s *Storage) CreateDirectoryAndParents(p string, mode os.FileMode) (os.FileInfo, error) {
	mode |= 0700 // be sure that we will whatever be able to interact with this directory
	fullPath := s.MakeAbsolute(p)
	logger.Debugf("create directory %s and parents with mode %o\n", fullPath, mode)
	if err := os.MkdirAll(fullPath, mode); err != nil {
		return nil, err
	}
	return os.Stat(fullPath)
}

func (s *Storage) GetDirectory(p string) (os.FileInfo, error) {
	fullPath := s.MakeAbsolute(p)

	fi, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, NewIsNotDirError(p)
	}
	return fi, nil
}

func (s *Storage) CreateRegularFile(p string, mode os.FileMode, data []byte) (os.FileInfo, error) {
	directory := filepath.Dir(p)
	if _, err := s.GetDirectory(directory); err != nil {
		return nil, err
	}
	cacheEntry := s.getDirectoryCacheEntry(directory)
	if err := s.ensureCacheEntryWriteContainer(directory, cacheEntry); err != nil {
		return nil, err
	}
	return (*cacheEntry.writeContainer).CreateRegularFile(filepath.Base(p), mode, data)
}

func (s *Storage) GetRegularFile(p string) (os.FileInfo, error) {
	directory := filepath.Dir(p)
	fullDirectory := s.MakeAbsolute(directory)
	fileName := filepath.Base(p)
	fi, err := os.Stat(fullDirectory)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, NewIsNotDirError(directory)
	}

	walker := newRegularFileContainerWalker(s, directory)
	for {
		container, err := walker.Next()
		if err != nil {
			return nil, err
		}
		if container == nil {
			return nil, NewFileNotFoundError(p)
		}
		if f, err := container.GetRegularFile(fileName); err == nil {
			return f, nil
		}
	}
}

func (s *Storage) GetFile(p string) (os.FileInfo, error) {
	d, derr := s.GetDirectory(p)
	if derr == nil {
		return d, nil
	}
	f, ferr := s.GetRegularFile(p)
	if ferr == nil {
		return f, nil
	}

	// If directory error was a permission problem, then the actual error is certainly this
	if os.IsPermission(derr) {
		return nil, derr
	}
	return nil, ferr
}

func (s *Storage) getDirectoryCacheEntry(directory string) *DirectoryCacheEntry {
	s.updateCacheMutex.Lock()
	defer s.updateCacheMutex.Unlock()

	var cacheEntry *DirectoryCacheEntry
	if rawEntry, found := s.directoryCache.Get(directory); found {
		cacheEntry, _ = rawEntry.(*DirectoryCacheEntry)
	} else {
		var nullContainer *RegularFileContainer
		cacheEntry = &DirectoryCacheEntry{
			containers:                make(map[string]*RegularFileContainer),
			writeContainer:            nullContainer,
			containersUpdateMutex:     sync.Mutex{},
			writeContainerUpdateMutex: sync.Mutex{},
		}
		s.directoryCache.Add(directory, cacheEntry)
	}
	return cacheEntry
}

// Ensure that we have a container open for writing
// If there is already one and it is not full, it returns this container
// If there is one but full, it closes and creates a new one
// If there is none yet, it tries to find the container with the highest number and opens it if not full (or corrupted)
// If none it found, it creates a new one
func (s *Storage) ensureCacheEntryWriteContainer(directory string, cacheEntry *DirectoryCacheEntry) error {
	cacheEntry.writeContainerUpdateMutex.Lock()
	defer cacheEntry.writeContainerUpdateMutex.Unlock()

	// First let's check if the actual container is not full
	if cacheEntry.writeContainer != nil && !cacheEntry.writeContainer.IsWriteable(s.config) {
		logger.Infof("Container %s is full -> close it\n", cacheEntry.writeContainer.Name)
		cacheEntry.writeContainer.Close()
		cacheEntry.writeContainer = nil
	}

	if cacheEntry.writeContainer == nil {
		logger.Debugf("No container opened to write in directory %s on node %s -> let's search for one\n", s.config.Storage.Path, s.config.Node.Name)
		var writeContainer *RegularFileContainer
		var maxNumber int = 0
		walker := newRegularFileContainerWalkerFromCacheEntry(s, directory, cacheEntry)
		for {
			container, err := walker.Next()
			if err != nil {
				return err
			}
			if container == nil {
				break
			}
			logger.Debugf("Walk through container %s\n", container.Name)
			if container.Node == s.config.Node.Name && container.Number > maxNumber {
				maxNumber = container.Number
			}
			if container.IsWriteable(s.config) && (writeContainer == nil || writeContainer.Number < container.Number) {
				logger.Debugf("Found one valid container %s\n", container.Name)
				writeContainer = container
			}
		}
		if writeContainer == nil {
			cacheEntry.containersUpdateMutex.Lock()
			defer cacheEntry.containersUpdateMutex.Unlock()
			name := NewRegularFileContainerName(s.config.Node.Shard, s.config.Node.Name, maxNumber+1)
			logger.Infof("No container available to write in directory %s on node %s -> let's create %s\n", s.config.Storage.Path, s.config.Node.Name, name)
			newWriteContainer, err := NewRegularFileContainer(s.MakeAbsolute(directory), name, s.config, nil)
			if err != nil {
				logger.Fatalf("Could not create new regular file container %s", err)
			}
			cacheEntry.containers[name] = newWriteContainer
			writeContainer = newWriteContainer
		}
		cacheEntry.writeContainer = writeContainer
	}
	return nil
}

func (s *Storage) ReadDir(directory string) ([]os.FileInfo, error) {
	fullpath := s.MakeAbsolute(directory)
	fi, err := os.Stat(fullpath)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, NewIsNotDirError(fullpath)
	}

	rawFiles, err := ioutil.ReadDir(fullpath)
	if err != nil {
		return nil, err
	}
	dirs := make([]os.FileInfo, 0)
	for _, f := range rawFiles {
		if f.IsDir() {
			dirs = append(dirs, f)
		}
	}

	files := make([]os.FileInfo, 0)
	walker := newRegularFileContainerWalker(s, directory)
	for {
		container, err := walker.Next()
		if err != nil {
			return nil, err
		}
		if container == nil {
			break
		}
		if fs, err := container.ListFiles(); err == nil {
			files = append(files, fs...)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	return append(dirs, files...), nil
}

func (s *Storage) ResetCache() {
	s.directoryCache.Clear()
}

func (s *Storage) Close() {
	s.ResetCache()
}

func (s *Storage) Destroy() error {
	s.Close()
	return os.RemoveAll(s.path)
}

func newRegularFileContainerWalker(s *Storage, directory string) *regularFileContainerWalker {
	return newRegularFileContainerWalkerFromCacheEntry(s, directory, s.getDirectoryCacheEntry(directory))
}

func newRegularFileContainerWalkerFromCacheEntry(s *Storage, directory string, entry *DirectoryCacheEntry) *regularFileContainerWalker {
	cacheKeys := make([]string, 0, len(entry.containers))
	for index, _ := range entry.containers {
		cacheKeys = append(cacheKeys, index)
	}
	return &regularFileContainerWalker{
		storage:      s,
		directory:    directory,
		cacheEntry:   entry,
		cacheKeys:    cacheKeys,
		currentIndex: -1,
	}
}

func (w *regularFileContainerWalker) Next() (*RegularFileContainer, error) {
	w.currentIndex++
	if w.currentIndex < len(w.cacheKeys) {
		return w.cacheEntry.containers[w.cacheKeys[w.currentIndex]], nil
	}

	fullpath := w.storage.MakeAbsolute(w.directory)
	if w.files == nil {
		files, err := ioutil.ReadDir(fullpath)
		if err != nil {
			return nil, err
		}
		w.files = files
	}

	// Be sure not to update the mutex with twice the same container
	w.cacheEntry.containersUpdateMutex.Lock()
	defer w.cacheEntry.containersUpdateMutex.Unlock()

	// Let's lookup the directory for not yet managed containers or lonely indexes
	containers := w.cacheEntry.containers
	var nextContainer *RegularFileContainer
	for ; w.currentIndex < len(w.cacheKeys)+len(w.files); w.currentIndex++ {
		file := w.files[w.currentIndex-len(w.cacheKeys)]
		if file.IsDir() {
			continue
		}
		name := file.Name()
		found := false
		// Lonely index ref if any
		var index *RegularFileContainerIndex
		if IsRegularFileContainer(name) {
			_, found = containers[name]
		} else if IsRegularFileContainerIndex(name) {
			for _, container := range containers {
				if container.index != nil && container.index.Name == name {
					found = true
					break
				}
			}
			if !found {
				// We found a lonely index
				index, err := NewRegularFileContainerIndex(fullpath, name, w.storage.config)
				if err != nil {
					logger.Errorln(err)
					continue
				}
				// Let's defined the name of the empty refular file container that will be created
				name = NewRegularFileContainerName(index.Shard, index.Node, index.Number)
			}
		}
		if !found {
			container, err := NewRegularFileContainer(fullpath, name, w.storage.config, index)
			if err != nil {
				logger.Errorln(err)
			} else {
				containers[name] = container
				if nextContainer == nil {
					nextContainer = container // still, let's continue to update the cache entry
				}
			}
		}
	}
	return nextContainer, nil
}
