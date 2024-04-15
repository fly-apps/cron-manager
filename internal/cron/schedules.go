package cron

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	schedulesFilePath = "/usr/local/share/schedules.json"
)

func SyncSchedules(store *Store, log *logrus.Logger) error {
	schedulesBytes, err := os.ReadFile(schedulesFilePath)
	if err != nil {
		return fmt.Errorf("failed to open schedules file: %w", err)
	}

	var schedules []Schedule
	if err := json.Unmarshal(schedulesBytes, &schedules); err != nil {
		return fmt.Errorf("failed to unmarshal schedules: %w", err)
	}

	// Query existing schedules
	existingSchedules, err := store.ListSchedules()
	if err != nil {
		return fmt.Errorf("failed to list schedules: %w", err)
	}

	// Track present schedules so we know which ones to delete
	activeSchedules := make(map[string]struct{})

	for _, schedule := range schedules {
		record := findScheduleByName(existingSchedules, schedule.Name)
		if record == nil {
			// If schedule does not exist, create it
			if err := store.CreateSchedule(schedule); err != nil {
				return fmt.Errorf("failed to create schedule: %w", err)
			}

			log.Infof("created schedule %s", schedule.Name)
			continue
		}

		// If schedule exists, update it
		if err := store.UpdateSchedule(schedule); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}

		log.Infof("updated schedule %s", schedule.Name)

		activeSchedules[schedule.Name] = struct{}{}
	}

	// Delete schedules that are no longer present
	for _, schedule := range existingSchedules {
		if _, exists := activeSchedules[schedule.Name]; !exists {
			idStr := fmt.Sprint(schedule.ID)
			if err := store.DeleteSchedule(idStr); err != nil {
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
