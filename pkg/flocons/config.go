package flocons

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Namespace string `json:"namespace"`
	Zookeeper string `json:"zookeeper"`
	Node      struct {
		Name            string `json:"name"`
		Port            int    `json:"port"`
		ExternalAddress string `json:"external_address"`
		Shard           string `json:"shard"`
	} `json:"node"`
	Storage struct {
		Path    string `json:"path"`
		MaxSize string `json:"max_size"`
	} `json:"storage"`
	Sync struct {
		DataTimeout     string `json:"data_timeout"`
		MetadataTimeout string `json:"metadata_timeout"`
	} `json:"sync"`
}

func NewConfigFromFile(file string) (*Config, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return NewConfigFromJson(content)
}

func NewConfigFromJson(content []byte) (*Config, error) {
	var config *Config
	err := json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
