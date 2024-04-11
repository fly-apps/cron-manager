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
	cronFilePath   = "/data/cronjobs"
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
	cronjobs, err := store.ListCronJobs()
	if err != nil {
		return fmt.Errorf("failed to list cronjobs: %w", err)
	}

	if len(cronjobs) == 0 {
		log.Printf("[INFO] No cronjobs to sync\n")
		return nil
	}

	file, err := os.Create(cronFilePath)
	if err != nil {
		return fmt.Errorf("failed to open crontab file: %w", err)
	}

	defer file.Close()
	for _, cronjob := range cronjobs {
		entry := fmt.Sprintf("%s %s %d >> %s 2>&1\n", cronjob.Schedule, executeCommand, cronjob.ID, logFilePath)
		file.WriteString(entry)
	}

	if err := exec.Command("crontab", cronFilePath).Run(); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	log.Printf("[INFO] Synced %d cronjobs to crontab\n", len(cronjobs))

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
