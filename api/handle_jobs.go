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
	ctx := r.Context()

	log := ctx.Value(loggerKey).(*logrus.Logger)

	var req triggerJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("failed to decode job run request")
		renderErr(w, err)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.WithError(err).Error("failed to close request body")
		}
	}()

	store, err := cron.NewStore(cron.DefaultStorePath)
	if err != nil {
		log.WithError(err).Error("failed to initialize sqlite")
		renderErr(w, err)
		return
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.WithError(err).Error("failed to close store")
		}
	}()

	if err := cron.ProcessJob(ctx, log, store, req.ID); err != nil {
		log.WithError(err).Error("failed to process job")
		renderErr(w, err)
		return
	}

}
