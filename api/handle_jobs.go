package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/superfly/fly-go"
)

type processJobRequest struct {
	ID int `json:"id"`
}

func handleJobProcess(w http.ResponseWriter, r *http.Request) {
	var req processJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode job run request: %s\n", err)
		renderErr(w, err)
		return
	}
	defer func() { _ = r.Body.Close() }()

	store, err := cron.NewStore()
	if err != nil {
		log.Printf("[ERROR] Failed to create store: %s\n", err)
		renderErr(w, err)
		return
	}

	if err := processJob(r.Context(), store, req); err != nil {
		log.Printf("[ERROR] Failed to process job: %s\n", err)
		renderErr(w, err)
		return
	}
}

func processJob(ctx context.Context, store *cron.Store, req processJobRequest) error {
	// Resolve the associated cronjob
	cronjob, err := store.FindCronJob(req.ID)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Processing Cronjob %d...\n", req.ID)

	// Create a new job
	jobID, err := store.CreateJob(cronjob.ID)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	failJob := func(err error) error {
		if err := store.FailJob(jobID, 1, err.Error()); err != nil {
			log.Printf("[ERROR] Failed to fail job %d: %s\n", jobID, err)
		}
		return err
	}

	// TODO - Consider coupling this with the above transaction
	job, err := store.FindJob(fmt.Sprint(jobID))
	if err != nil {
		return failJob(fmt.Errorf("failed to find job: %w", err))
	}

	// Initialize client
	client, err := cron.NewClient(ctx, cronjob.AppName, store)
	if err != nil {
		return failJob(fmt.Errorf("failed to create client: %w", err))
	}

	log.Printf("[INFO] Provisioning machine with image %s...\n", cronjob.Image)

	// Provision a new machine to run the job
	machine, err := client.MachineProvision(ctx, cronjob, job)
	if err != nil {
		if machine != nil {
			if err := client.MachineDestroy(ctx, machine); err != nil {
				log.Printf("[ERROR] Failed to destroy machine %s: %s\n", machine.ID, err)
			}
		}

		return failJob(fmt.Errorf("failed to provision machine: %w", err))
	}

	// Ensure the machine gets torn down on exit
	defer func() {
		log.Printf("[INFO] Cleaning up job %d...\n", job.ID)
		if err := client.MachineDestroy(ctx, machine); err != nil {
			log.Printf("[ERROR] Failed to destroy machine %s: %s\n", machine.ID, err)
		}
	}()

	log.Printf("[INFO] Waiting for machine to start...\n")
	// Wait for the machine to start
	if err := client.WaitForStatus(ctx, machine, fly.MachineStateStarted); err != nil {
		return failJob(fmt.Errorf("failed to wait for machine to start: %w", err))
	}

	log.Printf("[INFO] Executing command %s on machine %s...\n", cronjob.Command, machine.ID)

	// Execute the job
	resp, err := client.MachineExec(ctx, cronjob, job, machine)
	if err != nil {
		return failJob(fmt.Errorf("failed to execute job: %w", err))
	}

	// There's a bug in the exec code that prevents the proper exit code from getting picked up.
	// For now, if stderr exists we'll assume the job failed.
	if resp.StdErr != "" {
		log.Printf("[ERROR] Job %d failed with stderr %s\n", job.ID, resp.StdErr)
		return failJob(fmt.Errorf("%s", resp.StdErr))
	}

	log.Printf("[INFO] Job %d exited with code %d - stdout %s - stderr %s\n", job.ID, resp.ExitCode, resp.StdOut, resp.StdErr)
	// Complete the job
	if err := store.CompleteJob(job.ID, int(resp.ExitCode), resp.StdOut); err != nil {
		log.Printf("[ERROR] Failed to complete job %d: %s\n", job.ID, err)
	}

	log.Printf("[INFO] Job %d completed successfully!\n", job.ID)

	return nil
}
