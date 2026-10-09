package management

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/browser"
	managementservice "github.com/ach968/x-twitter-cli/internal/app/management"
)

func TestBrowserSetupConfirmationWarnsBeforeCleanup(t *testing.T) {
	var output bytes.Buffer
	confirm := browserSetupConfirmation(terminalLoginPrompt(strings.NewReader("yes\n"), &output))
	accepted, err := confirm(managementservice.BrowserSetupPrompt{
		Action:             "cleanup",
		RequiredRevision:   "1234",
		InstalledRevisions: []string{"1217", "1220"},
	})
	if err != nil || !accepted {
		t.Fatalf("accepted=%v err=%v", accepted, err)
	}
	want := "New managed Chromium revision 1234 is ready. Remove older managed revisions 1217, 1220? Warning: older twt builds may still require them. [y/N] "
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
}

func TestBrowserSetupCleanupDefaultsToKeepingOldRevisions(t *testing.T) {
	var output bytes.Buffer
	confirm := browserSetupConfirmation(terminalLoginPrompt(strings.NewReader("\n"), &output))
	accepted, err := confirm(managementservice.BrowserSetupPrompt{
		Action:             "cleanup",
		RequiredRevision:   "1234",
		InstalledRevisions: []string{"1217"},
	})
	if err != nil || accepted {
		t.Fatalf("accepted=%v err=%v", accepted, err)
	}
}

func TestTerminalLoginInputIsNotEchoedOrTrimmed(t *testing.T) {
	var output bytes.Buffer
	prompt := terminalLoginPrompt(strings.NewReader("user\n password with spaces \r\n012345\n"), &output)
	for _, tc := range []struct {
		label, want string
		secret      bool
	}{
		{"Username: ", "user", false}, {"Password: ", " password with spaces ", true}, {"Code: ", "012345", true},
	} {
		value, err := prompt(browser.LoginPrompt{Label: tc.label, Secret: tc.secret})
		if err != nil || value != tc.want {
			t.Fatalf("value=%q err=%v", value, err)
		}
	}
	if output.String() != "Username: Password: Code: " {
		t.Fatal("input echoed")
	}
	if _, err := prompt(browser.LoginPrompt{Label: "Code: ", Secret: true}); err == nil {
		t.Fatal("EOF accepted")
	}
}

type managementStub struct {
	login   func(managementservice.ConfirmBrowserSetup, managementservice.LoginOptions) (managementservice.StateChangeResult, error)
	refresh func() (managementservice.StateChangeResult, error)
}

func (*managementStub) Setup(context.Context, managementservice.ConfirmBrowserSetup) (managementservice.SetupResult, error) {
	return managementservice.SetupResult{}, nil
}

func (stub *managementStub) Login(_ context.Context, confirm managementservice.ConfirmBrowserSetup, options managementservice.LoginOptions) (managementservice.StateChangeResult, error) {
	return stub.login(confirm, options)
}

func (stub *managementStub) RefreshContracts(context.Context) (managementservice.StateChangeResult, error) {
	return stub.refresh()
}

func (*managementStub) ContractStatus(context.Context) managementservice.ContractStatus {
	return managementservice.ContractStatus{}
}

func TestLoginModeSelection(t *testing.T) {
	for _, tc := range []struct {
		arguments []string
		headless  bool
	}{
		{[]string{"login"}, false},
		{[]string{"login", "--headless"}, true},
		{[]string{"login", "--headed"}, false},
	} {
		t.Run(strings.Join(tc.arguments, " "), func(t *testing.T) {
			calls := 0
			service := &managementStub{login: func(_ managementservice.ConfirmBrowserSetup, options managementservice.LoginOptions) (managementservice.StateChangeResult, error) {
				calls++
				if options.Headless != tc.headless || (options.Prompt != nil) != tc.headless {
					t.Fatalf("login options=%#v", options)
				}
				return managementservice.StateChangeResult{Status: "authenticated"}, nil
			}}
			var stdout, stderr bytes.Buffer
			code := runAuth(service)(context.Background(), tc.arguments, strings.NewReader(""), &stdout, &stderr)
			if code != 0 || calls != 1 || stderr.Len() != 0 || stdout.String() != "{\"status\":\"authenticated\",\"contract_path\":\"\"}\n" {
				t.Fatalf("code=%d calls=%d stdout=%q stderr=%q", code, calls, stdout.String(), stderr.String())
			}
		})
	}
}

