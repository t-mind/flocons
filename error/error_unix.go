// +build !windows

package error

import (
	"os"
	"syscall"
)

func NewFileNotFoundError(path string) error {
	return &os.PathError{Op: "stat", Path: path, Err: syscall.ENOENT}
}
