package management_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twitter-cli3/internal/app"
	"github.com/ach968/x-twitter-cli3/internal/app/browser"
	"github.com/ach968/x-twitter-cli3/internal/app/contracts"
	"github.com/ach968/x-twitter-cli3/internal/app/management"
	"github.com/ach968/x-twitter-cli3/internal/app/state"
)

type transactionIDs func(string, string) (string, error)

func (generate transactionIDs) Generate(method, path string) (string, error) {
	return generate(method, path)
}

func serviceOptions(paths state.StatePaths) management.Options {
	return management.Options{
		Paths: paths,
		EnsureChromium: func(browser.ChromiumSetupOptions) (browser.ChromiumSetupResult, error) {
			return browser.ChromiumSetupResult{Status: "ready", ExecutablePath: "/managed/chromium"}, nil
		},
		Transport: successfulTransport{},
		NewTransactionIDs: func(context.Context) (app.TransactionIDGenerator, error) {
			return transactionIDs(func(string, string) (string, error) { return "generated", nil }), nil
		},
	}
}

type successfulTransport struct{}

func (successfulTransport) Execute(context.Context, app.OperationName, app.PreparedRequest) (app.UpstreamResponse, error) {
	return app.UpstreamResponse{Status: 200, Body: `{"data":{"ok":true}}`}, nil
}

func capturedState() app.CapturedState {
	return app.CapturedState{
		Contracts: app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{
			app.HomeTimeline: {
				Family: "graphql", Host: "x.com", Path: "/i/api/graphql/home/HomeTimeline", Method: "GET", Encoding: "query",
				Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
			},
			app.SearchTimeline: {
				Family: "graphql", Host: "x.com", Path: "/i/api/graphql/search/SearchTimeline", Method: "GET", Encoding: "query",
				Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
			},
			app.Bookmarks: {
				Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmarks/Bookmarks", Method: "GET", Encoding: "query",
				Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
			},
			app.BookmarkSearchTimeline: {
				Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmark-search/BookmarkSearchTimeline", Method: "GET", Encoding: "query",
				Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
			},
		},
		},
		Authentication: app.AuthenticationState{
			Cookies:       []app.AuthenticationCookie{{Name: "auth_token", Value: "auth"}, {Name: "ct0", Value: "csrf"}},
			Authorization: "Bearer public-client-token",
		},
	}
}

func testPaths(t *testing.T) state.StatePaths {
	t.Helper()
	return state.ResolvePaths(t.TempDir(), map[string]string{})
}

func TestLoginCapturesPersistsAndActivatesState(t *testing.T) {
	paths := testPaths(t)
	captureCalls := 0
	options := serviceOptions(paths)
	options.Capture = func(options browser.ContractCaptureOptions) (app.CapturedState, error) {
		captureCalls++
		if !options.Headless || options.ProfilePath != paths.ProfilePath {
			t.Fatalf("capture options = %#v", options)
		}
		if len(options.Steps) != 2 || options.Steps[1].URL != "https://x.com/i/bookmarks" || !options.Steps[1].TriggerBookmarkSearch {
			t.Fatalf("bookmark capture recipe = %#v", options.Steps)
		}
		return capturedState(), nil
	}
	service := management.New(options)

	result, err := service.Login(context.Background(), func(management.BrowserSetupPrompt) (bool, error) { return true, nil })
	if err != nil || result.Status != "authenticated" || result.AuthenticationPath != paths.AuthenticationPath || result.ContractPath != paths.ActiveContractPath {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if captureCalls != 1 {
		t.Fatalf("capture calls=%d", captureCalls)
	}
	if _, err := state.LoadAuthentication(paths.AuthenticationPath); err != nil {
		t.Fatalf("authentication was not persisted: %v", err)
	}
	if _, err := contracts.Load(paths.ActiveContractPath); err != nil {
		t.Fatalf("contracts were not activated: %v", err)
	}
	if _, err := os.Stat(paths.CandidateContractPath); !os.IsNotExist(err) {
		t.Fatalf("candidate remains after activation: %v", err)
	}
	if filepath.Dir(paths.ActiveContractPath) == "." {
		t.Fatal("test did not exercise nested state directories")
	}
}

func TestLoginUsesHeadedBrowserOnlyWhenAuthenticationIsRequired(t *testing.T) {
	paths := testPaths(t)
	options := serviceOptions(paths)
	var headlessValues []bool
	options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
		headlessValues = append(headlessValues, input.Headless)
		if input.Headless {
			return app.CapturedState{}, &browser.AuthenticationRequiredError{URL: "https://x.com/i/flow/login"}
		}
		return capturedState(), nil
	}
	service := management.New(options)
	result, err := service.Login(context.Background(), func(management.BrowserSetupPrompt) (bool, error) { return true, nil })
	if err != nil || result.Status != "authenticated" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if len(headlessValues) != 2 || !headlessValues[0] || headlessValues[1] {
		t.Fatalf("headless sequence=%v", headlessValues)
	}

	unknownFailure := errors.New("capture transport failed")
	headlessValues = nil
	options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
		headlessValues = append(headlessValues, input.Headless)
		return app.CapturedState{}, unknownFailure
	}
	service = management.New(options)
	if _, err := service.Login(context.Background(), func(management.BrowserSetupPrompt) (bool, error) { return true, nil }); !errors.Is(err, unknownFailure) {
		t.Fatalf("error=%v", err)
	}
	if len(headlessValues) != 1 || !headlessValues[0] {
		t.Fatalf("unknown failure unexpectedly opened headed browser: %v", headlessValues)
	}
}

