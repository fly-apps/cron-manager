package cron

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	fly "github.com/superfly/fly-go"
)

// Reconcile works to correct any inconsistencies between the state of the system and the state of the database.
func Reconcile(ctx context.Context, store *Store, log *logrus.Logger) error {
	schedules, err := store.ListSchedules()
	if err != nil {
		return fmt.Errorf("failed to list schedules: %w", err)
	}

	if len(schedules) == 0 {
		log.Info("nothing to reconcile...")
		return nil
	}

	if err := reconcileRunningMachines(ctx, store, log, schedules); err != nil {
		return fmt.Errorf("failed to reconcile running machines: %w", err)
	}

	if err := reconcileJobs(store, log); err != nil {
		return fmt.Errorf("failed to reconcile jobs: %w", err)
	}

	log.Infof("reconciliation complete")

	return nil
}

func reconcileRunningMachines(ctx context.Context, store *Store, log *logrus.Logger, schedules []Schedule) error {
	for _, schedule := range schedules {
		client, err := NewFlapsClient(context.Background(), schedule.AppName, store)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		machines, err := client.MachineList(ctx, "")
		if err != nil {
			return fmt.Errorf("failed to list machines for app %s: %w", schedule.AppName, err)
		}

		if len(machines) == 0 {
			continue
		}

		for _, machine := range machines {

			// Skip any machines that are not managed by the cron manager.
			if machine.Config.Metadata == nil || machine.Config.Metadata["managed-by-cron-manager"] != "true" {
				continue
			}

			// Find the job associated with the machine
			job, err := store.FindJobByMachineID(machine.ID)
			if err != nil {
				log.WithError(err).Errorf("failed to find job for machine %s", machine.ID)
				continue
			}

			if job == nil {
				log.Warnf("machine %s associated with app %s is not tied to an existing job...", machine.ID, schedule.AppName)
				continue
			}

			switch job.Status {
			case JobStatusRunning:
				log.WithField("job-id", job.ID).Info("reconciling running job")

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
				if job.UpdatedAt.Add(defaultExecTimeout).Before(job.UpdatedAt) {
					if err := client.MachineDestroy(context.Background(), machine); err != nil {
						log.WithError(err).Errorf("failed to destroy machine %s", job.MachineID.String)
						continue // Continue so this can be retried.
					}

					if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
						log.WithError(err).Error("failed to update job status")
					}
				}
			case JobStatusFailed, JobStatusCompleted:
				log.Warnf("found terminal job %d with running machine %s. cleaning it up...", job.ID, machine.ID)
				if err := client.MachineDestroy(context.Background(), machine); err != nil {
					log.WithError(err).Errorf("failed to destroy machine %s", machine.ID)
				}
			}
		}
	}

	return nil
}

func reconcileJobs(store *Store, log *logrus.Logger) error {
	// Find all jobs that are pending or running
	jobs, err := store.ListReconcilableJobs()
	if err != nil {
		return fmt.Errorf("failed to list reconciliable jobs: %w", err)
	}

	for _, job := range jobs {
		switch job.Status {
		case JobStatusPending:
			log.WithField("job-id", job.ID).Info("reconciling pending job")

			if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
				log.WithError(err).Error("failed to update job status")
			}
		case JobStatusRunning:
			log.WithField("job-id", job.ID).Info("reconciling running job")

			// this should never happen, but check just in case.
			if !job.MachineID.Valid {
				if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
					log.WithError(err).Error("failed to update job status")
				}
				continue
			}

			// Get the associated schedule.
			schedule, err := store.FindSchedule(job.ScheduleID)
			if err != nil {
				log.WithError(err).Errorf("failed to find schedule %d", job.ScheduleID)
				continue
			}

			// Initialize the flaps client
			client, err := NewFlapsClient(context.Background(), schedule.AppName, store)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			machine, err := client.MachineGet(context.Background(), job.MachineID.String)
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					log.WithError(err).Errorf("failed to get machine %+v", exitErr)
					if exitErr.ExitCode() != 404 {
						log.WithError(err).Errorf("failed to get machine %s", job.MachineID.String)
						continue
					}
				}
			}

			// If the machine is not found, we can assume it was already destroyed.
			if machine == nil {
				if err := store.FailJob(job.ID, -1, "job was interrupted on shutdown"); err != nil {
					log.WithError(err).Error("failed to reconcile job")
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

	return nil
}
