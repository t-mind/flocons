package storage

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/macq/flocons"
	"github.com/macq/flocons/file"
	. "github.com/macq/flocons/error"
)

var containerRegexp, _ = regexp.Compile("^files_(([^_]+)_v([0-9]+)_([0-9]+)).tar$")

func IsRegularFileContainer(name string) bool {
	return containerRegexp.MatchString(name)
}

func NewRegularFileContainerName(node string, number int) string {
	return fmt.Sprintf("files_%s_v1_%d.tar", node, number)
}

type RegularFileContainer struct {
	Name       string
	Node       string
	Version    int
	Number     int
	path       string
	config     *flocons.Config
	writeFd    *os.File
	tarWriter  *tar.Writer
	writeMutex *sync.Mutex
	index      *RegularFileContainerIndex
}

func NewRegularFileContainer(directory string, name string, config *flocons.Config, index *RegularFileContainerIndex) (*RegularFileContainer, error) {
	fullpath := filepath.Join(directory, name)
	parts := containerRegexp.FindStringSubmatch(name)
	if parts == nil {
		return nil, NewInternalError("Tried to create a container with name " + name + " which is invalid")
	}
	node := parts[2]
	version, _ := strconv.Atoi(parts[3])
	number, _ := strconv.Atoi(parts[4])

	_, err := os.Stat(fullpath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var index_err error
	if index == nil {
		index, index_err = FindRegularFileContainerIndex(directory, node, number, config)
		if index_err != nil && !os.IsNotExist(index_err) {
			return nil, index_err
		}
	}

	if err != nil {
		if index_err == nil {
			// We didn't find the container file but well the index file
			if config.Node.Name == node {
				// This can be normal only if the file comes from another node
				return nil, err
			}
		} else {
			// We didn't fint the container file neither the index file
			if config.Node.Name == node {
				// This can be normal only if the file is from this node
				f, err := os.Create(fullpath)
				if err != nil {
					return nil, err
				}
				f.Close()
				index, index_err = NewRegularFileContainerIndex(directory, NewRegularFileContainerIndexName(node, number), config)
				if index_err != nil {
					return nil, err
				}
			}
		}
	}

	return &RegularFileContainer{
		Name:       name,
		Node:       node,
		Version:    version,
		Number:     number,
		path:       fullpath,
		config:     config,
		writeMutex: &sync.Mutex{},
		index:      index,
	}, nil
}

func (c *RegularFileContainer) GetRegularFile(name string) (os.FileInfo, error) {
	var fi os.FileInfo
	var err error
	if c.index != nil {
		fi, err = c.index.GetRegularFile(name)
		if err != nil {
			return nil, err
		}
	} else {
		f, err := os.OpenFile(c.path, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		reader := tar.NewReader(f)
		var address int64

		for {
			h, err := reader.Next()
			if h == nil {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Name == name {
				fi = h.FileInfo()
				break
			}
			// Let's compute address for next header
			address, _ = f.Seek(0, os.SEEK_CUR)
			address += h.Size
			// in tar, blocks are rounded to 512
			mod512 := address % 512
			if mod512 > 0 {
				address += 512 - mod512
			}
		}
		if fi != nil {
			storageFileInfo := file.FileInfoFromFileInfo(fi, file.FileDataSource{Address: address})
			fi = storageFileInfo
		}
	}
	if fi == nil {
		return nil, NewFileNotFoundError(name)
	}

	storageFileInfo, _ := fi.(*file.FileInfo)
	storageFileInfo.UpdateDataSource(file.FileDataSource{
		Container: c.Name,
		Node:      c.Node,
		Data: func() ([]byte, error) {
			return c.GetRegularFileData(storageFileInfo)
		},
	})
	return storageFileInfo, nil
}

func (c *RegularFileContainer) GetRegularFileData(fi os.FileInfo) ([]byte, error) {
	f, err := os.OpenFile(c.path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := tar.NewReader(f)

	if storageFileInfo, ok := fi.(*file.FileInfo); ok {
		if storageFileInfo.Container() != c.Name {
			return nil, NewInternalError(fmt.Sprintf("Asked for file data in wrong container (%s != %s)", storageFileInfo.Container(), c.Name))
		}
		f.Seek(storageFileInfo.Address(), os.SEEK_SET)
		_, err := reader.Next()
		if err != nil {
			return nil, err
		}

	} else {
		for {
			h, err := reader.Next()
			if h == nil {
				return nil, NewFileNotFoundError(fi.Name())
			}
			if err != nil {
				return nil, err
			}
			if h.Name == fi.Name() {
				break
			}
		}
	}
	buffer := make([]byte, fi.Size())
	_, err = reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buffer, nil
}

func (c *RegularFileContainer) CreateRegularFile(name string, mode os.FileMode, data []byte) (os.FileInfo, error) {
	if c.config.Node.Name != c.Node {
		return nil, NewInternalError("Tried to write file in container of another node " + c.Name)
	}

	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	if c.writeFd == nil {
		f, err := os.OpenFile(c.path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		if _, err = f.Seek(0, os.SEEK_END); err != nil {
			return nil, err
		}

		c.writeFd = f
		c.tarWriter = tar.NewWriter(c.writeFd)
	}

	address, err := c.writeFd.Seek(0, os.SEEK_CUR)
	if err != nil {
		c.tarWriter.Close()
		c.writeFd.Close()
		c.writeFd = nil
		return nil, err
	}

	header := tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     (int64)(len(data)),
		Mode:     (int64)(mode),
		ModTime:  time.Now(),
	}
	if err := c.tarWriter.WriteHeader(&header); err != nil {
		return nil, err
	}
	if _, err := c.tarWriter.Write(data); err != nil {
		return nil, err
	}
	if err := c.tarWriter.Flush(); err != nil {
		return nil, err
	}

	fi := file.FileInfoFromFileInfo(header.FileInfo(), file.FileDataSource{Address: address})

	if c.index != nil {
		if err = c.index.AddRegularFile(fi); err != nil {
			return nil, err
		}
	}

	return fi, nil
}

func (c *RegularFileContainer) ListFiles() ([]os.FileInfo, error) {
	if c.index != nil {
		return c.index.ListFiles()
	}

	f, err := os.OpenFile(c.path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := tar.NewReader(f)
	files := make([]os.FileInfo, 0, 100)
	for {
		h, err := reader.Next()
		if h == nil {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, h.FileInfo())
	}
	return files, nil
}

func (c *RegularFileContainer) Close() {
	if c.writeFd != nil {
		// Never ever close the writer because it will add closing data at the end of tar, terminating the archive
		// c.tarWriter.Close()
		c.writeFd.Close()
		c.writeFd = nil
	}
	if c.index != nil {
		c.index.Close()
	}
}
