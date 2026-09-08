package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ach968/x-twt-cli/internal/app/cli"
)

func runApplication(arguments ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	status := cli.New(cli.Dependencies{}).Run(context.Background(), arguments, strings.NewReader(""), &stdout, &stderr)
	return status, stdout.String(), stderr.String()
}

func TestHelpIsHumanReadableAndSuccessful(t *testing.T) {
	for _, arguments := range [][]string{{}, {"help"}, {"--help"}, {"search", "--help"}, {"help", "search"}} {
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
