package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/macq/flocons/pkg/flocons"
	"github.com/macq/flocons/pkg/http"
)

func initServer(t *testing.T) *http.Server {
	directory, err := ioutil.TempDir(os.TempDir(), "flocons-test")
	if err != nil {
		panic(err)
	}

	json_config := fmt.Sprintf("{\"node\": {\"name\": \"node-%d\", \"port\": 5555}, \"storage\": {\"path\": %q}}", 0, directory)
	config, err := flocons.NewConfigFromJson([]byte(json_config))
	if err != nil {
		t.Errorf("Could not parse config %s: %s", json_config, err)
		t.FailNow()
	}

	server, err := http.NewServer(config)
	if err != nil {
		t.Errorf("Could instantiate server: %s", err)
		t.FailNow()
	}
	return server
}

func initClient(t *testing.T) *http.Client {
	client, err := http.NewClient("http://127.0.0.1:5555")
	if err != nil {
		t.Errorf("Could instantiate client: %s", err)
		t.FailNow()
	}
	return client
}

func TestReadWrites(t *testing.T) {
	server := initServer(t)
	defer server.CloseAndDestroyStorage()

	client := initClient(t)
	defer client.Close()

	TestCreateDirectory(t, client, "/testDir")
	TestGetDirectory(t, client, "/testDir")
	TestCreateFile(t, client, "/testDir", "testFile", "testData")
	TestReadFile(t, client, "/testDir", "testFile", "testData")
}

func TestLs(t *testing.T) {
	server := initServer(t)
	defer server.CloseAndDestroyStorage()

	client := initClient(t)
	defer client.Close()

	TestReadDir(t, client)
}

func TestConcurrentClients(t *testing.T) {
	server := initServer(t)
	defer server.CloseAndDestroyStorage()

	wg := sync.WaitGroup{}

	initialClient := initClient(t)
	defer initialClient.Close()
	TestCreateDirectory(t, initialClient, "/testDir")

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := initClient(t)
			defer client.Close()
			fileName := fmt.Sprintf("testFile-%d", id)
			data := fmt.Sprintf("testData-%d", id)
			TestCreateFile(t, client, "/testDir", fileName, data)
			TestReadFile(t, client, "/testDir", fileName, data)
		}(i)
	}
	wg.Wait()
}
