package cron

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

const (
	executeCommand = "/usr/local/bin/process-job"
	cronFilePath   = "/data/crontab"
)

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
		entry := fmt.Sprintf("%s %s %d\n", schedule.Schedule, executeCommand, schedule.ID)
		_, err := file.WriteString(entry)
		if err != nil {
			return fmt.Errorf("failed to write to crontab file: %w", err)
		}
	}

	if err := exec.Command("crontab", cronFilePath).Run(); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	log.Printf("Synced %d schedule(s) to crontab", len(schedules))

	return nil
}
