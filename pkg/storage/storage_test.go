package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/macq/flocons/pkg/flocons"
	"github.com/macq/flocons/pkg/test"
)

func initStorages(t *testing.T, count int) []*Storage {
	directory, err := ioutil.TempDir(os.TempDir(), "flocons-test")
	if err != nil {
		panic(err)
	}

	ss := make([]*Storage, count)
	for i := 0; i < count; i++ {
		json_config := fmt.Sprintf("{\"node\": {\"name\": \"node-%d\"}, \"storage\": {\"path\": %q}}", i, directory)
		config, err := flocons.NewConfigFromJson([]byte(json_config))
		if err != nil {
			t.Errorf("Could not parse config %s: %s", json_config, err)
			t.FailNow()
		}

		storage, err := NewStorage(config)
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
	test.TestCreateDirectory(t, storage, "/testDir")
	test.TestGetDirectory(t, storage, "/testDir")
	test.TestCreateDirectory(t, storage, "/testDir/testSubdir")
	test.TestGetDirectory(t, storage, "/testDir/testSubdir")
}

func TestRegularFile(t *testing.T) {
	storage := initStorages(t, 1)[0]
	defer storage.Destroy()

	testDir := "/testDir"
	test.TestCreateDirectory(t, storage, testDir)

	before := time.Now().Truncate(1e9) // Take now truncated at second
	test.TestCreateFile(t, storage, testDir, "testFile", "testData")
	f := test.TestReadFile(t, storage, testDir, "testFile", "testData")

	after := time.Now()
	if f.ModTime().Before(before) || f.ModTime().After(after) {
		t.Errorf("Modification time %s is not between %s and %s", f.ModTime().String(), before.String(), after.String())
	}

	test.TestCreateFile(t, storage, testDir, "testFile1", "testData1")
	test.TestCreateFile(t, storage, testDir, "testFile2", "testData2")
	test.TestCreateFile(t, storage, testDir, "testFile3", "testData3")
	test.TestReadFile(t, storage, testDir, "testFile3", "testData3")
	test.TestReadFile(t, storage, testDir, "testFile2", "testData2")
	test.TestReadFile(t, storage, testDir, "testFile1", "testData1")
}

func TestLs(t *testing.T) {
	storage := initStorages(t, 1)[0]
	defer storage.Destroy()

	test.TestReadDir(t, storage)
}
func TestConcurrentBasic(t *testing.T) {
	ss := initStorages(t, 2)
	defer ss[0].Destroy()
	defer ss[1].Close()

	testDir := "/testDir"

	test.TestCreateDirectory(t, ss[0], testDir)
	test.TestCreateFile(t, ss[1], testDir, "testFile1", "testData1")
	test.TestReadFile(t, ss[0], testDir, "testFile1", "testData1")
	test.TestCreateFile(t, ss[0], testDir, "testFile2", "testData2")
	test.TestCreateFile(t, ss[0], testDir, "testFile3", "testData3")
	test.TestReadFile(t, ss[1], testDir, "testFile3", "testData3")
	test.TestReadFile(t, ss[1], testDir, "testFile2", "testData2")
	test.TestCreateFile(t, ss[1], testDir, "testFile4", "testData4")
	test.TestReadFile(t, ss[0], testDir, "testFile4", "testData4")
}

func TestMissingIndexes(t *testing.T) {
	ss := initStorages(t, 2)
	defer ss[0].Destroy()

	testDir := "/testDir"
	test.TestCreateDirectory(t, ss[1], testDir)
	test.TestCreateFile(t, ss[1], testDir, "testFile1", "testData1")
	test.TestCreateFile(t, ss[1], testDir, "testFile2", "testData2")
	test.TestCreateFile(t, ss[1], testDir, "testFile3", "testData3")
	ss[1].Close()

	dirPath := path.Join(ss[0].path, testDir)
	files, _ := filepath.Glob(filepath.Join(dirPath, "index*"))
	for _, f := range files {
		os.Remove(f)
	}

	test.TestReadFile(t, ss[0], testDir, "testFile3", "testData3")
	test.TestReadFile(t, ss[0], testDir, "testFile2", "testData2")
	test.TestReadFile(t, ss[0], testDir, "testFile1", "testData1")
}

func TestMissingContainer(t *testing.T) {
	ss := initStorages(t, 2)
	defer ss[0].Destroy()

	testDir := "/testDir"
	test.TestCreateDirectory(t, ss[1], testDir)
	test.TestCreateFile(t, ss[1], testDir, "testFile1", "testData1")
	test.TestCreateFile(t, ss[1], testDir, "testFile2", "testData2")
	test.TestCreateFile(t, ss[1], testDir, "testFile3", "testData3")
	ss[1].Close()

	dirPath := path.Join(ss[0].path, testDir)
	files, _ := filepath.Glob(filepath.Join(dirPath, "files*"))
	for _, f := range files {
		os.Remove(f)
	}

	test.TestReadFile(t, ss[0], testDir, "testFile3", "")
	test.TestReadFile(t, ss[0], testDir, "testFile2", "")
	test.TestReadFile(t, ss[0], testDir, "testFile1", "")

	files, _ = filepath.Glob(filepath.Join(dirPath, "files*"))
	if len(files) > 0 {
		t.Errorf("Container files should not have been created")
	}
}
