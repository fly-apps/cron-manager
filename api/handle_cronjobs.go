package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/fly-apps/cron-manager/internal/cron"
)

type CreateCronJobRequest struct {
	AppName       string `json:"app_name"`
	Image         string `json:"image"`
	Schedule      string `json:"schedule"`
	Command       string `json:"command"`
	RestartPolicy string `json:"restart_policy"`
}

func handleCronSync(w http.ResponseWriter, _ *http.Request) {
	store, err := cron.NewStore()
	if err != nil {
		log.Printf("[ERROR] Failed to initialize store: %s\n", err)
		renderErr(w, err)
		return
	}

	if err := cron.SyncCrontab(store); err != nil {
		log.Printf("[ERROR] Failed to sync crontab: %s\n", err)
		renderErr(w, err)
		return
	}
}

func handleCreateCronJob(w http.ResponseWriter, r *http.Request) {
	var req CreateCronJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode create cronjob request: %s\n", err)
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

	if err := createCronJob(r.Context(), store, req); err != nil {
		log.Printf("[ERROR] Failed to create cronjob: %s\n", err)
		renderErr(w, err)
		return
	}
}

func createCronJob(ctx context.Context, store *cron.Store, req CreateCronJobRequest) error {
	return store.CreateCronJob(req.AppName, req.Image, req.Schedule, req.Command, req.RestartPolicy)
}
