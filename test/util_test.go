package test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/t-mind/flocons/file"
)

type FileService interface {
	CreateDirectory(string, os.FileMode) (os.FileInfo, error)
	GetDirectory(string) (os.FileInfo, error)
	CreateRegularFile(string, os.FileMode, []byte) (os.FileInfo, error)
	GetRegularFile(string) (os.FileInfo, error)
	ReadDir(string) ([]os.FileInfo, error)
}

func testCreateDirectory(t *testing.T, service FileService, dir string) {
	testName := filepath.Base(dir)
	fmt.Printf("Create temp directory %s for tests\n", testName)
	f, err := service.CreateDirectory(dir, 0755)
	if err != nil {
		t.Errorf("Could not create directory %s, %s", dir, err)
		t.FailNow()
	}
	if f.Name() != testName {
		t.Errorf("Path %s is different than expected %s", f.Name(), testName)
	}
	if !f.Mode().IsDir() {
		t.Errorf("Directory %s is not a directory", f.Name())
	}
	if runtime.GOOS != "windows" && f.Mode()&os.ModePerm != 0755 {
		t.Errorf("Mode %o is different than expected %o", f.Mode()&os.ModePerm, 0755)
	}
}

func testGetDirectory(t *testing.T, service FileService, dir string) {
	testName := filepath.Base(dir)
	f, err := service.GetDirectory(dir)
	if err != nil {
		t.Errorf("Could not get directory %s, %s", dir, err)
		t.FailNow()
	}
	if f.Name() != testName {
		t.Errorf("Path %s is different than expected %s", f.Name(), dir)
	}
	if !f.Mode().IsDir() {
		t.Errorf("Directory %s is not a directory", f.Name())
		t.FailNow()
	}
	if runtime.GOOS != "windows" && f.Mode()&os.ModePerm != 0755 {
		t.Errorf("Mode %o is different than expected %o", f.Mode()&os.ModePerm, 0755)
	}
}

func testCreateFile(t *testing.T, service FileService, dir string, name string, data string) {
	testCreateFileWithBytes(t, service, dir, name, []byte(data))
}

func testCreateFileWithBytes(t *testing.T, service FileService, dir string, name string, data []byte) {
	f, err := service.CreateRegularFile(filepath.Join(dir, name), 0644, data)
	if err != nil {
		t.Errorf("Could not create file %s: %s", name, err)
		t.FailNow()
	}
	if !f.Mode().IsRegular() {
		t.Errorf("File %s is not a regular file", f.Name())
	}
}

func testReadFile(t *testing.T, service FileService, dir string, name string, testData string) os.FileInfo {
	return testReadFileWithBytes(t, service, dir, name, []byte(testData))
}

func testReadFileWithBytes(t *testing.T, service FileService, dir string, name string, testData []byte) os.FileInfo {
	f, err := service.GetRegularFile(filepath.Join(dir, name))
	if err != nil {
		t.Errorf("Could get back file %s: %s", name, err)
		t.FailNow()
		return nil
	}
	if !f.Mode().IsRegular() {
		t.Errorf("File %s is not a regular file", f.Name())
	}

	if sf, ok := f.(*file.FileInfo); ok {
		if len(testData) != 0 { // empty string is used when we don't want to test data
			data, err := sf.Data()
			if err != nil {
				t.Errorf("Could not get data: %s", err)
			} else if bytes.Compare(testData, data) != 0 {
				t.Errorf("Data value does not match: %s != %s", data, testData)
			}
		}
	} else {
		t.Error("File info is not of type file.FileInfo")
	}
	return f
}

func testReadDir(t *testing.T, service FileService) {
	testDir := "/testDir"
	service.CreateDirectory(testDir, 0755)

	testSubDirs := []string{"testSubDir-1", "testSubDir-2", "testSubDir-3"}
	for _, dir := range testSubDirs {
		service.CreateDirectory(filepath.Join(testDir, dir), 0755)
	}
	testFiles := []string{"file-1", "file-2", "file-3"}
	testData := "testData"
	for _, file := range testFiles {
		service.CreateRegularFile(filepath.Join(testDir, file), 0644, []byte(testData))
	}
	files, err := service.ReadDir(testDir)
	if err != nil {
		t.Errorf("Could not read files from directory %s: %s", testDir, err)
		t.FailNow()
	}
	if len(files) != len(testFiles)+len(testSubDirs) {
		t.Errorf("Number of files read is different than exepcted %d != %d", len(files), len(testFiles)+len(testSubDirs))
		t.FailNow()
	}
	for index, dir := range testSubDirs {
		if dir != files[index].Name() {
			t.Errorf("Dir name is different than expected %s != %s", files[index].Name(), dir)
		}
	}
	for index, file := range testFiles {
		if file != files[index+len(testSubDirs)].Name() {
			t.Errorf("File name is different than expected %s != %s", files[index].Name(), file)
		}
	}
}
