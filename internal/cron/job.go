package cron

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	fly "github.com/superfly/fly-go"
)

func ProcessJob(ctx context.Context, log *logrus.Logger, store *Store, scheduleID int) error {
	// Resolve the associated schedule
	schedule, err := store.FindSchedule(scheduleID)
	if err != nil {
		return err
	}

	logger := log.WithField("schedule-id", schedule.ID)
	logger.Info("processing scheduled task...")

	// Create a new job
	jobID, err := store.CreateJob(schedule.ID)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	failJob := func(exitCode int, err error) error {
		if err := store.FailJob(jobID, exitCode, err.Error()); err != nil {
			logger.WithError(err).Error("failed to update job status")
		}
		logger.WithError(err).Errorf("job failed with exit code %d", exitCode)
		return err
	}

	// TODO - Consider coupling this with the above transaction
	job, err := store.FindJob(fmt.Sprint(jobID))
	if err != nil {
		return failJob(1, fmt.Errorf("failed to find job: %w", err))
	}

	logger = logger.WithField("job-id", job.ID)

	// Initialize client
	client, err := NewClient(ctx, schedule.AppName, store)
	if err != nil {
		return failJob(1, fmt.Errorf("failed to create client: %w", err))
	}

	logger.Debugf("provisioning machine with image %s...", schedule.Image)

	// Provision a new machine to run the job
	machine, err := client.MachineProvision(ctx, schedule, job)
	if err != nil {
		if machine != nil {
			if err := client.MachineDestroy(ctx, machine); err != nil {
				logger.Warnf("failed to destroy machine %s: %s", machine.ID, err)
			}
		}
		return failJob(1, fmt.Errorf("failed to provision machine: %w", err))
	}

	logger = logger.WithField("machine-id", machine.ID)

	// Ensure the machine gets torn down on exit
	defer func() {
		logger.Debugf("cleaning up job...")
		if err := client.MachineDestroy(ctx, machine); err != nil {
			logger.WithError(err).Warn("failed to destroy machine")
		}
	}()

	logger.Debug("waiting for machine to start...")

	// Wait for the machine to start
	if err := client.WaitForStatus(ctx, machine, fly.MachineStateStarted); err != nil {
		return failJob(1, fmt.Errorf("failed to wait for machine to start: %w", err))
	}

	logger.Debugf("executing command `%s` against machine...", schedule.Command)

	// Execute the job
	resp, err := client.MachineExec(ctx, schedule, job, machine)
	if err != nil {
		return failJob(1, fmt.Errorf("failed to execute job: %w", err))
	}

	// Handle failures.
	switch {
	case resp.ExitCode != 0:
		logger.Errorf("job failed with exit code %d", resp.ExitCode)
		return failJob(int(resp.ExitCode), fmt.Errorf("job failed with exit code %d", resp.ExitCode))
	case resp.ExitCode == 0 && resp.StdErr != "":
		logger.Errorf("job failed with stderr %s", resp.StdErr)
		return failJob(-1, fmt.Errorf("%s", resp.StdErr))
	}

	if err := store.CompleteJob(job.ID, int(resp.ExitCode), resp.StdOut); err != nil {
		logger.WithError(err).Error("failed to set job status to complete")
	}

	logger.Infof("job completed successfully!")

	return nil
}
