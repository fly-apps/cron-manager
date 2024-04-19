package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/fly-apps/cron-manager/internal/flycheck"
)

type contextKey int

const (
	Port                 = 5500
	loggerKey contextKey = iota
)

var shuttingDown bool

func shutdownInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shuttingDown {
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func StartHttpServer(logger *logrus.Logger) error {
	r := chi.NewMux()
	r.Use(shutdownInterceptor)
	r.Mount("/flycheck", flycheck.Handler())
	r.Mount("/command", Handler(logger))

	w := logger.Writer()
	defer func() { _ = w.Close() }()

	server := &http.Server{
		Handler:           r,
		Addr:              fmt.Sprintf(":%v", Port),
		ReadHeaderTimeout: 3 * time.Second,
		ErrorLog:          log.New(w, "", 0),
	}

	// Create a channel to listen for shutdown signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Error starting server: %v", err)
		}
	}()

	<-stopChan
	shuttingDown = true // Indicate that the server is shutting down
	logger.Println("Shutdown signal received, shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the server gracefully
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error during server shutdown: %v", err)
	}

	logger.Println("Server gracefully stopped")
	return nil
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
