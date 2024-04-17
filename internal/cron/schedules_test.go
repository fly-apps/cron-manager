package cron

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const (
	testStorePath  = "../../test/test.db"
	migrationsPath = "../../migrations"
	schedulesPath  = "../../schedules.json"
)

func TestReadSchedulesFromFile(t *testing.T) {
	rawSchedule := `[
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
            "disable_machine_autostart": true,
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
            "disable_machine_autostart": true,
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
	schedulesFile, err := createSchedulesFile([]byte(rawSchedule))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = schedulesFile.Close() }()
	defer func() { _ = os.Remove(schedulesFile.Name()) }()

	schedules, err := readSchedulesFromFile(schedulesFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(schedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(schedules))
	}

	schedule := schedules[0]

	if schedule.Name != "uptime-check" {
		t.Errorf("expected schedule name to be uptime-check, got %s", schedule.Name)
	}

	if schedule.AppName != "shaun-pg-flex" {
		t.Errorf("expected app name to be shaun-pg-flex, got %s", schedule.AppName)
	}

	if schedule.Enabled != true {
		t.Errorf("expected enabled to be true, got %t", schedule.Enabled)
	}

	if schedule.Schedule != "* * * * *" {
		t.Errorf("expected schedule to be * * * * *, got %s", schedule.Schedule)
	}

	if schedule.Region != "iad" {
		t.Errorf("expected region to be iad, got %s", schedule.Region)
	}

	if schedule.Command != "uptime" {
		t.Errorf("expected command to be uptime, got %s", schedule.Command)
	}

	if schedule.CommandTimeout != 60 {
		t.Errorf("expected command timeout to be 60, got %d", schedule.CommandTimeout)
	}

	if schedule.Config.Image != "ghcr.io/livebook-dev/livebook:0.11.4" {
		t.Errorf("expected image to be ghcr.io/livebook-dev/livebook:0.11.4, got %s", schedule.Config.Image)
	}

	if !schedule.Config.AutoDestroy {
		t.Errorf("expected auto destroy to be true, got %t", schedule.Config.AutoDestroy)
	}

	if schedule.Config.Guest.MemoryMB != 512 {
		t.Errorf("expected memory to be 512, got %d", schedule.Config.Guest.MemoryMB)
	}

	schedule = schedules[1]

	if schedule.Name != "test-check" {
		t.Errorf("expected schedule name to be test-check, got %s", schedule.Name)
	}

	if schedule.CommandTimeout != 0 {
		t.Errorf("expected command timeout to be 30, got %d", schedule.CommandTimeout)
	}

	if schedule.Enabled != false {
		t.Errorf("expected enabled to be false, got %t", schedule.Enabled)
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
