package flocons

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	. "github.com/macq/flocons/error"
)

const DEFAULT_PORT int = 62116

type Config struct {
	Namespace string   `json:"namespace"`
	Zookeeper []string `json:"zookeeper"`
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

// Loads config from a Json file
func NewConfigFromFile(file string) (*Config, error) {
	logger.Infof("Open config file %s", file)
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return NewConfigFromJson(content)
}

// Create a config from a Json string
func NewConfigFromJson(content []byte) (*Config, error) {
	var config *Config
	err := json.Unmarshal(content, &config)
	if err != nil {
		return nil, NewConfigError(err.Error())
	}

	if err := sanitizeConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func sanitizeConfig(config *Config) error {
	ipv4RegExp, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	ipv6Regexp, _ := regexp.Compile(`([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`)
	hotnameRegexp, _ := regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

	if config.Namespace == "" {
		config.Namespace = "flocons"
	} else if !hotnameRegexp.MatchString(config.Namespace) {
		return NewConfigError(fmt.Sprintf("namespace %s is not valid", config.Namespace))
	}

	if len(config.Zookeeper) == 0 {
		config.Zookeeper = append(config.Zookeeper, "127.0.0.1:2181")
	} else {
		for _, address := range config.Zookeeper {
			scIndex := strings.LastIndex(address, ":")
			if scIndex > 0 {
				if _, err := strconv.ParseInt(address[scIndex+1:], 10, 16); err != nil {
					return NewConfigError(fmt.Sprintf("zookeeper address %s is not valid", address))
				}
				address = address[:scIndex]
			}
			if !hotnameRegexp.MatchString(address) && !ipv4RegExp.MatchString(address) && !ipv6Regexp.MatchString(address) {
				return NewConfigError(fmt.Sprintf("zookeeper address %s is not valid", address))
			}
		}
	}

	isNodeConfig := config.Node.Name != "" || config.Node.Port != 0 || config.Node.ExternalAddress != "" || config.Node.Shard != "" ||
		config.Storage.Path != ""

	if isNodeConfig {
		if config.Storage.Path == "" {
			return NewConfigError("node config without storage specified")
		}
		if config.Node.Port == 0 {
			config.Node.Port = DEFAULT_PORT
		}

		if config.Node.Name == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			logger.Debugf("No node name specified, take hostname %s", hostname)
			config.Node.Name = hostname
		} else if !hotnameRegexp.MatchString(config.Node.Name) {
			return NewConfigError("node name is invalid")
		}

		if config.Node.ExternalAddress == "" {
			config.Node.ExternalAddress = fmt.Sprintf("http://%s:%d", config.Node.Name, config.Node.Port)
		} else {
			if _, err := url.Parse(config.Node.ExternalAddress); err != nil {
				return NewConfigError("node external address is note a valid url")
			}
		}

		if config.Node.Shard == "" {
			config.Node.Shard = "shard-1"
		}

	}

	return nil
}
