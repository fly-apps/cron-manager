package api

import (
	"encoding/json"
	"net/http"

	"github.com/fly-apps/cron-manager/internal/cron"
	"github.com/sirupsen/logrus"
)

type triggerJobRequest struct {
	ID int `json:"id"`
}

func handleJobTrigger(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(loggerKey).(*logrus.Logger)

	var req triggerJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Failed to decode job run request")
		renderErr(w, err)
		return
	}
	defer func() { _ = r.Body.Close() }()

	store, err := cron.NewStore(cron.StorePath)
	if err != nil {
		log.WithError(err).Error("Failed to initialize sqlite")
		renderErr(w, err)
		return
	}
	defer store.Close()

	if err := cron.ProcessJob(r.Context(), log, store, req.ID); err != nil {
		log.WithError(err).Error("Failed to process job")
		renderErr(w, err)
		return
	}
}
