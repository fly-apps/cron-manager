package cron

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	fly "github.com/superfly/fly-go"

	_ "github.com/mattn/go-sqlite3"
)

const (
	testStorePath = "./test.db"
)

const testData = `[
    {
        "name": "uptime-check",
        "app_name": "shaun-pg-flex",
        "schedule": "* * * * *",
        "region": "iad",
        "command": "uptime",
		"command_timeout": 60,
        "enabled": true,
        "config": {
            "auto_destroy": true,
            "guest": {
                "cpu_kind": "shared",
                "cpus": 1,
                "memory_mb": 512
            },
            "image": "ghcr.io/livebook-dev/livebook:0.11.4",
            "restart": {
                "max_retries": 1,
                "policy": "no"
            }
        }
    },
	{
		"name": "test-check",
        "app_name": "shaun-pg-flex",
        "schedule": "* * * * *",
        "region": "iad",
        "command": "uptime",
        "enabled": false,
        "config": {
            "auto_destroy": true,
            "guest": {
                "cpu_kind": "shared",
                "cpus": 1,
                "memory_mb": 512
            },
            "image": "ghcr.io/livebook-dev/livebook:0.11.4",
            "restart": {
                "max_retries": 1,
                "policy": "no"
            }
        }
	}

]`

// same data as testData but with different region
const testData2 = `[
    {
        "name": "uptime-check",
        "app_name": "shaun-pg-flex",
        "schedule": "* * * * *",
        "region": "ord",
        "command": "uptime",
		"command_timeout": 60,
        "enabled": true,
        "config": {
            "auto_destroy": true,
            "guest": {
                "cpu_kind": "shared",
                "cpus": 1,
                "memory_mb": 512
            },
            "image": "ghcr.io/livebook-dev/livebook:0.11.4",
            "restart": {
                "max_retries": 1,
                "policy": "no"
            }
        }
    },
	{
		"name": "test-check",
        "app_name": "shaun-pg-flex",
        "schedule": "* * * * *",
        "region": "ord",
        "command": "uptime",
        "enabled": false,
        "config": {
            "auto_destroy": true,
            "guest": {
                "cpu_kind": "shared",
                "cpus": 1,
                "memory_mb": 512
            },
            "image": "ghcr.io/livebook-dev/livebook:0.11.4",
            "restart": {
                "max_retries": 1,
                "policy": "no"
            }
        }
	}

]`

func TestReadSchedulesFromFile(t *testing.T) {
	schedulesFile, err := createSchedulesFile([]byte(testData))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(schedulesFile.Name()) }()

	schedules, err := readSchedulesFromFile(schedulesFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(schedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(schedules))
	}

	expected := []Schedule{
		{
			Name:           "uptime-check",
			AppName:        "shaun-pg-flex",
			Schedule:       "* * * * *",
			Region:         "iad",
			Command:        "uptime",
			CommandTimeout: 60,
			Enabled:        true,
			Config: fly.MachineConfig{
				AutoDestroy: true,
				Guest: &fly.MachineGuest{
					CPUKind:  "shared",
					CPUs:     1,
					MemoryMB: 512,
				},
				Image: "ghcr.io/livebook-dev/livebook:0.11.4",
				Restart: &fly.MachineRestart{
					MaxRetries: 1,
					Policy:     "no",
				},
			},
		},
		{
			Name:           "test-check",
			AppName:        "shaun-pg-flex",
			Schedule:       "* * * * *",
			Region:         "iad",
			Command:        "uptime",
			CommandTimeout: 0,
			Enabled:        false,
			Config: fly.MachineConfig{
				AutoDestroy: true,
				Guest: &fly.MachineGuest{
					CPUKind:  "shared",
					CPUs:     1,
					MemoryMB: 512,
				},
				Image: "ghcr.io/livebook-dev/livebook:0.11.4",
				Restart: &fly.MachineRestart{
					MaxRetries: 1,
					Policy:     "no",
				},
			},
		},
	}
	if diff := cmp.Diff(expected, schedules); diff != "" {
		t.Errorf("Schedules mismatch (-want +got):\n%s", diff)
	}
}

