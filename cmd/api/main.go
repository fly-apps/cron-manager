package main

import (
	"fmt"
	"os"

	"github.com/fly-apps/cron-manager/api"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

func main() {
	var log = logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	// Allow overriding log level via env var
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			panic(fmt.Errorf("failed to parse log level: %w", err))
		}
		log.SetLevel(level)
	}

	if err := api.StartHttpServer(log); err != nil {
		panic(err)
	}
}
