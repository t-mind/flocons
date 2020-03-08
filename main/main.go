package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"

	log "github.com/sirupsen/logrus"

	. "github.com/t-mind/flocons/cluster"
	. "github.com/t-mind/flocons/config"
	. "github.com/t-mind/flocons/http"
	. "github.com/t-mind/flocons/storage"
)

func main() {
	log.SetLevel(log.DebugLevel)
	var configFile string
	flag.StringVar(&configFile, "config", "", "Configuration file")

	flag.Parse()

	config, err := NewConfigFromFile(configFile)
	if err != nil {
		logger.Fatal(err)
	}

	storage, err := NewStorage(config)
	if err != nil {
		logger.Fatal(err)
	}

	server, _ := NewServer(config, storage, NewSingleTopologyNodeClient(config))
	if err != nil {
		logger.Fatal(err)
	}
	waitForInterruption()
	logger.Info("Received interruption")
	server.Close()
}

func waitForInterruption() {
	var end_waiter sync.WaitGroup
	end_waiter.Add(1)
	var signal_channel chan os.Signal
	signal_channel = make(chan os.Signal, 1)
	signal.Notify(signal_channel, os.Interrupt)
	go func() {
		<-signal_channel
		end_waiter.Done()
	}()
	end_waiter.Wait()
}
