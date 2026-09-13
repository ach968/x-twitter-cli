//go:build live

package test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twitter-cli3/internal/app"
	"github.com/ach968/x-twitter-cli3/internal/app/browser"
	"github.com/ach968/x-twitter-cli3/internal/app/contracts"
	"github.com/ach968/x-twitter-cli3/internal/app/httpclient"
	"github.com/ach968/x-twitter-cli3/internal/app/operations/search"
	"github.com/ach968/x-twitter-cli3/internal/app/state"
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
		t.Logf("[DEBUG-search-flake] status=%d contentType=%q", result.Upstream.Status, contentType)
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

func TestCaptureSearchTimelineEvidence(t *testing.T) {
	if os.Getenv("TWT_CAPTURE_SEARCH_EVIDENCE") != "1" {
		t.Skip("set TWT_CAPTURE_SEARCH_EVIDENCE=1 to save a SearchTimeline source payload")
	}
	query := strings.TrimSpace(os.Getenv("TWT_SEARCH_QUERY"))
	if query == "" {
		t.Fatal("TWT_SEARCH_QUERY is required")
	}
	product := strings.TrimSpace(os.Getenv("TWT_SEARCH_PRODUCT"))
	if product == "" {
		product = "Top"
	}
	cursor := os.Getenv("TWT_SEARCH_CURSOR")
	scenario := strings.TrimSpace(os.Getenv("TWT_SEARCH_SCENARIO"))
	if scenario == "" {
		scenario = strings.ToLower(product) + "-initial"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	paths := state.ResolvePaths(home, currentEnvironment())
	properties, err := contracts.Load(paths.ActiveContractPath)
	if err != nil {
		t.Fatal(err)
	}
	authentication, err := state.LoadAuthentication(paths.AuthenticationPath)
	if err != nil {
		t.Fatal(err)
	}
	generator, err := transaction.New(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	overrides := map[string]any{"product": product}
	page := 1
	if cursor != "" {
		overrides["cursor"] = cursor
		page = 2
	}
	overrides["rawQuery"] = query
	requestClient := httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator)
	result, err := requestClient.Execute(context.Background(), app.SearchTimeline, overrides)
	if err != nil || !result.OK {
		logLiveFailure(t, "SearchTimeline", result, err)
		t.Fatal("SearchTimeline evidence request failed; see safe metadata above")
	}
	path, err := saveSearchTimelineEvidence(paths.EvidenceDirectory, time.Now(), searchTimelineSample{
		Scenario:       scenario,
		Query:          query,
		Product:        product,
		Page:           page,
		CursorProvided: cursor != "",
		Payload:        result.Payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("saved pending SearchTimeline evidence to %s", path)
}

func TestLiveAuthenticatedProfileExecutesAllRequiredOperations(t *testing.T) {
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
				Timeout:     45 * time.Second,
				Steps: []browser.ContractCaptureStep{
					{URL: "https://x.com/search?q=x&src=typed_query", WaitFor: []app.OperationName{app.SearchTimeline}},
					{URL: "https://x.com/i/bookmarks", WaitFor: []app.OperationName{app.Bookmarks}, TriggerBookmarkSearch: true},
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
	t.Run("required_operation_verification", func(t *testing.T) {
		requestClient := httpclient.New(capture.Contracts, capture.Authentication, transport, initialization.generator)
		type outcome struct {
			result   app.OperationResult
			err      error
			duration time.Duration
		}
		searchResults := make(chan outcome, 1)
		bookmarks := make(chan outcome, 1)
		bookmarkSearch := make(chan outcome, 1)
		go func() {
			started := time.Now()
			result, err := requestClient.Execute(context.Background(), app.SearchTimeline, map[string]any{"rawQuery": "x", "product": "Top"})
			searchResults <- outcome{result: result, err: err, duration: time.Since(started)}
		}()
		go func() {
			started := time.Now()
			result, err := requestClient.Execute(context.Background(), app.Bookmarks, nil)
			bookmarks <- outcome{result: result, err: err, duration: time.Since(started)}
		}()
		go func() {
			started := time.Now()
			result, err := requestClient.Execute(context.Background(), app.BookmarkSearchTimeline, map[string]any{"rawQuery": "x-twitter-cli3-contract-validation-improbable-6d1e2f"})
			bookmarkSearch <- outcome{result: result, err: err, duration: time.Since(started)}
		}()
		searchOutcome := <-searchResults
		bookmarksOutcome := <-bookmarks
		bookmarkSearchOutcome := <-bookmarkSearch
		t.Run("search_timeline_over_direct_http", func(t *testing.T) {
			// t.Logf("completed in %s", searchOutcome.duration)
			if searchOutcome.err != nil || !searchOutcome.result.OK {
				logLiveFailure(t, "SearchTimeline", searchOutcome.result, searchOutcome.err)
				t.Fatal("SearchTimeline failed; see safe metadata above")
			}
		})
		t.Run("bookmarks_over_direct_http", func(t *testing.T) {
			if bookmarksOutcome.err != nil || !bookmarksOutcome.result.OK {
				logLiveFailure(t, "Bookmarks", bookmarksOutcome.result, bookmarksOutcome.err)
				t.Fatal("Bookmarks failed; see safe metadata above")
			}
		})
		t.Run("bookmark_search_over_direct_http", func(t *testing.T) {
			if bookmarkSearchOutcome.err != nil || !bookmarkSearchOutcome.result.OK {
				logLiveFailure(t, "BookmarkSearchTimeline", bookmarkSearchOutcome.result, bookmarkSearchOutcome.err)
				t.Fatal("BookmarkSearchTimeline failed; see safe metadata above")
			}
		})
		t.Run("search_timeline_decodes_without_printing_source", func(t *testing.T) {
			if searchOutcome.err != nil || !searchOutcome.result.OK {
				t.Skip("SearchTimeline did not return a successful source payload")
			}
			page, err := search.DecodePage(searchOutcome.result.Payload, "x", search.TabTop)
			if err != nil {
				t.Fatalf("SearchTimeline payload is incompatible with the decoder: %v", err)
			}
			if _, err := json.Marshal(page); err != nil {
				t.Fatalf("normalized search page cannot be encoded: %v", err)
			}
		})
	})
}
