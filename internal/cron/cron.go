package cron

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

	if err := SyncCrontab(store); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	// TODO - This should be supervised.
	if err := startDaemon(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	return nil
}

func SyncCrontab(store *Store) error {
	schedules, err := store.ListSchedules()
	if err != nil {
		return fmt.Errorf("failed to list schedules: %w", err)
	}

	file, err := os.Create(cronFilePath)
	if err != nil {
		return fmt.Errorf("failed to open crontab file: %w", err)
	}
	defer file.Close()

	for _, schedule := range schedules {
		entry := fmt.Sprintf("%s %s %d >> %s 2>&1\n", schedule.Schedule, executeCommand, schedule.ID, logFilePath)
		file.WriteString(entry)
	}

	if err := exec.Command("crontab", cronFilePath).Run(); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	log.Printf("[INFO] Synced %d schedule(s) to crontab\n", len(schedules))

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
