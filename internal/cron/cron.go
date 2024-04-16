package cron

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

const (
	executeCommand = "/usr/local/bin/process-job"
	logFilePath    = "/data/cron.log"
	cronFilePath   = "/data/crontab"
)

func InitializeCron(store *Store) error {
	if err := initializeLogFile(); err != nil {
		return fmt.Errorf("failed to initialize log file: %w", err)
	}

	if err := startDaemon(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	return nil
}

func syncCrontab(store *Store, log *logrus.Logger) error {
	schedules, err := store.ListEnabledSchedules()
	if err != nil {
		return fmt.Errorf("failed to list schedules: %w", err)
	}

	file, err := os.Create(cronFilePath)
	if err != nil {
		return fmt.Errorf("failed to open crontab file: %w", err)
	}
	defer func() { _ = file.Close() }()

	for _, schedule := range schedules {
		entry := fmt.Sprintf("%s %s %d >> %s 2>&1\n", schedule.Schedule, executeCommand, schedule.ID, logFilePath)
		_, err := file.WriteString(entry)
		if err != nil {
			return fmt.Errorf("failed to write to crontab file: %w", err)
		}
	}

	if err := exec.Command("crontab", cronFilePath).Run(); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	log.Printf("synced %d schedule(s) to crontab", len(schedules))

	return nil
}

func startDaemon() error {
	return exec.Command("service", "cron", "start").Run()
}

func initializeLogFile() error {
	if _, err := os.Stat(logFilePath); err != nil {
		if os.IsNotExist(err) {
			if _, err := os.Create(logFilePath); err != nil {
				return err
			}
		}
	}

	return nil
}
