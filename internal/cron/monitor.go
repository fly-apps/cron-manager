package cron

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/superfly/fly-go"
)

const (
	monitorFrequency = 5 * time.Second
)

// MonitorActiveJobs checks the status of all active jobs and updates their status.
func MonitorActiveJobs(ctx context.Context, store *Store, log *logrus.Logger) error {
	ticker := time.NewTicker(monitorFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Find all active jobs
			jobs, err := store.ListJobsByStatus(ctx, JobStatusRunning)
			if err != nil {
				return fmt.Errorf("failed to find active jobs: %w", err)
			}
			var wg sync.WaitGroup

			// Loop through the running jobs and determine their status
			for _, job := range jobs {
				wg.Add(1)
				go func(job Job) {
					defer wg.Done()
					if err := evaluateJob(ctx, log, store, job); err != nil {
						log.WithError(err).Errorf("failed to monitor job %d", job.ID)
					}
				}(job)

			}

			wg.Wait()
		}
	}
}

func evaluateJob(ctx context.Context, logger *logrus.Logger, store *Store, job Job) error {
	// Fetch the associated schedule for the job
	schedule, err := store.FindSchedule(ctx, job.ScheduleID)
	if err != nil {
		return fmt.Errorf("failed to find schedule for job %d: %w", job.ID, err)
	}

	log := logger.WithFields(logrus.Fields{
		"app-name":   schedule.AppName,
		"schedule":   schedule.Name,
		"job-id":     job.ID,
		"machine-id": job.MachineID.String,
	})

	// Initialize the flaps client
	client, err := NewFlapsClient(ctx, schedule.AppName, store)
	if err != nil {
		return fmt.Errorf("failed to create client for job %d: %w", job.ID, err)
	}

	// Fetch the machine associated with the job
	machine, err := client.MachineGet(ctx, job.MachineID.String)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 404 {
				// Machines are queryable up to 48 hours after they are destroyed.
				// If the cron manager is shutdown or inactive for more than 48 hours, we will not be able to evaluate the result.
				log.WithError(err).Errorf("failed to get machine %s: %v", job.MachineID.String, err)
				if err := store.FailJob(ctx, job.ID, -1, "machine destroyed before we could interpret the results"); err != nil {
					log.WithError(err).Errorf("failed to update job %d status", job.ID)
				}
			} else {
				log.WithError(err).Errorf("failed to get machine %s", job.MachineID.String)
			}
		}

		return nil
	}

	if machine == nil {
		log.Errorf("job %d has a nil machine %s", job.ID, job.MachineID.String)
		return nil
	}

	log.Debugf("Monitoring job")

	startEvent := findEvent(machine, "start")
	if startEvent == nil {
		log.Debugf("Machine %s has not started yet", machine.ID)
		return nil
	}

	switch machine.State {
	case fly.MachineStateDestroyed:
		log.Debugf("Machine %s is destroyed", machine.ID)

		log = log.WithField("execution-time", fmt.Sprintf("%.2fs", calculateExecutionTime(machine)))

		// Find the exit event
		event := findEvent(machine, "exit")
		if event == nil {
			return fmt.Errorf("failed to find exit event for destroyed machine %s", machine.ID)
		}

		// Get the exit code
		if event.Request != nil && event.Request.ExitEvent != nil {
			exitCode := event.Request.ExitEvent.ExitCode
			if exitCode != 0 {
				if err := store.FailJob(ctx, job.ID, exitCode, ""); err != nil {
					log.WithError(err).Errorf("failed to update job %d status", job.ID)
				}
				log.Infof("Job failed with exit code %d", exitCode)
			} else {
				if err := store.CompleteJob(ctx, job.ID, exitCode, ""); err != nil {
					log.WithError(err).Errorf("failed to update job %d status", job.ID)
				}
				log.Infof("Job completed successfully")
			}
		}
	default:
		executionTime := calculateExecutionTime(machine)
		log = log.WithField("execution-time", fmt.Sprintf("%.2fs", executionTime))

		// Machine is in a non-destroyed state, verify run time hasn't exceeded the command timeout
		if executionTime > float64(schedule.CommandTimeout) {
			err := fmt.Sprintf("machine `%s` exceeded the command timeout of %d seconds.",
				machine.ID,
				schedule.CommandTimeout)

			log.Warn(err)

			if err := client.MachineDestroy(ctx, machine); err != nil {
				return fmt.Errorf("failed to destroy machine %s: %w", machine.ID, err)
			}

			if err := store.FailJob(ctx, job.ID, -1, err); err != nil {
				log.WithError(err).Errorf("failed to update job %d status", job.ID)
			}
		}

		log.Debugf("Machine is in state %s", machine.State)
	}

	return nil
}

func calculateExecutionTime(machine *fly.Machine) float64 {
	// Base it off the time the machine entered a start state.
	startEvent := findEvent(machine, "start")
	if startEvent == nil {
		return 0
	}

	startTime := startEvent.Time()

	// Default to the current time if the machine has not exited yet.
	endTime := time.Now()

	exitEvent := findEvent(machine, "exit")
	if exitEvent != nil {
		endTime = exitEvent.Time()
	}

	return endTime.Sub(startTime).Seconds()
}

func findEvent(machine *fly.Machine, eventType string) *fly.MachineEvent {
	if len(machine.Events) == 0 {
		return nil
	}

	for _, event := range machine.Events {
		if event.Type == eventType {
			return event
		}
	}

	return nil
}
