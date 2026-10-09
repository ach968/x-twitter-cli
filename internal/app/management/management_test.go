package management_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/browser"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	"github.com/ach968/x-twitter-cli/internal/app/management"
	"github.com/ach968/x-twitter-cli/internal/app/state"
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
		NewTransactionIDs: func(context.Context, app.AuthenticationState) (app.TransactionIDGenerator, error) {
			return transactionIDs(func(string, string) (string, error) { return "generated", nil }), nil
		},
	}
}

type successfulTransport struct{}

func (successfulTransport) Execute(_ context.Context, operation app.OperationName, _ app.PreparedRequest) (app.UpstreamResponse, error) {
	if operation == app.TweetDetail {
		return app.UpstreamResponse{Status: 200, Body: `{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineAddEntries","entries":[{"entryId":"tweet-100","content":{"entryType":"TimelineTimelineItem","itemContent":{"itemType":"TimelineTweet","tweet_results":{"result":{"rest_id":"100","legacy":{"full_text":"synthetic"}}}}}}]}]}}}`}, nil
	}
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
			app.TweetDetail: {Family: "graphql", Host: "x.com", Path: "/i/api/graphql/view/TweetDetail", Method: "GET", Encoding: "query", Variables: map[string]any{"focalTweetId": "100"}, Features: map[string]any{}, FieldToggles: map[string]any{}},
		},
		},
		Authentication: app.AuthenticationState{
			Cookies:       []app.AuthenticationCookie{{Name: "auth_token", Value: "auth"}, {Name: "ct0", Value: "csrf"}},
			Authorization: "Bearer public-client-token",
		},
	}
}

func TestRefreshCannotDropViewFromNewCapture(t *testing.T) {
	paths := testPaths(t)
	options := serviceOptions(paths)
	options.Capture = func(browser.ContractCaptureOptions) (app.CapturedState, error) {
		captured := capturedState()
		delete(captured.Contracts.Operations, app.TweetDetail)
		return captured, nil
	}
	_, err := management.New(options).RefreshContracts(context.Background())
	if err == nil {
		t.Fatal("refresh accepted a capture without View")
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
		if !options.Headless || !options.Login || options.ProfilePath != paths.ProfilePath || options.Context == nil || options.Timeout != time.Minute {
			t.Fatalf("capture options = %#v", options)
		}
		if len(options.Steps) != 2 || options.Steps[1].URL != "https://x.com/i/bookmarks" || !options.Steps[1].TriggerBookmarkSearch {
			t.Fatalf("bookmark capture recipe = %#v", options.Steps)
		}
		if !options.Steps[0].TriggerPostView {
			t.Fatal("View was omitted from login capture")
		}
		return capturedState(), nil
	}
	service := management.New(options)

	result, err := service.Login(context.Background(), func(management.BrowserSetupPrompt) (bool, error) { return true, nil }, management.LoginOptions{Headless: true})
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

func TestLoginUsesSelectedBrowserOnce(t *testing.T) {
	for _, headless := range []bool{true, false} {
		t.Run(map[bool]string{true: "headless", false: "headed"}[headless], func(t *testing.T) {
			options := serviceOptions(testPaths(t))
			ctx := context.WithValue(context.Background(), struct{}{}, "test")
			captureCalls := 0
			promptCalls := 0
			prompt := func(browser.LoginPrompt) (string, error) { promptCalls++; return "user", nil }
			options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
				captureCalls++
				if input.Headless != headless || !input.Login || input.Context != ctx {
					t.Fatalf("capture options=%#v", input)
				}
				if value, err := input.LoginPrompt(browser.LoginPrompt{Label: "Username: "}); err != nil || value != "user" {
					t.Fatalf("prompt value=%q error=%v", value, err)
				}
				return capturedState(), nil
			}
			result, err := management.New(options).Login(ctx, func(management.BrowserSetupPrompt) (bool, error) { return true, nil }, management.LoginOptions{Headless: headless, Prompt: prompt})
			if err != nil || result.Status != "authenticated" || captureCalls != 1 || promptCalls != 1 {
				t.Fatalf("result=%#v err=%v captures=%d prompts=%d", result, err, captureCalls, promptCalls)
			}
		})
	}
}