func TestSetupReportsManagedChromiumOutcome(t *testing.T) {
	paths := testPaths(t)
	options := serviceOptions(paths)
	options.ExecutablePath = "/explicit/chromium"
	options.EnsureChromium = func(input browser.ChromiumSetupOptions) (browser.ChromiumSetupResult, error) {
		if input.ExecutablePath != "/explicit/chromium" {
			t.Fatalf("executable=%q", input.ExecutablePath)
		}
		confirmed, err := input.ConfirmInstall(browser.ChromiumSetupPrompt{Action: "update", RequiredRevision: "123", InstalledRevisions: []string{"122"}})
		if err != nil || !confirmed {
			t.Fatalf("confirmed=%v err=%v", confirmed, err)
		}
		return browser.ChromiumSetupResult{Status: "ready", ExecutablePath: "/explicit/chromium", Action: "update"}, nil
	}
	service := management.New(options)
	result, err := service.Setup(context.Background(), func(prompt management.BrowserSetupPrompt) (bool, error) {
		if prompt.Action != "update" || prompt.RequiredRevision != "123" || len(prompt.InstalledRevisions) != 1 {
			t.Fatalf("prompt=%#v", prompt)
		}
		return true, nil
	})
	if err != nil || result.Status != "ready" || result.ExecutablePath != "/explicit/chromium" || result.Action != "update" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestContractRefreshUsesHeadedBrowserOnlyWhenAuthenticationIsRequired(t *testing.T) {
	paths := testPaths(t)
	options := serviceOptions(paths)
	var headlessValues []bool
	options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
		headlessValues = append(headlessValues, input.Headless)
		if input.Headless {
			return app.CapturedState{}, &browser.AuthenticationRequiredError{URL: "https://x.com/i/flow/login"}
		}
		return capturedState(), nil
	}
	service := management.New(options)
	result, err := service.RefreshContracts(context.Background())
	if err != nil || result.Status != "refreshed" || result.ContractPath != paths.ActiveContractPath {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if len(headlessValues) != 2 || !headlessValues[0] || headlessValues[1] {
		t.Fatalf("headless sequence=%v", headlessValues)
	}

	unknownFailure := errors.New("capture transport failed")
	headlessValues = nil
	options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
		headlessValues = append(headlessValues, input.Headless)
		return app.CapturedState{}, unknownFailure
	}
	service = management.New(options)
	if _, err := service.RefreshContracts(context.Background()); !errors.Is(err, unknownFailure) {
		t.Fatalf("error=%v", err)
	}
	if len(headlessValues) != 1 || !headlessValues[0] {
		t.Fatalf("unknown failure unexpectedly opened headed browser: %v", headlessValues)
	}
}

func TestContractStatusInspectsLocalFilesWithoutCapturing(t *testing.T) {
	paths := testPaths(t)
	options := serviceOptions(paths)
	options.Capture = func(browser.ContractCaptureOptions) (app.CapturedState, error) {
		t.Fatal("status launched a browser")
		return app.CapturedState{}, nil
	}
	service := management.New(options)
	missing := service.ContractStatus(context.Background())
	if missing.LocallyConfigured || missing.Authentication.Status != "missing" || missing.Contracts.Status != "missing" {
		t.Fatalf("missing status=%#v", missing)
	}

	if err := state.SaveCaptured(paths.CandidateContractPath, paths.AuthenticationPath, capturedState()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(paths.CandidateContractPath, paths.ActiveContractPath); err != nil {
		t.Fatal(err)
	}
	ready := service.ContractStatus(context.Background())
	if !ready.LocallyConfigured || ready.Authentication.Status != "stored" || ready.Contracts.Status != "valid" {
		t.Fatalf("ready status=%#v", ready)
	}
}
