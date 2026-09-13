package management

import (
	"bytes"
	"strings"
	"testing"

	managementservice "github.com/ach968/x-twitter-cli/internal/app/management"
)

func TestBrowserSetupConfirmationWarnsBeforeCleanup(t *testing.T) {
	var output bytes.Buffer
	confirm := browserSetupConfirmation(strings.NewReader("yes\n"), &output)
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
	confirm := browserSetupConfirmation(strings.NewReader("\n"), &output)
	accepted, err := confirm(managementservice.BrowserSetupPrompt{
		Action:             "cleanup",
		RequiredRevision:   "1234",
		InstalledRevisions: []string{"1217"},
	})
	if err != nil || accepted {
		t.Fatalf("accepted=%v err=%v", accepted, err)
	}
}
