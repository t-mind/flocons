package test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/docker/go-units"
	log "github.com/sirupsen/logrus"

	"github.com/t-mind/flocons/config"
	"github.com/t-mind/flocons/storage"
)

func initStorages(t *testing.T, count int) []*storage.Storage {
	directory, err := ioutil.TempDir(os.TempDir(), "flocons-test")
	if err != nil {
		panic(err)
	}

	ss := make([]*storage.Storage, count)
	for i := 0; i < count; i++ {
		json_config := fmt.Sprintf(`{"node": {"name": "node-%d"}, "storage": {"path": %q}}`, i, directory)
		config, err := config.NewConfigFromJson([]byte(json_config))
		if err != nil {
			t.Errorf("Could not parse config %s: %s", json_config, err)
			t.FailNow()
		}

		storage, err := storage.NewStorage(config)
		if err != nil {
			t.Errorf("Could not mount storage on %s: %s", directory, err)
			t.FailNow()
		}
		ss[i] = storage
	}
	return ss
}

func TestDirectory(t *testing.T) {
	storage := initStorages(t, 1)[0]
	defer storage.Destroy()

	f, err := storage.GetDirectory("/testDir")
	if err == nil {
		t.Errorf("Found directory %s which should not exist", f.Name())
	}
	testCreateDirectory(t, storage, "/testDir")
	testGetDirectory(t, storage, "/testDir")
	testCreateDirectory(t, storage, "/testDir/testSubdir")
	testGetDirectory(t, storage, "/testDir/testSubdir")
}

func TestStorageRegularFile(t *testing.T) {
	storage := initStorages(t, 1)[0]
	defer storage.Destroy()

	testDir := "/testDir"
	testCreateDirectory(t, storage, testDir)

	before := time.Now().Truncate(1e9) // Take now truncated at second
	testCreateFile(t, storage, testDir, "testFile", "testData")
	f := testReadFile(t, storage, testDir, "testFile", "testData")

	after := time.Now()
	if f.ModTime().Before(before) || f.ModTime().After(after) {
		t.Errorf("Modification time %s is not between %s and %s", f.ModTime().String(), before.String(), after.String())
	}

	testCreateFile(t, storage, testDir, "testFile1", "testData1")
	testCreateFile(t, storage, testDir, "testFile2", "testData2")
	testCreateFile(t, storage, testDir, "testFile3", "testData3")
	testReadFile(t, storage, testDir, "testFile3", "testData3")
	testReadFile(t, storage, testDir, "testFile2", "testData2")
	testReadFile(t, storage, testDir, "testFile1", "testData1")
}

func TestStorageLs(t *testing.T) {
	storage := initStorages(t, 1)[0]
	defer storage.Destroy()

	testReadDir(t, storage)
}
func TestStorageConcurrentBasic(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ss := initStorages(t, 2)
	defer ss[0].Destroy()
	defer ss[1].Close()

	testDir := "/testDir"

	testCreateDirectory(t, ss[0], testDir)
	testCreateFile(t, ss[1], testDir, "testFile1", "testData1")
	testReadFile(t, ss[0], testDir, "testFile1", "testData1")
	testCreateFile(t, ss[0], testDir, "testFile2", "testData2")
	testCreateFile(t, ss[0], testDir, "testFile3", "testData3")
	testReadFile(t, ss[1], testDir, "testFile3", "testData3")
	testReadFile(t, ss[1], testDir, "testFile2", "testData2")
	testCreateFile(t, ss[1], testDir, "testFile4", "testData4")
	testReadFile(t, ss[0], testDir, "testFile4", "testData4")
}

func TestExceedContainerCapacity(t *testing.T) {
	ss := initStorages(t, 1)
	s := ss[0]
	defer s.Destroy()

	testDir := "/testDir"
	fileSize, _ := FromHumanSize("10MB")
	content := make([]byte, fileSize)
	rand.Read(content)

	testCreateDirectory(t, s, testDir)
	for i := 0; i < 55; i++ {
		testCreateFileWithBytes(t, s, testDir, fmt.Sprintf("testFile%d", i), content)
	}
	for i := 0; i < 55; i++ {
		testReadFileWithBytes(t, s, testDir, fmt.Sprintf("testFile%d", i), content)
	}
	dirPath := s.MakeAbsolute(testDir)
	indexFiles, _ := filepath.Glob(filepath.Join(dirPath, "index*"))
	containerFiles, _ := filepath.Glob(filepath.Join(dirPath, "*.tar"))

	expectedNumber := 6
	if len(indexFiles) != expectedNumber {
		t.Errorf("Expected %d index created, found only %d", expectedNumber, len(indexFiles))
		t.FailNow()
	}
	if len(containerFiles) != expectedNumber {
		t.Errorf("Expected %d containers created, found only %d", expectedNumber, len(containerFiles))
		t.FailNow()
	}
}

