package http

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/macq/flocons/pkg/flocons"
	"github.com/macq/flocons/pkg/test"
)

func initServer(t *testing.T) *Server {
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

	server, err := NewServer(config)
	if err != nil {
		t.Errorf("Could instantiate server: %s", err)
		t.FailNow()
	}
	return server
}

func initClient(t *testing.T) *Client {
	client, err := NewClient("http://127.0.0.1:5555")
	if err != nil {
		t.Errorf("Could instantiate client: %s", err)
		t.FailNow()
	}
	return client
}

func TestReadWrites(t *testing.T) {
	server := initServer(t)
	defer server.storage.Destroy()
	defer server.Close()

	client := initClient(t)
	defer client.Close()

	test.TestCreateDirectory(t, client, "/testDir")
	test.TestGetDirectory(t, client, "/testDir")
	test.TestCreateFile(t, client, "/testDir", "testFile", "testData")
	test.TestReadFile(t, client, "/testDir", "testFile", "testData")
}

func TestLs(t *testing.T) {
	server := initServer(t)
	defer server.storage.Destroy()
	defer server.Close()

	client := initClient(t)
	defer client.Close()

	test.TestReadDir(t, client)
}

func TestConcurrentClients(t *testing.T) {
	server := initServer(t)
	defer server.storage.Destroy()
	defer server.Close()

	wg := sync.WaitGroup{}

	initialClient := initClient(t)
	defer initialClient.Close()
	test.TestCreateDirectory(t, initialClient, "/testDir")

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			client := initClient(t)
			defer client.Close()
			defer wg.Done()
			fileName := fmt.Sprintf("testFile-%d", id)
			data := fmt.Sprintf("testData-%d", id)
			test.TestCreateFile(t, client, "/testDir", fileName, data)
			test.TestReadFile(t, client, "/testDir", fileName, data)
		}(i)
	}
	wg.Wait()
}
