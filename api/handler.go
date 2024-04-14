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
	r.Route("/jobs", func(r chi.Router) {
		r.Post("/trigger", WithLogging(handleJobTrigger, logger))
	})

	return r
}

func WithLogging(h http.HandlerFunc, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loggerKey, logger)
		h(w, r.WithContext(ctx))
	}
}
