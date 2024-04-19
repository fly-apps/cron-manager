package main

import (
	"context"
	"fmt"

	"github.com/fly-apps/cron-manager/internal/cron"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger := cron.SetupLogging()

	ctx := context.Background()

	store, err := cron.NewStore(cron.StorePath)
	if err != nil {
		panic(fmt.Errorf("failed to create store: %w", err))
	}
	defer func() { _ = store.Close() }()

	if err := cron.MonitorActiveJobs(ctx, store, logger); err != nil {
		panic(err)
	}
}
