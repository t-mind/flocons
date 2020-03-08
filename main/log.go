package main

import (
	log "github.com/sirupsen/logrus"
)

var logger *log.Entry = log.WithFields(log.Fields{
	"package": "main",
})
