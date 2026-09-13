package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ach968/x-twitter-cli3/internal/app/cli"
	"github.com/ach968/x-twitter-cli3/internal/app/management"
)

func runApplication(arguments ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	status := cli.New(cli.Dependencies{}).Run(context.Background(), arguments, strings.NewReader(""), &stdout, &stderr)
	return status, stdout.String(), stderr.String()
}

type controlledManagement struct {
	calls []string
}

func (service *controlledManagement) Setup(context.Context, management.ConfirmBrowserSetup) (management.SetupResult, error) {
	service.calls = append(service.calls, "setup")
	return management.SetupResult{Status: "ready", ExecutablePath: "/managed/chromium"}, nil
}

func (service *controlledManagement) Login(context.Context, management.ConfirmBrowserSetup) (management.StateChangeResult, error) {
	service.calls = append(service.calls, "auth login")
	return management.StateChangeResult{Status: "authenticated", AuthenticationPath: "/config/authentication.json", ContractPath: "/config/contracts.json"}, nil
}

func (service *controlledManagement) RefreshContracts(context.Context) (management.StateChangeResult, error) {
	service.calls = append(service.calls, "contract refresh")
	return management.StateChangeResult{Status: "refreshed", ContractPath: "/config/contracts.json"}, nil
}

func (service *controlledManagement) ContractStatus(context.Context) management.ContractStatus {
	service.calls = append(service.calls, "contract status")
	return management.ContractStatus{
		LocallyConfigured: true,
		Authentication:    management.FileStatus{Status: "stored", Path: "/config/authentication.json"},
		Contracts:         management.FileStatus{Status: "valid", Path: "/config/contracts.json"},
	}
}

func runManagedApplication(service management.Service, arguments ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	status := cli.New(cli.Dependencies{Management: service}).Run(context.Background(), arguments, strings.NewReader(""), &stdout, &stderr)
	return status, stdout.String(), stderr.String()
}

func TestManagementCommandsExposeAcceptedCLI(t *testing.T) {
	service := &controlledManagement{}
	tests := []struct {
		arguments []string
		stdout    string
	}{
		{[]string{"setup"}, "{\"status\":\"ready\",\"executable_path\":\"/managed/chromium\"}\n"},
		{[]string{"auth", "login"}, "{\"status\":\"authenticated\",\"authentication_path\":\"/config/authentication.json\",\"contract_path\":\"/config/contracts.json\"}\n"},
		{[]string{"contract", "refresh"}, "{\"status\":\"refreshed\",\"contract_path\":\"/config/contracts.json\"}\n"},
		{[]string{"contract", "status"}, "{\"locally_configured\":true,\"authentication\":{\"status\":\"stored\",\"path\":\"/config/authentication.json\"},\"contracts\":{\"status\":\"valid\",\"path\":\"/config/contracts.json\"}}\n"},
	}
	for _, test := range tests {
		status, stdout, stderr := runManagedApplication(service, test.arguments...)
		if status != 0 || stderr != "" || stdout != test.stdout {
			t.Fatalf("arguments %q: status=%d stdout=%q stderr=%q", test.arguments, status, stdout, stderr)
		}
	}
	wantCalls := "setup,auth login,contract refresh,contract status"
	if strings.Join(service.calls, ",") != wantCalls {
		t.Fatalf("calls=%q, want %q", service.calls, wantCalls)
	}
}

func TestHelpIsHumanReadableAndSuccessful(t *testing.T) {
	for _, arguments := range [][]string{
		{}, {"help"}, {"--help"},
		{"search", "--help"}, {"help", "search"},
		{"bookmarks", "--help"}, {"help", "bookmarks"},
		{"setup", "--help"}, {"help", "setup"},
		{"auth", "--help"}, {"auth", "login", "--help"}, {"help", "auth"},
		{"contract", "--help"}, {"contract", "refresh", "--help"}, {"contract", "status", "--help"}, {"help", "contract"},
	} {
		status, stdout, stderr := runApplication(arguments...)
		if status != 0 || stderr != "" || !strings.Contains(stdout, "Usage:") {
			t.Fatalf("arguments %q: status=%d stdout=%q stderr=%q", arguments, status, stdout, stderr)
		}
	}
}

func TestUnknownCommandFails(t *testing.T) {
	status, stdout, stderr := runApplication("unknown")
	if status == 0 || stdout != "" || stderr != "{\"code\":\"INVALID_ARGUMENT\",\"message\":\"Unknown command: unknown\"}\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
}
