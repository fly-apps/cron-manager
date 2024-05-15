package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

const (
	DefaultSchedulesFilePath = "/usr/local/share/schedules.json"
	executeCommand           = "/usr/local/bin/process-job"
	cronFilePath             = "/data/crontab"
	defaultCommandTimeout    = 30
)

// SyncSchedules reads schedules from a file and syncs them with the store
func SyncSchedules(ctx context.Context, store *Store, log *logrus.Logger, schedulesFilePath string) error {
	if schedulesFilePath == "" {
		schedulesFilePath = DefaultSchedulesFilePath
	}

	schedules, err := readSchedulesFromFile(schedulesFilePath)
	if err != nil {
		return err
	}

	existingSchedules, err := store.ListSchedules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list schedules: %w", err)
	}

	// Track present schedules so we know which ones to delete
	presentSchedules := make(map[string]int)

	for _, schedule := range schedules {

		// Set command timeout to default if not provided
		if schedule.CommandTimeout == 0 {
			schedule.CommandTimeout = defaultCommandTimeout
		}

		record := findScheduleByName(existingSchedules, schedule.Name)
		if record == nil {
			if err := store.CreateSchedule(ctx, schedule); err != nil {
				return fmt.Errorf("failed to create schedule: %w", err)
			}

			log.Infof("Created schedule %s", schedule.Name)
			continue
		}

		// If schedule exists, update it
		if err := store.UpdateSchedule(ctx, schedule); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}

		log.Infof("Updated schedule %s", schedule.Name)
		presentSchedules[schedule.Name] = 1
	}

	// Delete schedules that are no longer present
	for _, schedule := range existingSchedules {
		if _, exists := presentSchedules[schedule.Name]; !exists {
			if err := store.DeleteSchedule(ctx, fmt.Sprint(schedule.ID)); err != nil {
				return fmt.Errorf("failed to delete schedule: %w", err)
			}
			log.Infof("deleted schedule %s", schedule.Name)
		}
	}

	return nil
}

// SyncCrontab queries the store for enabled schedules and writes them to the crontab file
func SyncCrontab(ctx context.Context, store *Store, log *logrus.Logger) error {
	schedules, err := store.ListEnabledSchedules(ctx)
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

	return file.Sync()
}

func findScheduleByName(schedules []Schedule, name string) *Schedule {
	for _, schedule := range schedules {
		if schedule.Name == name {
			return &schedule
		}
	}
	return nil
}

func readSchedulesFromFile(schedulesFilePath string) ([]Schedule, error) {
	schedulesBytes, err := os.ReadFile(schedulesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open schedules file: %w", err)
	}

	// If the file is empty, return an empty slice.
	// This is expected behavior on initial launch, or in the event all schedules are being deleted.
	if len(schedulesBytes) == 0 {
		return []Schedule{}, nil
	}

	var schedules []Schedule
	if err := json.Unmarshal(schedulesBytes, &schedules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedules: %w", err)
	}

	return schedules, nil
}
