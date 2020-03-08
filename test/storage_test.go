package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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
