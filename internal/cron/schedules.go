package cron

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	schedulesFilePath     = "/usr/local/share/schedules.json"
	defaultCommandTimeout = 30
)

func SyncSchedules(store *Store, log *logrus.Logger) error {
	// Read schedules from file
	schedules, err := readSchedulesFromFile(schedulesFilePath)
	if err != nil {
		return err
	}

	existingSchedules, err := store.ListSchedules()
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
			if err := store.CreateSchedule(schedule); err != nil {
				return fmt.Errorf("failed to create schedule: %w", err)
			}

			log.Infof("Created schedule %s", schedule.Name)

			continue
		}

		// If schedule exists, update it
		if err := store.UpdateSchedule(schedule); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}

		log.Infof("Updated schedule %s", schedule.Name)
		presentSchedules[schedule.Name] = 1
	}

	// Delete schedules that are no longer present
	for _, schedule := range existingSchedules {
		if _, exists := presentSchedules[schedule.Name]; !exists {
			if err := store.DeleteSchedule(fmt.Sprint(schedule.ID)); err != nil {
				return fmt.Errorf("failed to delete schedule: %w", err)
			}
			log.Infof("deleted schedule %s", schedule.Name)
		}
	}

	if err := syncCrontab(store, log); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	return nil
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

	// If the file is empty, return an empty slice
	// This is expected behavior on initial launch, or if all schedules are being deleted
	if len(schedulesBytes) == 0 {
		return []Schedule{}, nil
	}

	var schedules []Schedule
	if err := json.Unmarshal(schedulesBytes, &schedules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedules: %w", err)
	}

	return schedules, nil
}
