package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	. "github.com/macq/flocons/pkg/error"
	"github.com/macq/flocons/pkg/flocons"

	"github.com/golang/groupcache/lru"
)

const DIRECTORY_CACHE_SIZE int = 1000

type Storage struct {
	path             string
	config           *flocons.Config
	directoryCache   *lru.Cache
	updateCacheMutex *sync.Mutex
}

type DirectoryCacheEntry struct {
	writeContainer *RegularFileContainer
	containers     map[string]*RegularFileContainer
	updateMutex    sync.Mutex
}

type regularFileContainerWalker struct {
	storage    *Storage
	directory  string
	cacheEntry *DirectoryCacheEntry

	cacheKeys    []string
	currentIndex int
	files        []os.FileInfo
}

func NewStorage(config *flocons.Config) (*Storage, error) {
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
		cacheEntry.updateMutex.Lock()
		defer cacheEntry.updateMutex.Unlock()
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
	if !filepath.IsAbs(p) {
		return filepath.Join(s.path, p)
	}
	return p
}

func (s *Storage) CreateDirectory(p string, mode os.FileMode) (os.FileInfo, error) {
	mode |= 0700 // be sure that we will whatever be able to interact with this directory
	fullPath := s.MakeAbsolute(p)
	fmt.Printf("create directory %s with mode %o\n", fullPath, mode)
	if err := os.Mkdir(fullPath, mode); err != nil {
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
	s.ensureCacheEntryWriteContainer(directory, cacheEntry)
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
			containers:     make(map[string]*RegularFileContainer),
			writeContainer: nullContainer,
			updateMutex:    sync.Mutex{},
		}
		s.directoryCache.Add(directory, cacheEntry)
	}
	return cacheEntry
}

func (s *Storage) ensureCacheEntryWriteContainer(directory string, cacheEntry *DirectoryCacheEntry) {
	cacheEntry.updateMutex.Lock()
	defer cacheEntry.updateMutex.Unlock()
	if cacheEntry.writeContainer == nil {
		var writeContainer *RegularFileContainer
		for _, container := range cacheEntry.containers {
			if container.Node == s.config.Node.Name && (writeContainer == nil || writeContainer.Number < container.Number) {
				writeContainer = container
			}
		}
		if writeContainer == nil {
			name := NewRegularFileContainerName(s.config.Node.Name, 1)
			writeContainer, _ = NewRegularFileContainer(s.MakeAbsolute(directory), name, s.config, nil)
			cacheEntry.containers[name] = writeContainer
			cacheEntry.writeContainer = writeContainer
		}
	}
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

func (s *Storage) Close() {
	s.directoryCache.Clear()
}

func (s *Storage) Destroy() error {
	s.Close()
	return os.RemoveAll(s.path)
}

func newRegularFileContainerWalker(s *Storage, directory string) *regularFileContainerWalker {
	entry := s.getDirectoryCacheEntry(directory)
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
	w.cacheEntry.updateMutex.Lock()
	defer w.cacheEntry.updateMutex.Unlock()

	// Let's lookup the directory for not yet managed containers or lonely indexes
	containers := w.cacheEntry.containers
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
					fmt.Println(err)
					continue
				}
				// Let's defined the name of the empty refular file container that will be created
				name = NewRegularFileContainerName(index.Node, index.Number)
			}
		}
		if !found {
			container, err := NewRegularFileContainer(fullpath, name, w.storage.config, index)
			if err != nil {
				fmt.Println(err)
			} else {
				containers[name] = container
				return container, nil
			}
		}
	}
	return nil, nil
}
