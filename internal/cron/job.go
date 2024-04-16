package cron

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	fly "github.com/superfly/fly-go"
)

const (
	defaultExecTimeout = 30
)

// ReconcileJobs reconciles jobs that are flagged as pending or running at the time of startup.
func ReconcileJobs(store *Store, log *logrus.Logger) error {
	log.Info("reconciling jobs...")

	// Initialize the flaps client
	client, err := NewClient(context.Background(), "", store)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Query pending jobs
	jobs, err := store.ListReconcilableJobs()
	if err != nil {
		return fmt.Errorf("failed to list reconciliable jobs: %w", err)
	}

	for _, job := range jobs {
		switch job.Status {
		case JobStatusPending:
			log.WithField("job-id", job.ID).Info("reconciling pending job")
			// If the job is pending and does not have a machine ID, it was interrupted during shutdown
			// and can be marked as failed.
			if job.MachineID.Valid {
				if err := client.MachineDestroy(context.Background(), &fly.Machine{ID: job.MachineID.String}); err != nil {
					log.WithError(err).Error("failed to reconcile machine tied to pending job")
					continue
				}
			}

			if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
				log.WithError(err).Error("failed to update job status")
			}
		case JobStatusRunning:
			log.WithField("job-id", job.ID).Info("reconciling running job")

			if !job.MachineID.Valid {
				// this should never happen.
				if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
					log.WithError(err).Error("failed to update job status")
				}
				continue
			}

			machine, err := client.MachineGet(context.Background(), job.MachineID.String)
			// If the machine is not found, we can assume it was already destroyed.
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					log.WithError(err).Errorf("failed to get machine %+v", exitErr)
					if exitErr.ExitCode() != 404 {
						log.WithError(err).Errorf("failed to get machine %s", job.MachineID.String)
						continue
					}
				}
			}

			if machine == nil {
				if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
					log.WithError(err).Error("failed to reconcile job")
				}
				continue
			}

			if machine.State != fly.MachineStateStarted {
				if err := client.MachineDestroy(context.Background(), machine); err != nil {
					log.WithError(err).Errorf("failed to destroy machine %s", job.MachineID.String)
				}

				if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
					log.WithError(err).Error("failed to update job status")
				}

				continue
			}

			// If the machine is in a started state, we don't actually know whether the command was issued or not.
			// For now, we will tear it down so long as the time since the last update is greater than the exec timeout.
			// This is a bit of a hack, but it's the best we can do without a more sophisticated solution.
			if job.UpdatedAt.Add(defaultExecTimeout).Before(job.UpdatedAt) {
				if err := client.MachineDestroy(context.Background(), machine); err != nil {

					log.WithError(err).Errorf("failed to destroy machine %s", job.MachineID.String)
					// Continue so this can be retried.
					continue
				}

				if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
					log.WithError(err).Error("failed to update job status")
				}
			}
		}
	}

	if len(jobs) > 0 {
		log.Infof("reconciled %d jobs", len(jobs))
	} else {
		log.Info("no jobs to reconcile")
	}

	return nil
}

func ProcessJob(ctx context.Context, log *logrus.Logger, store *Store, scheduleID int) error {
	schedule, err := store.FindSchedule(scheduleID)
	if err != nil {
		return err
	}

	logger := log.WithField("schedule-id", schedule.ID)
	logger.Info("processing scheduled job...")

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

	logger.Debugf("provisioning machine with image %s...", schedule.Config.Image)

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

	// Ensure the machine gets torn down on exit
	defer func() {
		logger.Debugf("cleaning up job...")
		if err := client.MachineDestroy(ctx, machine); err != nil {
			logger.WithError(err).Warn("failed to destroy machine")
		}
	}()

	// Set the job status to running
	if err := store.UpdateJobStatus(jobID, JobStatusRunning); err != nil {
		return failJob(1, fmt.Errorf("failed to update job status: %w", err))
	}

	logger = logger.WithField("machine-id", machine.ID)
	logger.Debug("waiting for machine to start...")

	// Wait for the machine to start
	if err := client.WaitForStatus(ctx, machine, fly.MachineStateStarted); err != nil {
		return failJob(1, fmt.Errorf("failed to wait for machine to start: %w", err))
	}

	logger.Debugf("executing command `%s` against machine...", schedule.Command)

	// Execute the job
	resp, err := client.MachineExec(ctx, schedule.Command, machine.ID, defaultExecTimeout)
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
