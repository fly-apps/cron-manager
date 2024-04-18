package cron

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/superfly/fly-go"
)

const (
	monitorFrequency = 5 * time.Second
)

func MonitorActiveJobs(ctx context.Context, store *Store, logger *logrus.Logger) error {
	log := logger.WithField("thr", "monitor")
	ticker := time.NewTicker(monitorFrequency)
	defer ticker.Stop()

	log.Infof("starting async job monitor...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Find all active jobs
			jobs, err := store.ListJobsByStatus(JobStatusRunning)
			if err != nil {
				return fmt.Errorf("failed to find active jobs: %w", err)
			}

			// Loop through the running jobs and determine their status
			for _, job := range jobs {
				// Fetch the associated schedule for the job
				schedule, err := store.FindSchedule(job.ScheduleID)
				if err != nil {
					log.WithError(err).Errorf("failed to find schedule for job %d", job.ID)
					continue
				}

				// Initialize the flaps client
				client, err := NewFlapsClient(ctx, schedule.AppName, store)
				if err != nil {
					log.WithError(err).Errorf("failed to create client for job %d", job.ID)
					continue
				}

				// Fetch the machine associated with the job
				machine, err := client.MachineGet(ctx, job.MachineID.String)
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						if exitErr.ExitCode() == 404 {
							// Machines are queryable up to 48 hours after they are destroyed.
							// If the cron manager is shutdown or inactive for more than 48 hours, we will not be able to evaluate the result.
							log.WithError(err).Errorf("failed to get machine %s: %w", job.MachineID.String, err)
							if err := store.FailJob(job.ID, -1, "machine destroyed before we could interpret the results"); err != nil {
								log.WithError(err).Errorf("failed to update job %d status", job.ID)
							}

							continue
						}

						log.WithError(err).Errorf("failed to get machine %s: %v", job.MachineID.String, err)
					}
				}

				log.Debugf("monitoring job %d running on machine %s", job.ID, machine.ID)

				switch machine.State {
				case fly.MachineStateDestroyed:
					log.Debugf("machine %s is destroyed", machine.ID)

					// Find the exit event
					event := findExitEvent(machine)
					if event == nil {
						log.Errorf("failed to find exit event for machine %s", machine.ID)
						continue
					}

					// Get the exit code
					if event.Request != nil && event.Request.ExitEvent != nil {
						exitCode := event.Request.ExitEvent.ExitCode
						if exitCode != 0 {
							if err := store.FailJob(job.ID, exitCode, ""); err != nil {
								log.WithError(err).Errorf("failed to update job %d status", job.ID)
							}
						} else {
							if err := store.CompleteJob(job.ID, exitCode, ""); err != nil {
								log.WithError(err).Errorf("failed to update job %d status", job.ID)
							}
						}
					}
				default:
					log.Debugf("machine %s is in state %s", machine.ID, machine.State)
				}
			}
		}
	}
}

func findExitEvent(machine *fly.Machine) *fly.MachineEvent {
	if len(machine.Events) == 0 {
		return nil
	}

	for _, event := range machine.Events {
		if event.Type == "exit" {
			return event
		}
	}

	return nil
}
