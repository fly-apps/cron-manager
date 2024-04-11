package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

const Port = 5500

func StartHttpServer() error {
	log.SetFlags(0)

	r := chi.NewMux()
	r.Mount("/", Handler())

	server := &http.Server{
		Handler:           r,
		Addr:              fmt.Sprintf(":%v", Port),
		ReadHeaderTimeout: 3 * time.Second,
	}

	return server.ListenAndServe()
}

func Handler() http.Handler {
	r := chi.NewRouter()
	r.Route("/cronjobs", func(r chi.Router) {
		r.Post("/create", handleCreateCronJob)
		r.Post("/sync", handleCronSync)
	})
	r.Route("/jobs", func(r chi.Router) {
		r.Post("/process", handleJobProcess)
	})

	return r
}
