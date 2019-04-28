package storage

import (
	"encoding/csv"

	"github.com/macq/flocons/pkg/file"
	"github.com/macq/flocons/pkg/flocons"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	. "github.com/macq/flocons/pkg/error"
)

var indexRegexp, _ = regexp.Compile("^index_(([^_]+)_v([0-9]+)_([0-9]+)).csv$")

func IsRegularFileContainerIndex(name string) bool {
	return indexRegexp.MatchString(name)
}

func NewRegularFileContainerIndexName(node string, number int) string {
	return fmt.Sprintf("index_%s_v1_%d.csv", node, number)
}

type RegularFileContainerIndex struct {
	Name       string
	Node       string
	Version    int
	Number     int
	path       string
	config     *flocons.Config
	entries    map[string]os.FileInfo
	lastSize   int64
	writeFd    *os.File
	writeMutex *sync.Mutex
}

func NewRegularFileContainerIndex(directory string, name string, config *flocons.Config) (*RegularFileContainerIndex, error) {
	fullpath := filepath.Join(directory, name)
	parts := indexRegexp.FindStringSubmatch(name)
	if parts == nil {
		return nil, NewInternalError("Tried to create a container index with name " + name + " which is invalid")
	}
	node := parts[2]
	version, _ := strconv.Atoi(parts[3])
	number, _ := strconv.Atoi(parts[4])
	index := RegularFileContainerIndex{
		Name:       name,
		Node:       node,
		Version:    version,
		Number:     number,
		path:       fullpath,
		config:     config,
		entries:    make(map[string]os.FileInfo),
		writeMutex: &sync.Mutex{},
	}
	_, err := os.Stat(fullpath)
	if err != nil {
		// We didn't find the index file or we have a permission problem
		if !os.IsNotExist(err) || node != config.Node.Name {
			return nil, err
		}
		// It can be normal if we dind't find the file and we are on the same node, let's create it
		f, err := os.Create(fullpath)
		if err != nil {
			return nil, err
		}
		f.Close()
	}

	err = index.updateEntries()
	if err != nil {
		return nil, err
	}
	return &index, nil
}

func FindRegularFileContainerIndex(directory string, node string, number int, config *flocons.Config) (*RegularFileContainerIndex, error) {
	pattern := fmt.Sprintf("%s_%s_v*_%d.csv", filepath.Join(directory, "index"), node, number)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, NewFileNotFoundError(pattern)
	}
	return NewRegularFileContainerIndex(directory, filepath.Base(matches[0]), config)
}

func (i *RegularFileContainerIndex) GetRegularFile(name string) (os.FileInfo, error) {
	entry, found := i.entries[name]
	if !found {
		err := i.updateEntries()
		if err != nil {
			return nil, err
		}
		entry, found = i.entries[name]
	}
	if found {
		return entry, nil
	}
	return nil, NewFileNotFoundError(name)
}

func (i *RegularFileContainerIndex) AddRegularFile(f os.FileInfo) error {
	storageFileInfo, ok := f.(*file.FileInfo)
	if !ok {
		return NewInternalError("Tried to add a file to the index with wrong file info type")
	}

	i.writeMutex.Lock()
	defer i.writeMutex.Unlock()

	if i.writeFd == nil {
		f, err := os.OpenFile(i.path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		if _, err = f.Seek(0, os.SEEK_END); err != nil {
			return err
		}
		i.writeFd = f
	}

	writer := csv.NewWriter(i.writeFd)
	err := writer.Write([]string{
		storageFileInfo.Name(),
		strconv.FormatInt(storageFileInfo.Address(), 10),
		strconv.FormatUint((uint64)(storageFileInfo.Mode()), 8),
		strconv.FormatInt(storageFileInfo.Size(), 10),
		strconv.FormatInt(storageFileInfo.ModTime().Unix(), 10),
	})
	if err != nil {
		return err
	}
	writer.Flush()
	i.lastSize, _ = i.writeFd.Seek(0, os.SEEK_CUR)
	i.entries[storageFileInfo.Name()] = storageFileInfo
	return i.writeFd.Sync()
}

func (i *RegularFileContainerIndex) ListFiles() ([]os.FileInfo, error) {
	if err := i.updateEntries(); err != nil {
		fmt.Println(err)
	}
	files := make([]os.FileInfo, 0, len(i.entries))
	for _, file := range i.entries {
		files = append(files, file)
	}
	return files, nil
}

func (i *RegularFileContainerIndex) updateEntries() error {
	fi, err := os.Stat(i.path)
	if err != nil {
		return err
	}
	if fi.Size() > i.lastSize {
		f, err := os.Open(i.path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.Seek(i.lastSize, os.SEEK_SET); err != nil {
			return err
		}

		reader := csv.NewReader(f)
		reader.FieldsPerRecord = 5
		reader.ReuseRecord = true
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err == nil {
				name := record[0]
				address, _ := strconv.ParseInt(record[1], 10, 64)
				mode, _ := strconv.ParseUint(record[2], 8, 32)
				size, _ := strconv.ParseInt(record[3], 10, 64)
				modTime, _ := strconv.ParseInt(record[4], 10, 64)

				i.entries[name] = file.NewFileInfo(name,
					(os.FileMode)(mode), size, time.Unix(modTime, 0),
					file.FileDataSource{
						Node:    i.Node,
						Address: address,
					})
			}
		}

		i.lastSize = fi.Size()
	}
	return nil
}

func (i *RegularFileContainerIndex) Close() {
	if i.writeFd != nil {
		i.writeFd.Close()
		i.writeFd = nil
	}
}
