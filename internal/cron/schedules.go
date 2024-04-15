package cron

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	schedulesFilePath = "/usr/local/share/schedules.json"
)

func SyncSchedules(log *logrus.Logger, store *Store) error {
	schedulesBytes, err := os.ReadFile(schedulesFilePath)
	if err != nil {
		return fmt.Errorf("failed to open schedules file: %w", err)
	}

	var schedules []Schedule
	if err := json.Unmarshal(schedulesBytes, &schedules); err != nil {
		return fmt.Errorf("failed to unmarshal schedules: %w", err)
	}

	for _, schedule := range schedules {
		// Check if schedule exists
		record, err := store.FindScheduleByName(schedule.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to find schedule: %w", err)
		}

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
	}

	if err := SyncCrontab(store); err != nil {
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	return nil
}
