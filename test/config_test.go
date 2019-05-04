package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/macq/flocons/config"
)

func TestEmptyConfig(t *testing.T) {
	config, err := config.NewConfigFromJson([]byte(`{}`))
	if err != nil {
		t.Errorf("Could not parse config %s", err)
	} else {
		if config.Namespace != "flocons" {
			t.Errorf("Namespace %s is different than expected flocons", config.Namespace)
		}
		if len(config.Zookeeper) != 1 || config.Zookeeper[0] != "127.0.0.1:2181" {
			t.Errorf("Zookeeper config %v is different than expected [127.0.0.1:2181]", config.Zookeeper)
		}
	}
}

func TestSimpleConfig(t *testing.T) {
	config, err := config.NewConfigFromJson([]byte(`{"namespace": "test"}`))
	if err != nil {
		t.Errorf("Could not parse config %s", err)
	} else {
		if config.Namespace != "test" {
			t.Errorf("Namespace %s is different than expected test", config.Namespace)
		}
	}
}

func TestSimpleNodeConfig(t *testing.T) {
	config, err := config.NewConfigFromJson([]byte(`{"node": {"port": 5555}, "storage": {"path": "/tmp"}}`))
	if err != nil {
		t.Errorf("Could not parse config %s", err)
	} else {
		if config.Node.Port != 5555 {
			t.Errorf("Port %d is different than expected %d", config.Node.Port, 5555)
		}
		hostname, _ := os.Hostname()
		if config.Node.Name != hostname {
			t.Errorf("Hostname %s is different than expected %s", config.Node.Name, hostname)
		}
		externalAddress := fmt.Sprintf("http://%s:%d", hostname, 5555)
		if config.Node.ExternalAddress != externalAddress {
			t.Errorf("ExternalAddress %s is different than expected %s", config.Node.ExternalAddress, externalAddress)
		}
		if config.Node.Shard != "shard-1" {
			t.Errorf("Shard name %s is different than expected %s", config.Node.Shard, "shard-1")
		}
		if config.Storage.Path != "/tmp" {
			t.Errorf("Storage path %s is different than expected %s", config.Storage.Path, "/tmp")
		}
	}
}

func TestBadConfig(t *testing.T) {
	if config, err := config.NewConfigFromJson([]byte(`{"namespace": "@name"}`)); err == nil {
		t.Errorf("Config %v should have failed", config)
	}
	if config, err := config.NewConfigFromJson([]byte(`{"zookeeper": "localhost:2181"}`)); err == nil {
		t.Errorf("Config %v should have failed", config)
	}
	if config, err := config.NewConfigFromJson([]byte(`{"zookeeper": ["localhost:999999"]}`)); err == nil {
		t.Errorf("Config %v should have failed", config)
	}
	if config, err := config.NewConfigFromJson([]byte(`{"zookeeper": ["127.0.0.1", "@@:2181]}`)); err == nil {
		t.Errorf("Config %v should have failed", config)
	}
	if config, err := config.NewConfigFromJson([]byte(`{"node": {"port": 5555}}`)); err == nil {
		t.Errorf("Config %v should have failed", config)
	}
}
