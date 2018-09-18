package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

var logger = log.New()

func main() {

	if err := mainCmd.Execute(); err != nil {
		os.Exit(1)
	}

}