func TestSyncSchedules(t *testing.T) {
	log := logrus.New()

	store, err := InitializeStore(context.TODO(), testStorePath, "../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(testStorePath) }()

	t.Run("adds new schedules", func(t *testing.T) {
		schedulesFile, err := createSchedulesFile([]byte(testData))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(schedulesFile.Name()) }()

		if err := SyncSchedules(context.TODO(), store, log, schedulesFile.Name()); err != nil {
			t.Fatal(err)
		}

		schedules, err := store.ListSchedules(context.TODO())
		if err != nil {
			t.Fatal(err)
		}

		if len(schedules) != 2 {
			t.Fatalf("expected 2 schedules, got %d", len(schedules))
		}

		expected := []Schedule{
			{
				ID:             1,
				Name:           "uptime-check",
				AppName:        "shaun-pg-flex",
				Schedule:       "* * * * *",
				Region:         "iad",
				Command:        "uptime",
				CommandTimeout: 60,
				Enabled:        true,
				Config: fly.MachineConfig{
					AutoDestroy: true,
					Guest: &fly.MachineGuest{
						CPUKind:  "shared",
						CPUs:     1,
						MemoryMB: 512,
					},
					Image: "ghcr.io/livebook-dev/livebook:0.11.4",
					Restart: &fly.MachineRestart{
						MaxRetries: 1,
						Policy:     "no",
					},
				},
			},
			{
				ID:             2,
				Name:           "test-check",
				AppName:        "shaun-pg-flex",
				Schedule:       "* * * * *",
				Region:         "iad",
				Command:        "uptime",
				CommandTimeout: 30,
				Enabled:        false,
				Config: fly.MachineConfig{
					AutoDestroy: true,
					Guest: &fly.MachineGuest{
						CPUKind:  "shared",
						CPUs:     1,
						MemoryMB: 512,
					},
					Image: "ghcr.io/livebook-dev/livebook:0.11.4",
					Restart: &fly.MachineRestart{
						MaxRetries: 1,
						Policy:     "no",
					},
				},
			},
		}
		if diff := cmp.Diff(expected, schedules); diff != "" {
			t.Errorf("Schedules mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("updates existing schedules", func(t *testing.T) {
		originalFile, err := createSchedulesFile([]byte(testData))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(originalFile.Name()) }()

		if err := SyncSchedules(context.TODO(), store, log, originalFile.Name()); err != nil {
			t.Fatal(err)
		}

		schedulesFile, err := createSchedulesFile([]byte(testData2))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(schedulesFile.Name()) }()

		if err := SyncSchedules(context.TODO(), store, log, schedulesFile.Name()); err != nil {
			t.Fatal(err)
		}

		schedules, err := store.ListSchedules(context.TODO())
		if err != nil {
			t.Fatal(err)
		}

		if len(schedules) != 2 {
			t.Fatalf("expected 2 schedules, got %d", len(schedules))
		}

		expected := []Schedule{
			{
				ID:             1,
				Name:           "uptime-check",
				AppName:        "shaun-pg-flex",
				Schedule:       "* * * * *",
				Region:         "ord",
				Command:        "uptime",
				CommandTimeout: 60,
				Enabled:        true,
				Config: fly.MachineConfig{
					AutoDestroy: true,
					Guest: &fly.MachineGuest{
						CPUKind:  "shared",
						CPUs:     1,
						MemoryMB: 512,
					},
					Image: "ghcr.io/livebook-dev/livebook:0.11.4",
					Restart: &fly.MachineRestart{
						MaxRetries: 1,
						Policy:     "no",
					},
				},
			},
			{
				ID:             2,
				Name:           "test-check",
				AppName:        "shaun-pg-flex",
				Schedule:       "* * * * *",
				Region:         "ord",
				Command:        "uptime",
				CommandTimeout: 30,
				Enabled:        false,
				Config: fly.MachineConfig{
					AutoDestroy: true,
					Guest: &fly.MachineGuest{
						CPUKind:  "shared",
						CPUs:     1,
						MemoryMB: 512,
					},
					Image: "ghcr.io/livebook-dev/livebook:0.11.4",
					Restart: &fly.MachineRestart{
						MaxRetries: 1,
						Policy:     "no",
					},
				},
			},
		}
		if diff := cmp.Diff(expected, schedules); diff != "" {
			t.Errorf("Schedules mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("removes schedules", func(t *testing.T) {
		schedulesFile, err := createSchedulesFile([]byte(``))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(schedulesFile.Name()) }()

		if err := SyncSchedules(context.TODO(), store, log, schedulesFile.Name()); err != nil {
			t.Fatal(err)
		}

		schedules, err := store.ListSchedules(context.TODO())
		if err != nil {
			t.Fatal(err)
		}

		if len(schedules) != 0 {
			t.Fatalf("expected 0 schedules, got %d", len(schedules))
		}
	})
}

func TestSyncCrontab(t *testing.T) {
	ctx := context.TODO()
	log := logrus.New()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := InitializeStore(ctx, dbPath, "../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	if err := store.CreateSchedule(ctx, Schedule{
		Name:           "test",
		AppName:        "app",
		Schedule:       "* * * * *",
		Command:        "uptime",
		CommandTimeout: 60,
		Region:         "iad",
		Enabled:        true,
		Config:         fly.MachineConfig{},
	}); err != nil {
		t.Fatal(err)
	}

	cronPath := filepath.Join(tmpDir, "crontab")
	t.Run("does not clobber on install failure", func(t *testing.T) {
		if err := os.WriteFile(cronPath, []byte("KNOWN_GOOD\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		runCrontab := func(path string) ([]byte, error) {
			return []byte("bad minute"), fmt.Errorf("exit status 1")
		}

		if err := syncCrontab(ctx, store, log, cronPath, runCrontab); err == nil {
			t.Fatalf("expected error")
		}

		b, err := os.ReadFile(cronPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "KNOWN_GOOD\n" {
			t.Fatalf("expected existing crontab to remain unchanged, got %q", string(b))
		}
	})

	t.Run("replaces on success", func(t *testing.T) {
		runCrontab := func(path string) ([]byte, error) {
			return nil, nil
		}

		if err := syncCrontab(ctx, store, log, cronPath, runCrontab); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		b, err := os.ReadFile(cronPath)
		if err != nil {
			t.Fatal(err)
		}
		got := string(b)
		if got == "" {
			t.Fatalf("expected crontab file to be written")
		}
		if wantSub := executeCommand; !strings.Contains(got, wantSub) {
			t.Fatalf("expected crontab to contain %q, got %q", wantSub, got)
		}
	})
}

func createSchedulesFile(schedules []byte) (*os.File, error) {
	// Write schedules to a temp file
	tmpFile, err := os.CreateTemp("./", "schedules.json")
	if err != nil {
		return nil, err
	}

	if _, err := tmpFile.Write(schedules); err != nil {
		if err := tmpFile.Close(); err != nil {
			return nil, err
		}
		return nil, err
	}

	return tmpFile, nil
}
