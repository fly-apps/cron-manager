package flycheck

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/superfly/fly-checks/check"
)

func Handler() http.Handler {
	r := http.NewServeMux()

	r.HandleFunc("/flycheck/cron", runCronChecks)

	return r
}

func runCronChecks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (5 * time.Second))
	defer cancel()

	suite := &check.CheckSuite{Name: "cron"}
	suite, err := CheckCron(ctx, suite)
	if err != nil {
		handleError(w, err)
		return
	}

	go func(ctx context.Context) {
		suite.Process(ctx)
		cancel()
	}(ctx)

	<-ctx.Done()

	handleCheckResponse(w, suite, false)
}

func handleCheckResponse(w http.ResponseWriter, suite *check.CheckSuite, raw bool) {
	if suite.ErrOnSetup != nil {
		handleError(w, suite.ErrOnSetup)
		return
	}
	var result string
	if raw {
		result = suite.RawResult()
	} else {
		result = suite.Result()
	}
	if !suite.Passed() {
		handleError(w, errors.New(result))
		return
	}
	if _, err := io.WriteString(w, result); err != nil {
		log.Printf("failed to handle check response: %s", err)
	}
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if _, err := io.WriteString(w, err.Error()); err != nil {
		log.Printf("failed to handle check error: %s", err)
	}
}
