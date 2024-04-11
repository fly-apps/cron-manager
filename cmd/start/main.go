package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fly-apps/cron-manager/api"
	"github.com/fly-apps/cron-manager/internal/cron"
)

func main() {
	log.SetFlags(0)

	requiredPasswords := []string{"FLY_API_TOKEN"}
	for _, str := range requiredPasswords {
		if _, exists := os.LookupEnv(str); !exists {
			panic(fmt.Errorf("%s is required", str))
		}
	}

	log.Printf("Configuring Cronjob State store...")
	store, err := cron.NewStore()
	if err != nil {
		panic(fmt.Errorf("failed to create store: %w", err))
	}

	if err := store.SetupDB(); err != nil {
		panic(fmt.Errorf("failed to setup db: %w", err))
	}

	if err := cron.InitializeCron(store); err != nil {
		panic(fmt.Errorf("failed to sync crontab: %w", err))
	}

	log.Printf("Starting HTTP server on port %v...", api.Port)
	if err := api.StartHttpServer(); err != nil {
		panic(fmt.Errorf("failed to start http server: %w", err))
	}
}