func TestCloseAndOpenStorage(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ss := initStorages(t, 1)
	s := ss[0]
	defer s.Destroy()

	testDir := "/testDir"
	fileSize, _ := FromHumanSize("10MB")
	content := make([]byte, fileSize)
	rand.Read(content)

	testCreateDirectory(t, s, testDir)
	dirPath := s.MakeAbsolute(testDir)

	testCreateFileWithBytes(t, s, testDir, "testFile", content)

	s.Close()

	containerFiles, _ := filepath.Glob(filepath.Join(dirPath, "*.tar"))

	testCreateFileWithBytes(t, s, testDir, "testFile1", content)

	testReadFileWithBytes(t, s, testDir, "testFile", content)
	testReadFileWithBytes(t, s, testDir, "testFile1", content)

	containerFiles2, _ := filepath.Glob(filepath.Join(dirPath, "*.tar"))
	if len(containerFiles) != len(containerFiles2) {
		t.Errorf("New container created when closing and reopening file ! %d containers -> %d containers", len(containerFiles), len(containerFiles2))
	}

	for i := 0; i < 10; i++ {
		testCreateFileWithBytes(t, s, testDir, fmt.Sprintf("testFile-%d", i), content)
	}

	containerFiles3, _ := filepath.Glob(filepath.Join(dirPath, "*.tar"))
	var lastContainerInfo os.FileInfo
	var lastContainerFileName string
	for _, file := range containerFiles3 {
		if file != containerFiles[0] {
			lastContainerInfo, _ = os.Stat(file)
			lastContainerFileName = file
		}
	}
	fmt.Printf("LAST CONTAINER SIZE %d\n", lastContainerInfo.Size())

	s.Close()

	testCreateFileWithBytes(t, s, testDir, "testFile2", content)
	containerFiles4, _ := filepath.Glob(filepath.Join(dirPath, "*.tar"))
	if len(containerFiles3) != len(containerFiles4) {
		t.Errorf("New container created when closing and reopening file ! %d containers -> %d containers", len(containerFiles3), len(containerFiles4))
	}

	newLastContainerInfo, _ := os.Stat(lastContainerFileName)
	if lastContainerInfo.Size() == newLastContainerInfo.Size() {
		t.Errorf("File has been happened to the wrong container. Container size %d == %d", lastContainerInfo.Size(), newLastContainerInfo.Size())
	}
}

func TestStorageMissingIndexes(t *testing.T) {
	ss := initStorages(t, 2)
	defer ss[0].Destroy()

	testDir := "/testDir"
	testCreateDirectory(t, ss[1], testDir)
	testCreateFile(t, ss[1], testDir, "testFile1", "testData1")
	testCreateFile(t, ss[1], testDir, "testFile2", "testData2")
	testCreateFile(t, ss[1], testDir, "testFile3", "testData3")
	ss[1].Close()

	dirPath := ss[0].MakeAbsolute(testDir)
	files, _ := filepath.Glob(filepath.Join(dirPath, "index*"))
	for _, f := range files {
		os.Remove(f)
	}

	testReadFile(t, ss[0], testDir, "testFile3", "testData3")
	testReadFile(t, ss[0], testDir, "testFile2", "testData2")
	testReadFile(t, ss[0], testDir, "testFile1", "testData1")
}

func TestMissingContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ss := initStorages(t, 2)
	defer ss[0].Destroy()

	testDir := "/testDir"
	testCreateDirectory(t, ss[1], testDir)
	testCreateFile(t, ss[1], testDir, "testFile1", "testData1")
	testCreateFile(t, ss[1], testDir, "testFile2", "testData2")
	testCreateFile(t, ss[1], testDir, "testFile3", "testData3")
	ss[1].Close()

	dirPath := ss[0].MakeAbsolute(testDir)
	files, _ := filepath.Glob(filepath.Join(dirPath, "files*"))
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			t.Errorf("Could not remove files %s: %s", f, err)
		}
	}

	testReadFile(t, ss[0], testDir, "testFile3", "")
	testReadFile(t, ss[0], testDir, "testFile2", "")
	testReadFile(t, ss[0], testDir, "testFile1", "")

	files, _ = filepath.Glob(filepath.Join(dirPath, "files*"))
	if len(files) > 0 {
		t.Errorf("Container files should not have been created")
	}
}
