package main

import (
	"github.com/fly-apps/cron-manager/api"
	"github.com/fly-apps/cron-manager/internal/cron"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger := cron.SetupLogging()

	if err := api.StartHttpServer(logger); err != nil {
		panic(err)
	}
}
