package http

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"

	. "github.com/macq/flocons/error"
	"github.com/macq/flocons/file"
)

type Client struct {
	host       string
	httpClient *http.Client
}

func NewClient(host string) (*Client, error) {
	if _, err := url.Parse(host); err != nil {
		return nil, err
	}
	httpClient := http.Client{}
	return &Client{
		host:       host,
		httpClient: &httpClient,
	}, nil
}

func (c *Client) pathToURL(p string) *url.URL {
	uri, _ := url.Parse(c.host + path.Join(FILES_PREFIX, filepath.ToSlash(p)))
	return uri
}

func (c *Client) CreateDirectory(p string, mode os.FileMode) (os.FileInfo, error) {
	uri := c.pathToURL(p)

	req, err := http.NewRequest("POST", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(CONTENT_TYPE, file.DIRECTORY_MIME_TYPE)
	req.Header.Set(CONTENT_MODE, strconv.FormatUint((uint64)(mode), 8))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return responseToFileInfo(uri, resp)
}

func (c *Client) CreateRegularFile(p string, mode os.FileMode, data []byte) (os.FileInfo, error) {
	uri := c.pathToURL(p)

	req, err := http.NewRequest("POST", uri.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set(CONTENT_MODE, strconv.FormatUint((uint64)(mode), 8))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return responseToFileInfo(uri, resp)
}

func (c *Client) GetFile(p string) (os.FileInfo, error) {
	uri := c.pathToURL(p)

	req, err := http.NewRequest("HEAD", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return responseToFileInfo(uri, resp)
}

func (c *Client) GetFileData(p string) (os.FileInfo, []byte, error) {
	uri := c.pathToURL(p)

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	fi, err := responseToFileInfo(uri, resp)
	if err != nil {
		return nil, nil, err
	}
	size := resp.ContentLength
	buffer := make([]byte, size)
	if size > 0 {
		_, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
	}
	return fi, buffer, nil
}

func (c *Client) GetDirectory(p string) (os.FileInfo, error) {
	fi, err := c.GetFile(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsDir() {
		return nil, NewIsNotDirError(p)
	}
	return fi, nil
}

func (c *Client) GetRegularFile(p string) (os.FileInfo, error) {
	fi, err := c.GetFile(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, NewIsDirError(p)
	}
	dataFileInfo, _ := fi.(*file.FileInfo)
	dataFileInfo.UpdateDataSource(file.FileDataSource{
		Data: func() ([]byte, error) {
			return c.GetRegularFileData(p)
		},
	})
	return dataFileInfo, nil
}

func (c *Client) GetRegularFileData(p string) ([]byte, error) {
	fi, data, err := c.GetFileData(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, NewIsDirError(p)
	}
	return data, nil
}

func (c *Client) GetRegularFileWithData(p string) (os.FileInfo, error) {
	fi, data, err := c.GetFileData(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, NewIsDirError(p)
	}
	dataFileInfo, _ := fi.(*file.FileInfo)
	dataFileInfo.UpdateDataSource(file.FileDataSource{
		Data: func() ([]byte, error) {
			return data, nil
		},
	})
	return dataFileInfo, nil
}

func (c *Client) ReadDir(p string) ([]os.FileInfo, error) {
	fi, data, err := c.GetFileData(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsDir() {
		return nil, NewIsNotDirError(p)
	}
	return csvToFilesInfo(data)
}
