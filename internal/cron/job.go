package cron

import (
	"context"
	"fmt"

	"github.com/google/shlex"
	"github.com/sirupsen/logrus"
)

const (
	defaultExecTimeout = 30
)

func ProcessJob(ctx context.Context, log *logrus.Logger, store *Store, scheduleID int) error {
	schedule, err := store.FindSchedule(scheduleID)
	if err != nil {
		return err
	}

	// Prepare the job
	if err := prepareJob(schedule); err != nil {
		return fmt.Errorf("failed to prepare job: %w", err)
	}

	// Create a new job
	job, err := store.CreateJob(schedule.ID)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	logger := log.WithFields(logrus.Fields{
		"app-name": schedule.AppName,
		"schedule": schedule.Name,
		"job-id":   job.ID,
	})

	failJob := func(exitCode int, err error) error {
		if err := store.FailJob(job.ID, exitCode, err.Error()); err != nil {
			logger.WithError(err).Error("failed to update job status")
		}
		logger.WithError(err).Errorf("job failed with exit code %d", exitCode)
		return err
	}

	logger.Info("Preparing job...")

	// Initialize client
	client, err := NewFlapsClient(ctx, schedule.AppName, store)
	if err != nil {
		return failJob(1, fmt.Errorf("failed to create client: %w", err))
	}

	// Provision a new machine to run the job
	machine, err := client.MachineProvision(ctx, logger, schedule, job)
	if err != nil {
		if machine != nil {
			if err := client.MachineDestroy(ctx, machine); err != nil {
				logger.Warnf("failed to destroy machine %s: %s", machine.ID, err)
			}
		}

		return failJob(1, fmt.Errorf("failed to provision machine: %w", err))
	}

	logger = logger.WithField("machine-id", machine.ID)

	// Set the job status to running
	if err := store.UpdateJobStatus(job.ID, JobStatusRunning); err != nil {
		return failJob(1, fmt.Errorf("failed to update job status: %w", err))
	}

	logger.Infof("Running job...")

	return nil
}

func prepareJob(schedule *Schedule) error {
	cmdSlice, err := shlex.Split(schedule.Command)
	if err != nil {
		return fmt.Errorf("failed to split command: %w", err)
	}

	schedule.Config.Init.Cmd = cmdSlice

	if schedule.Config.Metadata == nil {
		schedule.Config.Metadata = make(map[string]string)
	}

	// Indicate the associated Machine was created by the cron manager
	// This helps scope the reconciliation logic.
	schedule.Config.Metadata["managed-by-cron-manager"] = "true"

	return nil
}
