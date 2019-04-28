package http

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/macq/flocons/pkg/error"
	"github.com/macq/flocons/pkg/file"
)

const FILES_PREFIX string = "/files"

func errorToHttpStatus(err error) int {
	switch {
	case os.IsNotExist(err):
		return 404
	case os.IsPermission(err):
		return 503
	case os.IsExist(err):
		return 509
	default:
		return 500
	}
}

func getResponseBodyString(resp *http.Response) string {
	var body []byte
	if resp.ContentLength > 0 {
		body = make([]byte, resp.ContentLength)
		resp.Body.Read(body)
	}
	if body == nil {
		return ""
	}
	return string(body)
}

func responseToFileInfo(uri *url.URL, resp *http.Response) (os.FileInfo, error) {
	switch {
	case resp.StatusCode == 404:
		return nil, NewFileNotFoundError(path.Base(uri.Path))
	case resp.StatusCode == 500:
		return nil, NewInternalError(fmt.Sprintf("%s: %s", resp.Status, getResponseBodyString(resp)))
	case resp.StatusCode >= 300:
		return nil, NewHttpError(fmt.Sprintf("%s: %s", resp.Status, getResponseBodyString(resp)), resp.StatusCode)
	}

	h := resp.Header

	modified, err := strconv.ParseInt(h.Get(LAST_MODIFIED), 10, 64)
	if err != nil {
		modified = 0
	}

	return file.NewFileInfo(
		path.Base(uri.Path),
		headerToFileMode(h),
		resp.ContentLength,
		time.Unix(modified, 0),
		file.FileDataSource{},
	), nil
}

func headerToFileMode(h http.Header) os.FileMode {
	mimeType := h.Get(CONTENT_TYPE)
	if mimeType == "" {
		mimeType = file.DEFAULT_FILE_MIME_TYPE
	}

	parsedFileMode, err := strconv.ParseUint(h.Get(CONTENT_MODE), 8, 32)
	if err != nil {
		if mimeType == file.DIRECTORY_MIME_TYPE {
			parsedFileMode = 0755
		} else {
			parsedFileMode = 0644
		}
	}
	fileMode := (os.FileMode)(parsedFileMode)
	if mimeType == file.DIRECTORY_MIME_TYPE {
		fileMode |= os.ModeDir
	}
	return fileMode
}

func fileInfoToHeader(fi os.FileInfo, h http.Header) {
	mode := (uint32)(fi.Mode())
	// Remove all information about type of file in sent mode because it is OS dependant
	// We will add this information in Content-Type header
	mode &= ^((uint32)(os.ModeType))

	h.Set(CONTENT_MODE, strconv.FormatUint((uint64)(mode), 8))
	h.Set(LAST_MODIFIED, strconv.FormatInt(fi.ModTime().Unix(), 10))
	if fi.Mode().IsDir() {
		h.Set(CONTENT_TYPE, file.DIRECTORY_MIME_TYPE)
	} else {
		h.Set(CONTENT_TYPE, mime.TypeByExtension(filepath.Ext(fi.Name())))
	}
	h.Set(CONTENT_LENGTH, strconv.FormatInt(fi.Size(), 10))
}

func filesInfoToCsv(files []os.FileInfo) ([]byte, error) {
	output := bytes.Buffer{}
	writer := csv.NewWriter(&output)
	// Mask to remove all information about type of file in sent mode because it is OS dependant
	modeTypeSuppressMask := ^((uint32)(os.ModeType))
	for _, fi := range files {
		var fileTypeIdentifier string
		if fi.Mode().IsDir() {
			fileTypeIdentifier = "d"
		} else {
			fileTypeIdentifier = "-"
		}
		mode := (uint32)(fi.Mode()) & modeTypeSuppressMask
		err := writer.Write([]string{
			fileTypeIdentifier,
			fi.Name(),
			strconv.FormatUint((uint64)(mode), 8),
			strconv.FormatInt(fi.Size(), 10),
			strconv.FormatInt(fi.ModTime().Unix(), 10),
		})
		if err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return output.Bytes(), nil
}

func csvToFilesInfo(data []byte) ([]os.FileInfo, error) {
	output := make([]os.FileInfo, 0)
	input := bytes.NewReader(data)
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = 5
	reader.ReuseRecord = true
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		fileTypeIdentifier := record[0]
		name := record[1]
		mode, _ := strconv.ParseUint(record[2], 8, 32)
		if fileTypeIdentifier == "d" {
			mode |= (uint64)(os.ModeDir)
		}
		size, _ := strconv.ParseInt(record[3], 10, 64)
		modTime, _ := strconv.ParseInt(record[4], 10, 64)

		output = append(output, file.NewFileInfo(name,
			(os.FileMode)(mode), size, time.Unix(modTime, 0),
			file.FileDataSource{}))
	}
	return output, nil
}
