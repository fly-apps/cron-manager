package cron

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

func (c *Client) MachineProvision(ctx context.Context, log *logrus.Entry, schedule *Schedule, job *Job) (*fly.Machine, error) {
	machineConfig := fly.LaunchMachineInput{
		Config: &schedule.Config,
		Region: schedule.Region,
	}

	machine, err := c.flapsClient.Launch(ctx, machineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to launch machine: %w", err)
	}

	log.WithFields(logrus.Fields{
		"schedule-id": schedule.ID,
		"job-id":      job.ID,
		"region":      schedule.Region,
		"image":       schedule.Config.Image,
	}).Info("provisioning machine...")

	if err := c.store.UpdateJobMachine(job.ID, machine.ID); err != nil {
		return machine, fmt.Errorf("failed to update job machine: %w", err)
	}

	return machine, nil
}

func (c *Client) WaitForStatus(ctx context.Context, machine *fly.Machine, targetStatus string) error {
	if err := c.flapsClient.Wait(ctx, machine, targetStatus, 30*time.Second); err != nil {
		return fmt.Errorf("failed to wait for machine to start: %w", err)
	}

	return nil
}

func (c *Client) MachineGet(ctx context.Context, machineID string) (*fly.Machine, error) {
	return c.flapsClient.Get(ctx, machineID)
}

func (c *Client) MachineDestroy(ctx context.Context, machine *fly.Machine) error {
	input := fly.RemoveMachineInput{
		ID:   machine.ID,
		Kill: true,
	}

	if err := c.flapsClient.Destroy(ctx, input, ""); err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil
		}

		return err
	}

	return nil
}

func (c *Client) MachineExec(ctx context.Context, cmd string, machineID string, timeout int) (*fly.MachineExecResponse, error) {
	execReq := &fly.MachineExecRequest{
		Cmd:     cmd,
		Timeout: timeout,
	}
	return c.flapsClient.Exec(ctx, machineID, execReq)
}
