package cron

import (
	"context"
	"fmt"

	"github.com/google/shlex"
	"github.com/sirupsen/logrus"
)

func ProcessJob(ctx context.Context, log *logrus.Logger, store *Store, scheduleID int) (err error) {
	schedule, err := store.FindSchedule(ctx, scheduleID)
	if err != nil {
		return err
	}

	if err := prepareJob(schedule); err != nil {
		return fmt.Errorf("failed to prepare job: %w", err)
	}

	job, err := store.CreateJob(ctx, schedule.ID)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	logger := log.WithFields(logrus.Fields{
		"app-name": schedule.AppName,
		"schedule": schedule.Name,
		"job-id":   job.ID,
	})

	// Defer a function to handle job processing errors
	defer func() {
		if err != nil {
			logger.WithError(err).Error("job processing failed")
			if failErr := store.FailJob(ctx, job.ID, 1, err.Error()); failErr != nil {
				logger.WithError(failErr).Error("failed to update job status")
			}
		}
	}()

	logger.Info("Preparing job...")

	client, err := NewFlapsClient(ctx, schedule.AppName, store)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Provision machine to run the job
	machine, err := client.MachineProvision(ctx, schedule, job)
	if err != nil && machine != nil {
		if destroyErr := client.MachineDestroy(ctx, machine); destroyErr != nil {
			logger.Warnf("failed to destroy machine %s: %s", machine.ID, destroyErr)
		}
		return fmt.Errorf("failed to provision machine: %w", err)
	}

	logger = logger.WithField("machine-id", machine.ID)

	// Set the job status to running
	if err := store.UpdateJobStatus(ctx, job.ID, JobStatusRunning); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
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
	schedule.Config.Metadata["managed-by-cron-manager"] = "true"

	return nil
}
