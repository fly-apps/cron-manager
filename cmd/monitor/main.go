package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3"
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

	ctx := context.Background()

	store, err := cron.NewStore(cron.StorePath)
	if err != nil {
		panic(fmt.Errorf("failed to create store: %w", err))
	}
	defer store.Close()

	if err := cron.MonitorActiveJobs(ctx, store, log); err != nil {
		panic(err)
	}
}