func TestLoginRejectsUnsupportedArguments(t *testing.T) {
	for _, arguments := range [][]string{
		nil, {"login-api"}, {"login-api", "--help"}, {"login", "--headless", "--headed"},
		{"login", "--headless", "--headless"}, {"login", "--headed", "--headed"},
		{"login", "--headless=true"}, {"login", "--username=user"}, {"login", "unexpected"},
	} {
		var stdout, stderr bytes.Buffer
		code := runAuth(nil)(context.Background(), arguments, strings.NewReader(""), &stdout, &stderr)
		if code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "INVALID_ARGUMENT") {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", arguments, code, stdout.String(), stderr.String())
		}
	}
}

func TestLoginSetupAndCredentialsShareInputWithoutEchoingSecrets(t *testing.T) {
	service := &managementStub{login: func(confirm managementservice.ConfirmBrowserSetup, options managementservice.LoginOptions) (managementservice.StateChangeResult, error) {
		accepted, err := confirm(managementservice.BrowserSetupPrompt{Action: "install", RequiredRevision: "1234"})
		if err != nil || !accepted {
			t.Fatalf("accepted=%v error=%v", accepted, err)
		}
		for _, tc := range []struct {
			label, want string
			secret      bool
		}{
			{"Username: ", "account", false},
			{"Password: ", " my password ", true},
			{"Code: ", "012345", true},
		} {
			answer, err := options.Prompt(browser.LoginPrompt{Label: tc.label, Secret: tc.secret})
			if err != nil || answer != tc.want {
				t.Fatalf("answer=%q error=%v", answer, err)
			}
		}
		return managementservice.StateChangeResult{Status: "authenticated"}, nil
	}}
	var stdout, stderr bytes.Buffer
	code := runAuth(service)(context.Background(), []string{"login", "--headless"}, strings.NewReader("yes\naccount\n my password \n012345\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "my password") || strings.Contains(stderr.String(), "012345") || strings.Contains(stdout.String(), "account") {
		t.Fatal("login input leaked to command output")
	}
}

func TestManagementFailuresPreserveSafeAuthGuidanceAndMaskOtherErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		expected string
	}{
		{"unknown", errors.New("password=secret;cookie=secret"), "Unable to authenticate with X"},
		{"challenge", &app.OperationFailure{Code: "AUTH_CHALLENGE_UNSUPPORTED", Message: "Complete this challenge with twt auth login --headed"}, "Complete this challenge with twt auth login --headed"},
		{"required", &app.OperationFailure{Code: "AUTHENTICATION_REQUIRED", Message: "Run twt auth login"}, "Run twt auth login"},
		{"rate limited", &app.OperationFailure{Code: "AUTH_RATE_LIMITED", Message: "X has temporarily limited login attempts; try again later"}, "X has temporarily limited login attempts; try again later"},
		{"login page error", &app.OperationFailure{Code: "AUTH_LOGIN_UNAVAILABLE", Message: "X could not load the next login step; try again later"}, "X could not load the next login step; try again later"},
		{"other typed error", &app.OperationFailure{Code: "TRANSPORT_ERROR", Message: "password=secret"}, "Unable to authenticate with X"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &managementStub{login: func(managementservice.ConfirmBrowserSetup, managementservice.LoginOptions) (managementservice.StateChangeResult, error) {
				return managementservice.StateChangeResult{}, fmt.Errorf("wrapped: %w", tc.err)
			}}
			var stdout, stderr bytes.Buffer
			code := runAuth(service)(context.Background(), []string{"login"}, strings.NewReader(""), &stdout, &stderr)
			if code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.expected) || strings.Contains(stderr.String(), "secret") {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestContractRefreshShowsAuthenticationRecovery(t *testing.T) {
	service := &managementStub{refresh: func() (managementservice.StateChangeResult, error) {
		return managementservice.StateChangeResult{}, &app.OperationFailure{Code: "AUTHENTICATION_REQUIRED", Message: "Run twt auth login or twt auth login --headed"}
	}}
	var stdout, stderr bytes.Buffer
	code := runContract(service)(context.Background(), []string{"refresh"}, strings.NewReader(""), &stdout, &stderr)
	if code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "AUTHENTICATION_REQUIRED") || !strings.Contains(stderr.String(), "twt auth login --headed") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
