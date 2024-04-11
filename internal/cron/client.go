package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	fly "github.com/superfly/fly-go"
	"github.com/superfly/fly-go/flaps"
	"github.com/superfly/fly-go/tokens"
)

const (
	apiEndpoint = "https://api.machines.dev/v1"
)

type Client struct {
	appName     string
	flapsClient *flaps.Client
	store       *Store
}

func NewClient(ctx context.Context, appName string, store *Store) (*Client, error) {
	flapsClient, err := flaps.NewWithOptions(ctx, flaps.NewClientOpts{
		AppName: appName,
		Tokens: &tokens.Tokens{
			UserTokens: []string{os.Getenv("FLY_API_TOKEN")},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create flaps client: %w", err)
	}

	return &Client{
		appName:     appName,
		flapsClient: flapsClient,
		store:       store,
	}, nil
}

func (c *Client) MachineProvision(ctx context.Context, cronjob *CronJob, job *Job) (*fly.Machine, error) {
	machineConfig := fly.LaunchMachineInput{
		Config: &fly.MachineConfig{
			Guest: &fly.MachineGuest{
				CPUKind:  "shared",
				CPUs:     1,
				MemoryMB: 1024,
			},
			Image: cronjob.Image,
			Restart: &fly.MachineRestart{
				MaxRetries: 1,
				Policy:     fly.MachineRestartPolicy(cronjob.RestartPolicy),
			},
		},
		Region: "ord",
	}

	machine, err := c.flapsClient.Launch(ctx, machineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to launch machine: %w", err)
	}

	if err := c.store.UpdateJobMachine(job.ID, machine.ID); err != nil {
		return machine, fmt.Errorf("failed to update job machine: %w", err)
	}

	log.Printf("Machine %s created under App %s", machine.ID, cronjob.AppName)

	if err := c.flapsClient.Wait(ctx, machine, fly.MachineStateStarted, 30*time.Second); err != nil {
		return machine, fmt.Errorf("failed to wait for machine to start: %w", err)
	}

	return machine, nil
}

func (c *Client) MachineDestroy(ctx context.Context, machine *fly.Machine) error {
	input := fly.RemoveMachineInput{
		ID:   machine.ID,
		Kill: true,
	}

	if err := c.flapsClient.Destroy(ctx, input, ""); err != nil {
		return fmt.Errorf("failed to destroy machine: %w", err)
	}

	return nil
}

func (c *Client) MachineExec(ctx context.Context, cronJob *CronJob, job *Job, machine *fly.Machine) (*fly.MachineExecResponse, error) {
	execReq := &fly.MachineExecRequest{
		Cmd: cronJob.Command,
	}
	return c.flapsClient.Exec(ctx, machine.ID, execReq)
}
