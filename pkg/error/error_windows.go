// +build windows

package error

import (
	"os"
	"syscall"
)

func NewFileNotFoundError(path string) error {
	return &os.PathError{Op: "stat", Path: path, Err: syscall.ERROR_FILE_NOT_FOUND}
}
