//go:build live

package test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/browser"
	"github.com/ach968/x-twt-cli/internal/app/httpclient"
	"github.com/ach968/x-twt-cli/internal/app/state"
)

func logLiveFailure(t *testing.T, operation string, result app.OperationResult, err error) {
	t.Helper()
	t.Logf("[DEBUG-search-flake] operation=%s transportError=%v", operation, err)
	if result.Error != nil {
		t.Logf("[DEBUG-search-flake] code=%s message=%q recovery=%q", result.Error.Code, result.Error.Message, result.Error.RecoveryCommand)
	}
	if result.Upstream != nil {
		contentType := ""
		if result.Upstream.ContentType != nil {
			contentType = *result.Upstream.ContentType
		}
		body := result.Upstream.Body
		if len(body) > 600 {
			body = body[:600]
		}
		t.Logf("[DEBUG-search-flake] status=%d contentType=%q body=%q", result.Upstream.Status, contentType, body)
	}
}

func currentEnvironment() map[string]string {
	result := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			result[key] = value
		}
	}
	return result
}

func TestLiveAuthenticatedProfileExecutesBothOperations(t *testing.T) {
	if os.Getenv("TWT_LIVE_X") != "1" {
		t.Skip("set TWT_LIVE_X=1 to contact X with the local application profile")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	paths := state.ResolvePaths(home, currentEnvironment())
	var capture app.CapturedState
	type transactionIDInitialization struct {
		generator app.TransactionIDGenerator
		err       error
		duration  time.Duration
	}
	initializationContext, cancelInitialization := context.WithCancel(context.Background())
	defer cancelInitialization()
	transactionInitializationResult := make(chan transactionIDInitialization, 1)
	go func() {
		started := time.Now()
		generator, err := transaction.New(initializationContext, nil)
		transactionInitializationResult <- transactionIDInitialization{
			generator: generator,
			err:       err,
			duration:  time.Since(started),
		}
	}()

	browserActivated := t.Run("browser_activation", func(t *testing.T) {
		if !t.Run("capture_contracts_and_authentication", func(t *testing.T) {
			var err error
			capture, err = browser.CaptureOperationContracts(browser.ContractCaptureOptions{
				ProfilePath: paths.ProfilePath,
				Headless:    true,
				Timeout:     10 * time.Minute,
				Steps: []browser.ContractCaptureStep{
					{URL: "https://x.com/home", WaitFor: []app.OperationName{app.HomeTimeline}},
					{URL: "https://x.com/search?q=x&src=typed_query", WaitFor: []app.OperationName{app.SearchTimeline}},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
		}) {
			return
		}
		t.Run("persist_captured_state", func(t *testing.T) {
			if err := state.SaveCaptured(paths.CandidateContractPath, paths.AuthenticationPath, capture); err != nil {
				t.Fatal(err)
			}
		})
	})
	if !browserActivated {
		cancelInitialization()
		<-transactionInitializationResult
		return
	}
	initialization := <-transactionInitializationResult
	if initialization.err != nil {
		t.Fatal(initialization.err)
	}
	// t.Logf("transaction ID generator initialized in %s", initialization.duration)
	transport := httpclient.NewTransport(nil)
	t.Run("home_and_search_verification", func(t *testing.T) {
		requestClient := httpclient.New(capture.Contracts, capture.Authentication, transport, initialization.generator)
		type outcome struct {
			result   app.OperationResult
			err      error
			duration time.Duration
		}
		home := make(chan outcome, 1)
		search := make(chan outcome, 1)
		go func() {
			started := time.Now()
			result, err := requestClient.HomeTimeline(context.Background(), nil)
			home <- outcome{result: result, err: err, duration: time.Since(started)}
		}()
		go func() {
			started := time.Now()
			result, err := requestClient.SearchTimeline(context.Background(), "x", nil)
			search <- outcome{result: result, err: err, duration: time.Since(started)}
		}()
		homeOutcome := <-home
		searchOutcome := <-search
		t.Run("home_timeline_over_direct_http", func(t *testing.T) {
			// t.Logf("completed in %s", homeOutcome.duration)
			if homeOutcome.err != nil || !homeOutcome.result.OK {
				logLiveFailure(t, "HomeTimeline", homeOutcome.result, homeOutcome.err)
				t.Fatalf("home failed: %#v %v", homeOutcome.result, homeOutcome.err)
			}
		})
		t.Run("search_timeline_over_direct_http", func(t *testing.T) {
			// t.Logf("completed in %s", searchOutcome.duration)
			if searchOutcome.err != nil || !searchOutcome.result.OK {
				logLiveFailure(t, "SearchTimeline", searchOutcome.result, searchOutcome.err)
				t.Fatalf("search failed: %#v %v", searchOutcome.result, searchOutcome.err)
			}
		})
	})
}
