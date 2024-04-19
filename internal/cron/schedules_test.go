package cron

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	fly "github.com/superfly/fly-go"

	_ "github.com/mattn/go-sqlite3"
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
