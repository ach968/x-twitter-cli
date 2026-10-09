package management

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/browser"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	"github.com/ach968/x-twitter-cli/internal/app/state"
	"github.com/ach968/x-twitter-cli/internal/app/workflow"
)

type Options struct {
	Paths             state.StatePaths
	ExecutablePath    string
	EnsureChromium    func(browser.ChromiumSetupOptions) (browser.ChromiumSetupResult, error)
	Capture           func(browser.ContractCaptureOptions) (app.CapturedState, error)
	Transport         app.XTransport
	NewTransactionIDs func(context.Context, app.AuthenticationState) (app.TransactionIDGenerator, error)
}

type Manager struct {
	options Options
}

func New(options Options) *Manager {
	if options.EnsureChromium == nil {
		options.EnsureChromium = browser.EnsureChromiumAvailable
	}
	if options.Capture == nil {
		options.Capture = browser.CaptureOperationContracts
	}
	if options.Transport == nil {
		options.Transport = httpclient.NewTransport(nil)
	}
	if options.NewTransactionIDs == nil {
		options.NewTransactionIDs = httpclient.NewTransactionIDGenerator
	}
	return &Manager{options: options}
}

func NewLocal() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	environment := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_STATE_HOME":  os.Getenv("XDG_STATE_HOME"),
	}
	return New(Options{
		Paths:          state.ResolvePaths(home, environment),
		ExecutablePath: os.Getenv("TWT_CHROMIUM_EXECUTABLE"),
	}), nil
}

func (manager *Manager) Setup(_ context.Context, confirm ConfirmBrowserSetup) (SetupResult, error) {
	result, err := manager.ensureChromium(confirm)
	if err != nil {
		return SetupResult{}, err
	}
	return SetupResult{
		Status: result.Status, ExecutablePath: result.ExecutablePath,
		Action: result.Action, InstallCommand: result.InstallCommand,
	}, nil
}

func (manager *Manager) Login(ctx context.Context, confirm ConfirmBrowserSetup, options LoginOptions) (StateChangeResult, error) {
	setup, err := manager.ensureChromium(confirm)
	if err != nil {
		return StateChangeResult{}, err
	}
	if setup.Status != "ready" {
		return StateChangeResult{}, fmt.Errorf("managed Chromium %s was declined; run %s", setup.Action, setup.InstallCommand)
	}
	capture, err := manager.capture(ctx, options.Headless, true, options.Prompt)
	if err != nil {
		return StateChangeResult{}, err
	}
	generator, err := manager.options.NewTransactionIDs(ctx, capture.Authentication)
	if err != nil {
		return StateChangeResult{}, err
	}
	if err := manager.persistAndActivate(ctx, capture, generator); err != nil {
		return StateChangeResult{}, err
	}
	return StateChangeResult{
		Status: "authenticated", AuthenticationPath: manager.options.Paths.AuthenticationPath,
		ContractPath: manager.options.Paths.ActiveContractPath,
	}, nil
}

func (manager *Manager) RefreshContracts(ctx context.Context) (StateChangeResult, error) {
	capture, err := manager.capture(ctx, true, false, nil)
	if err != nil {
		return StateChangeResult{}, err
	}
	generator, err := manager.options.NewTransactionIDs(ctx, capture.Authentication)
	if err != nil {
		return StateChangeResult{}, err
	}
	if err := manager.persistAndActivate(ctx, capture, generator); err != nil {
		return StateChangeResult{}, err
	}
	return StateChangeResult{Status: "refreshed", ContractPath: manager.options.Paths.ActiveContractPath}, nil
}

