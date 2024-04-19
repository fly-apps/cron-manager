package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/fly-apps/cron-manager/internal/supervisor"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger := cron.SetupLogging()

	// Ensure required environment variables are set
	checkRequiredEnvs([]string{"FLY_API_TOKEN"})

	// Initialize the store
	store, err := cron.InitializeStore(cron.DefaultStorePath, cron.DefaultMigrationsPath)
	if err != nil {
		panic(fmt.Errorf("failed to create store: %w", err))
	}
	defer func() { _ = store.Close() }()

	if err := cron.SyncSchedules(store, logger, cron.DefaultSchedulesFilePath); err != nil {
		logger.Warnf("There was a problem syncing your schedules: %s", err)
	} else if err := cron.SyncCrontab(store, logger); err != nil {
		logger.Warnf("Failed to sync crontab: %s", err)
	}

	svisor := supervisor.New("cron-manager", 5*time.Minute)
	svisor.AddProcess("cron", "/usr/sbin/cron -f", supervisor.WithRestart(0, 5*time.Second))
	svisor.AddProcess("monitor", "/usr/local/bin/monitor", supervisor.WithRestart(0, 5*time.Second))
	svisor.AddProcess("api", "/usr/local/bin/api", supervisor.WithRestart(0, 5*time.Second))
	svisor.StopOnSignal(syscall.SIGINT, syscall.SIGTERM)

	if err := svisor.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func checkRequiredEnvs(vars []string) {
	for _, v := range vars {
		if _, exists := os.LookupEnv(v); !exists {
			panic(fmt.Errorf("%s is required", v))
		}
	}
}
