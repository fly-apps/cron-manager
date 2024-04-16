package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fly-apps/cron-manager/api"
	"github.com/fly-apps/cron-manager/internal/cron"
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

	ctx := context.Background()

	requiredPasswords := []string{"FLY_API_TOKEN"}
	for _, str := range requiredPasswords {
		if _, exists := os.LookupEnv(str); !exists {
			panic(fmt.Errorf("%s is required", str))
		}
	}

	store, err := cron.NewStore(cron.StorePath)
	if err != nil {
		panic(fmt.Errorf("failed to create store: %w", err))
	}

	if err := store.SetupDB(log, cron.MigrationsPath); err != nil {
		panic(fmt.Errorf("failed to setup db: %w", err))
	}

	if err := cron.InitializeCron(store); err != nil {
		panic(fmt.Errorf("failed to sync crontab: %w", err))
	}

	if err := cron.Reconcile(ctx, store, log); err != nil {
		panic(fmt.Errorf("failed to reconcile jobs: %w", err))
	}

	if err := cron.SyncSchedules(store, log); err != nil {
		log.Warnf("failed to sync schedules: %s", err)
		log.Warnf("no schedule updates were made, please work to correct the issue re-deploy the application.")
	}

	if err := api.StartHttpServer(log); err != nil {
		panic(fmt.Errorf("failed to start http server: %w", err))
	}
}