func (manager *Manager) ContractStatus(context.Context) ContractStatus {
	authenticationStatus := fileStatus(manager.options.Paths.AuthenticationPath, "stored", func() error {
		_, err := state.LoadAuthentication(manager.options.Paths.AuthenticationPath)
		return err
	})
	contractStatus := fileStatus(manager.options.Paths.ActiveContractPath, "valid", func() error {
		_, err := contracts.Load(manager.options.Paths.ActiveContractPath)
		return err
	})
	return ContractStatus{
		LocallyConfigured: authenticationStatus.Status == "stored" && contractStatus.Status == "valid",
		Authentication:    authenticationStatus,
		Contracts:         contractStatus,
	}
}

func (manager *Manager) ensureChromium(confirm ConfirmBrowserSetup) (browser.ChromiumSetupResult, error) {
	if confirm == nil {
		return browser.ChromiumSetupResult{}, errors.New("browser setup confirmation is required")
	}
	return manager.options.EnsureChromium(browser.ChromiumSetupOptions{
		ExecutablePath: manager.options.ExecutablePath,
		ConfirmInstall: func(prompt browser.ChromiumSetupPrompt) (bool, error) {
			return confirm(BrowserSetupPrompt{
				Action: prompt.Action, RequiredRevision: prompt.RequiredRevision,
				InstalledRevisions: prompt.InstalledRevisions,
			})
		},
		ConfirmCleanup: func(prompt browser.ChromiumSetupPrompt) (bool, error) {
			return confirm(BrowserSetupPrompt{
				Action: prompt.Action, RequiredRevision: prompt.RequiredRevision,
				InstalledRevisions: prompt.InstalledRevisions,
			})
		},
	})
}

func (manager *Manager) capture(ctx context.Context, headless, login bool, prompt browser.LoginPromptFunc) (app.CapturedState, error) {
	capture, err := manager.options.Capture(browser.ContractCaptureOptions{
		Context:     ctx,
		ProfilePath: manager.options.Paths.ProfilePath,
		Headless:    headless,
		Login:       login,
		LoginPrompt: prompt,
		Timeout:     time.Minute,
		Steps: []browser.ContractCaptureStep{
			{URL: "https://x.com/search?q=x&src=typed_query", WaitFor: []app.OperationName{app.SearchTimeline}, TriggerPostView: true},
			{URL: "https://x.com/i/bookmarks", WaitFor: []app.OperationName{app.Bookmarks}, TriggerBookmarkSearch: true},
		},
	})
	var authenticationRequired *browser.AuthenticationRequiredError
	if errors.As(err, &authenticationRequired) {
		return app.CapturedState{}, &app.OperationFailure{
			Code:            "AUTHENTICATION_REQUIRED",
			Message:         "X authentication is required; run twt auth login or twt auth login --headed",
			RecoveryCommand: "twt auth login",
		}
	}
	return capture, err
}

func (manager *Manager) persistAndActivate(ctx context.Context, capture app.CapturedState, generator app.TransactionIDGenerator) error {
	for _, policy := range app.OperationPolicies() {
		if _, present := capture.Contracts.Operations[policy.Name]; policy.Refresh == app.Required && !present {
			return contracts.UnavailableFailure()
		}
	}
	paths := manager.options.Paths
	if err := state.SaveCaptured(paths.CandidateContractPath, paths.AuthenticationPath, capture); err != nil {
		return err
	}
	defer os.Remove(paths.CandidateContractPath)
	result, err := workflow.ValidateAndActivateCandidate(ctx, paths.CandidateContractPath, paths.ActiveContractPath, capture.Authentication, manager.options.Transport, generator)
	if err != nil {
		return err
	}
	if !result.Activated {
		if result.Failure != nil {
			return fmt.Errorf("captured operation contracts failed validation: %s", result.Failure.Message)
		}
		return errors.New("captured operation contracts failed validation")
	}
	return nil
}

func fileStatus(path, validStatus string, validate func() error) FileStatus {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FileStatus{Status: "missing", Path: path}
		}
		return FileStatus{Status: "unreadable", Path: path}
	}
	if err := validate(); err != nil {
		return FileStatus{Status: "invalid", Path: path}
	}
	return FileStatus{Status: validStatus, Path: path}
}
