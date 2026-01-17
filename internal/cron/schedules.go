package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	DefaultSchedulesFilePath = "/usr/local/share/schedules.json"
	executeCommand           = "/usr/local/bin/process-job"
	defaultCronFilePath      = "/data/crontab"
	defaultCommandTimeout    = 30
)

type crontabRunner func(path string) ([]byte, error)

func defaultRunCrontab(path string) ([]byte, error) {
	return exec.Command("crontab", path).CombinedOutput()
}

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
	return syncCrontab(ctx, store, log, defaultCronFilePath, defaultRunCrontab)
}

func syncCrontab(ctx context.Context, store *Store, log *logrus.Logger, cronFilePath string, runCrontab crontabRunner) error {
	schedules, err := store.ListEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list schedules: %w", err)
	}

	// Write to a temp file first so a single invalid schedule doesn't clobber the
	// last known-good crontab file.
	cronDir := filepath.Dir(cronFilePath)
	if err := os.MkdirAll(cronDir, 0o755); err != nil {
		return fmt.Errorf("failed to create crontab dir: %w", err)
	}

	file, err := os.CreateTemp(cronDir, "crontab-*")
	if err != nil {
		return fmt.Errorf("failed to create temp crontab file: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()

	for _, schedule := range schedules {
		entry := fmt.Sprintf("%s %s %d\n", schedule.Schedule, executeCommand, schedule.ID)
		_, err := file.WriteString(entry)
		if err != nil {
			return fmt.Errorf("failed to write to crontab file: %w", err)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp crontab file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temp crontab file: %w", err)
	}

	tempPath := file.Name()
	if out, err := runCrontab(tempPath); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			log.WithError(err).Errorf("crontab install failed: %s", msg)
			return fmt.Errorf("failed to sync crontab: %w: %s", err, msg)
		}
		log.WithError(err).Error("crontab install failed")
		return fmt.Errorf("failed to sync crontab: %w", err)
	}

	// Best-effort: keep the installed crontab content at the known path for debugging/ops.
	if err := os.Rename(tempPath, cronFilePath); err != nil {
		log.WithError(err).Warnf("failed to replace %s with generated crontab", cronFilePath)
	}

	log.Printf("Synced %d schedule(s) to crontab", len(schedules))
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
