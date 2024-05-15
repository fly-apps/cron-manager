package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/fly-apps/cron-manager/internal/supervisor"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger := cron.SetupLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure required environment variables are set
	if err := checkRequiredEnvs([]string{"FLY_API_TOKEN"}); err != nil {
		logger.Fatal(err)
	}

	// Initialize the store
	store, err := cron.InitializeStore(ctx, cron.DefaultStorePath, cron.DefaultMigrationsPath)
	if err != nil {
		logger.Fatal(fmt.Errorf("failed to create store: %w", err))
	}

	defer func() {
		if err := store.Close(); err != nil {
			logger.WithError(err).Error("failed to close store")
		}
	}()

	if err := cron.SyncSchedules(ctx, store, logger, cron.DefaultSchedulesFilePath); err != nil {
		logger.Warnf("There was a problem syncing your schedules: %s", err)
	} else if err := cron.SyncCrontab(ctx, store, logger); err != nil {
		logger.Warnf("Failed to sync crontab: %s", err)
	}

	svisor := supervisor.New("cron-manager", 5*time.Minute)
	svisor.AddProcess("cron", "/usr/sbin/cron -f", supervisor.WithRestart(0, 5*time.Second))
	svisor.AddProcess("monitor", "/usr/local/bin/monitor", supervisor.WithRestart(0, 5*time.Second))
	svisor.AddProcess("api", "/usr/local/bin/api", supervisor.WithRestart(0, 5*time.Second))
	svisor.StopOnSignal(syscall.SIGINT, syscall.SIGTERM)

	// Handle graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	if err := svisor.Run(); err != nil {
		logger.WithError(err).Error("supervisor exited with error")
		os.Exit(1)
	}
}

func checkRequiredEnvs(vars []string) error {
	for _, v := range vars {
		if _, exists := os.LookupEnv(v); !exists {
			return fmt.Errorf("%s is required", v)
		}
	}

	return nil
}