func TestLoginFailureDoesNotChangeBrowserModeOrStoredState(t *testing.T) {
	for _, failure := range []error{errors.New("capture transport failed"), &browser.AuthenticationRequiredError{URL: "https://x.com/i/flow/login"}} {
		paths := testPaths(t)
		if err := state.SaveCaptured(paths.ActiveContractPath, paths.AuthenticationPath, capturedState()); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(paths.AuthenticationPath)
		if err != nil {
			t.Fatal(err)
		}
		options := serviceOptions(paths)
		captureCalls := 0
		options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
			captureCalls++
			if !input.Headless {
				t.Fatal("failure opened a headed browser")
			}
			return app.CapturedState{}, failure
		}
		_, err = management.New(options).Login(context.Background(), func(management.BrowserSetupPrompt) (bool, error) { return true, nil }, management.LoginOptions{Headless: true})
		if err == nil || captureCalls != 1 {
			t.Fatalf("error=%v captures=%d", err, captureCalls)
		}
		after, err := os.ReadFile(paths.AuthenticationPath)
		if err != nil || string(after) != string(before) {
			t.Fatal("failed login changed authentication")
		}
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

func TestContractRefreshRequiresExplicitLoginAndNeverFallsBack(t *testing.T) {
	options := serviceOptions(testPaths(t))
	captureCalls := 0
	options.Capture = func(input browser.ContractCaptureOptions) (app.CapturedState, error) {
		captureCalls++
		if !input.Headless || input.Login || input.LoginPrompt != nil {
			t.Fatalf("refresh login options=%#v", input)
		}
		return app.CapturedState{}, &browser.AuthenticationRequiredError{URL: "https://x.com/i/flow/login"}
	}
	_, err := management.New(options).RefreshContracts(context.Background())
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != "AUTHENTICATION_REQUIRED" || failure.RecoveryCommand != "twt auth login" {
		t.Fatalf("error=%v", err)
	}
	if captureCalls != 1 {
		t.Fatalf("capture calls=%d", captureCalls)
	}
}

func TestContractRefreshUsesFreshCapturedAuthenticationForTransactionIDs(t *testing.T) {
	paths := testPaths(t)
	options := serviceOptions(paths)
	options.Capture = func(browser.ContractCaptureOptions) (app.CapturedState, error) {
		return capturedState(), nil
	}
	options.NewTransactionIDs = func(_ context.Context, auth app.AuthenticationState) (app.TransactionIDGenerator, error) {
		if len(auth.Cookies) != 2 || auth.Cookies[0].Value != "auth" || auth.Cookies[1].Value != "csrf" {
			t.Fatal("generator did not receive freshly captured authentication")
		}
		return transactionIDs(func(string, string) (string, error) { return "generated", nil }), nil
	}
	if _, err := management.New(options).RefreshContracts(context.Background()); err != nil {
		t.Fatal(err)
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

func TestRefreshCapturePlanCoversCatalogRequirements(t *testing.T) {
	options := serviceOptions(testPaths(t))
	called := false
	options.Capture = func(options browser.ContractCaptureOptions) (app.CapturedState, error) {
		called = true
		covered := map[app.OperationName]bool{}
		for _, step := range options.Steps {
			if step.URL == "" {
				t.Error("capture step has no navigation")
			}
			for _, name := range step.CapturedOperations() {
				if _, ok := app.LookupOperation(name); !ok {
					t.Errorf("capture plan includes unsupported operation %s", name)
				}
				covered[name] = true
			}
		}
		for _, policy := range app.OperationPolicies() {
			if policy.Refresh == app.Required && !covered[policy.Name] {
				t.Errorf("required operation %s has no capture step", policy.Name)
			}
		}
		return capturedState(), nil
	}
	if _, err := management.New(options).RefreshContracts(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("refresh did not use the capture plan")
	}
}
