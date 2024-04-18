package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/fly-apps/cron-manager/internal/supervisor"
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

	if err := cron.SyncSchedules(store, log); err != nil {
		log.Warnf("failed to sync schedules: %s", err)
		log.Warnf("no schedule updates were made, please work to correct the issue re-deploy the application.")
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
