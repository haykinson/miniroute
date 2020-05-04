package main

import (
	"log"
	"os"
)

var logger *log.Logger

func initLogging() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}
