package file

import (
	"os"
	"time"

	. "github.com/t-mind/flocons/error"
)

type FileDataSource struct {
	Node      string
	Shard     string
	Container string
	Address   int64
	Data      func() ([]byte, error)
}

type FileInfo struct {
	name    string
	mode    os.FileMode
	size    int64
	modTime time.Time
	sys     FileDataSource
}

func NewFileInfo(name string, mode os.FileMode, size int64, modTime time.Time, sys FileDataSource) *FileInfo {
	return &FileInfo{
		name:    name,
		mode:    mode,
		size:    size,
		modTime: modTime,
		sys:     sys,
	}
}

func FileInfoFromFileInfo(fi os.FileInfo, dataSource FileDataSource) *FileInfo {
	return &FileInfo{
		name:    fi.Name(),
		mode:    fi.Mode(),
		size:    fi.Size(),
		modTime: fi.ModTime(),
		sys:     dataSource,
	}
}

func (i *FileInfo) AttachDataSource(s FileDataSource) {
	i.sys = s
}

func (i *FileInfo) UpdateDataSource(s FileDataSource) {
	if s.Node != "" {
		i.sys.Node = s.Node
	}
	if s.Shard != "" {
		i.sys.Shard = s.Shard
	}
	if s.Container != "" {
		i.sys.Container = s.Container
	}
	if s.Address != 0 {
		i.sys.Address = s.Address
	}
	if s.Data != nil {
		i.sys.Data = s.Data
	}
}

// base name of the file
func (i *FileInfo) Name() string {
	return i.name
}

// length in bytes for regular files; system-dependent for others
func (i *FileInfo) Size() int64 {
	return i.size
}

// file mode bits
func (i *FileInfo) Mode() os.FileMode {
	return i.mode
}

func (i *FileInfo) ModTime() time.Time {
	return i.modTime
}

// abbreviation for Mode().IsDir()
func (i *FileInfo) IsDir() bool {
	return i.mode.IsDir()
}

// underlying data source (can return nil)
func (i *FileInfo) Sys() interface{} {
	return i.sys
}

func (i *FileInfo) Address() int64 {
	return i.sys.Address
}

func (i *FileInfo) Node() string {
	return i.sys.Node
}

func (i *FileInfo) Shard() string {
	return i.sys.Shard
}

func (i *FileInfo) Container() string {
	return i.sys.Container
}

func (i *FileInfo) IsDataAvailable() bool {
	return i.sys.Data != nil
}

func (i *FileInfo) Data() ([]byte, error) {
	if i.sys.Data == nil {
		return nil, NewInternalError("Data method not implemented")
	}
	return i.sys.Data()
}
