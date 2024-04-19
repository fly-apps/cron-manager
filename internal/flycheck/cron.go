package flycheck

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/superfly/fly-checks/check"
)

func CheckCron(ctx context.Context, checks *check.CheckSuite) (*check.CheckSuite, error) {
	checks.AddCheck("cron", func() (string, error) {
		return cronStatus()
	})
	return checks, nil
}

func cronStatus() (string, error) {
	cmd := exec.Command("service", "cron", "status")
	result, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to check cron status: %w", err)
	}

	resultStr := string(result)

	if strings.Contains(resultStr, "is running") {
		return "running", nil
	}

	return "", fmt.Errorf(resultStr)
}
