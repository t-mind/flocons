package test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/t-mind/flocons/cluster"
	"github.com/t-mind/flocons/config"
	"github.com/t-mind/flocons/http"
	"github.com/t-mind/flocons/storage"
	"github.com/t-mind/flocons/test/mock"
)

func createServerAndClient(t *testing.T, number int, zookeeper *mock.Zookeeper, trueDispatcher bool) (*http.Server, *http.Client, *storage.Storage) {
	directory, err := ioutil.TempDir(os.TempDir(), "flocons-test")
	if err != nil {
		panic(err)
	}

	json_config := fmt.Sprintf(`{"node": {"name": "node-%d", "port": %d, "external_address": "http://127.0.0.1:%d"}, "storage": {"path": %q}}`, number, 5555+number, 5555+number, directory)
	config, err := config.NewConfigFromJson([]byte(json_config))
	if err != nil {
		t.Errorf("Could not parse config %s: %s", json_config, err)
		t.FailNow()
	}

	storage, err := storage.NewStorage(config)
	if err != nil {
		t.Errorf("Could not instantiate storage: %s", err)
	}

	var dispatcher cluster.Dispatcher
	if !trueDispatcher {
		dispatcher = &mock.NullDispatcher{}
	} else {
		dispatcher, err = cluster.NewMaglevDispatcher()
		if err != nil {
			t.Errorf("Could not instantiate maglev dispatcher: %s", err)
		}
	}

	server, err := http.NewServer(config, storage, cluster.NewTopologyClientWithZookeperClientFactory(config, zookeeper.GetFactory(), dispatcher))
	if err != nil {
		t.Errorf("Could instantiate server: %s", err)
		storage.Destroy()
		t.FailNow()
	}

	client, err := http.NewClient(fmt.Sprintf("http://127.0.0.1:%d", 5555+number))
	if err != nil {
		t.Errorf("Could instantiate client: %s", err)
		server.CloseAndDestroyStorage()
		t.FailNow()
	}
	return server, client, storage
}

func TestDistributedIndex(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	mock := mock.NewZookeeper()
	server1, client1, storage1 := createServerAndClient(t, 1, mock, false)
	defer server1.CloseAndDestroyStorage()
	defer client1.Close()
	server2, client2, storage2 := createServerAndClient(t, 2, mock, false)
	defer server2.CloseAndDestroyStorage()
	defer client2.Close()

	testCreateDirectory(t, client2, "/dir")
	testCreateFile(t, client2, "/dir", "testFile", "testData")

	// Let's copy indexes only from storage2 to storage 1
	dir1 := storage1.MakeAbsolute("/dir")
	dir2 := storage2.MakeAbsolute("/dir")
	os.Mkdir(dir1, 0755)
	files, _ := filepath.Glob(filepath.Join(dir2, "index*"))
	for _, file := range files {
		origin, _ := os.Open(file)
		copy, _ := os.Create(filepath.Join(dir1, file[len(dir2):]))
		io.Copy(copy, origin)
		origin.Close()
		copy.Close()
	}
	testReadFile(t, client1, "/dir", "testFile", "testData")
}

func TestDistributedIndexAndBadContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	mock := mock.NewZookeeper()
	server1, client1, storage1 := createServerAndClient(t, 1, mock, false)
	defer server1.CloseAndDestroyStorage()
	defer client1.Close()
	server2, client2, storage2 := createServerAndClient(t, 2, mock, false)
	defer server2.CloseAndDestroyStorage()
	defer client2.Close()
	server3, client3, storage3 := createServerAndClient(t, 3, mock, false)
	defer server3.CloseAndDestroyStorage()
	defer client3.Close()
	server4, client4, storage4 := createServerAndClient(t, 4, mock, false)
	defer server4.CloseAndDestroyStorage()
	defer client4.Close()

	testCreateDirectory(t, client2, "/dir")
	testCreateFile(t, client2, "/dir", "testFile", "testData")

	// Let's copy indexes only from storage2 to storage 1 and storage 3
	dir1 := storage1.MakeAbsolute("/dir")
	dir2 := storage2.MakeAbsolute("/dir")
	dir3 := storage3.MakeAbsolute("/dir")
	dir4 := storage4.MakeAbsolute("/dir")
	os.Mkdir(dir1, 0755)
	os.Mkdir(dir3, 0755)
	os.Mkdir(dir4, 0755)
	files, _ := filepath.Glob(filepath.Join(dir2, "index*"))
	for _, file := range files {
		origin, _ := os.Open(file)
		copy, _ := os.Create(filepath.Join(dir1, file[len(dir2):]))
		io.Copy(copy, origin)
		copy.Close()

		origin.Seek(0, os.SEEK_SET)
		copy, _ = os.Create(filepath.Join(dir3, file[len(dir2):]))
		io.Copy(copy, origin)
		copy.Close()

		origin.Close()
	}
	// Move data from node 2 to node 4
	storage2.ResetCache()
	files, _ = filepath.Glob(filepath.Join(dir2, "files*"))
	for _, file := range files {
		err := os.Rename(file, filepath.Join(dir4, file[len(dir2):]))
		fmt.Println(err)
	}
	testReadFile(t, client1, "/dir", "testFile", "testData")
}

func TestSimpleDispatching(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	numFiles := 100
	munDir := 5
	mock := mock.NewZookeeper()
	server1, client1, _ := createServerAndClient(t, 1, mock, true)
	defer server1.CloseAndDestroyStorage()
	defer client1.Close()
	server2, client2, _ := createServerAndClient(t, 2, mock, true)
	defer server2.CloseAndDestroyStorage()
	defer client2.Close()

	testCreateDirectory(t, client1, "/dir")

	for i := 0; i < munDir; i++ {
		client := client1
		if i%2 == 0 {
			client = client2
		}
		dir := fmt.Sprintf("/dir/testDir%d", i)
		testCreateDirectory(t, client, dir)
	}

	for i := 0; i < numFiles; i++ {
		client := client1
		if i%2 == 0 {
			client = client2
		}
		dir := fmt.Sprintf("/dir/testDir%d", i%munDir)
		testCreateFile(t, client, dir, fmt.Sprintf("testFile%d", i), fmt.Sprintf("testData%d", i))
	}

	for i := 0; i < numFiles; i++ {
		client := client1
		if i%2 != 0 {
			client = client2
		}
		dir := fmt.Sprintf("/dir/testDir%d", i%munDir)
		testReadFile(t, client, dir, fmt.Sprintf("testFile%d", i), fmt.Sprintf("testData%d", i))
	}
}
