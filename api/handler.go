package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type contextKey int

const (
	// Port is the port the HTTP server listens on
	Port                 = 5500
	loggerKey contextKey = iota
)

func StartHttpServer(logger *logrus.Logger) error {
	r := chi.NewMux()
	r.Mount("/", Handler(logger))

	w := logger.Writer()
	defer w.Close()

	server := &http.Server{
		Handler:           r,
		Addr:              fmt.Sprintf(":%v", Port),
		ReadHeaderTimeout: 3 * time.Second,
		ErrorLog:          log.New(w, "", 0),
	}

	return server.ListenAndServe()
}

func Handler(logger *logrus.Logger) http.Handler {
	r := chi.NewRouter()
	r.Route("/cronjobs", func(r chi.Router) {
		r.Post("/create", handleCreateCronJob)
		r.Post("/sync", handleCronSync)
	})
	r.Route("/jobs", func(r chi.Router) {
		r.Post("/process", WithLogging(handleJobProcess, logger))
	})

	return r
}

func WithLogging(h http.HandlerFunc, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loggerKey, logger)
		h(w, r.WithContext(ctx))
	}
}
