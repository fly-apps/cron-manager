package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/fly-apps/cron-manager/internal/cron"
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

	// Create a new job
	jobID, err := store.CreateJob(cronjob.ID)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// TODO - Consider coupling this with the above transaction
	job, err := store.FindJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	// Initialize client
	client, err := cron.NewClient(ctx, cronjob.AppName, store)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Provision a new machine to run the job
	machine, err := client.MachineProvision(ctx, cronjob, job)
	if err != nil {
		if machine != nil {
			if err := client.MachineDestroy(ctx, machine); err != nil {
				log.Printf("[ERROR] Failed to destroy machine %s: %s\n", machine.ID, err)
			}
		}

		if err := store.FailJob(job.ID, 1, err.Error()); err != nil {
			log.Printf("[ERROR] Failed to fail job %d: %s\n", job.ID, err)
		}

		return fmt.Errorf("failed to provision machine: %w", err)
	}

	defer func() {
		if err := client.MachineDestroy(ctx, machine); err != nil {
			log.Printf("[ERROR] Failed to destroy machine %s: %s\n", machine.ID, err)
		}
	}()

	// Execute the job
	resp, err := client.MachineExec(ctx, cronjob, job, machine)
	if err != nil {
		if err := store.FailJob(job.ID, int(resp.ExitCode), err.Error()); err != nil {
			log.Printf("[ERROR] Failed to update job job %d: %s\n", job.ID, err)
		}
	}

	// Complete the job
	if err := store.CompleteJob(job.ID, int(resp.ExitCode), resp.StdOut); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	log.Printf("[INFO] Job %d completed successfully!\n", job.ID)

	return nil
}
