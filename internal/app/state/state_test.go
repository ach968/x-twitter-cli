package state

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func TestResolvePathsAndContractSource(t *testing.T) {
	paths := ResolvePaths("/home/user", map[string]string{"XDG_CONFIG_HOME": "/config", "XDG_STATE_HOME": "/state"})
	if paths.ActiveContractPath != "/config/x-twt/contracts.json" || paths.ProfilePath != "/state/x-twt/chromium-profile" || paths.EvidenceDirectory != "/state/x-twt/evidence" {
		t.Fatalf("unexpected paths: %#v", paths)
	}
	if value := ResolveContractSource("/option", map[string]string{"TWT_CONTRACT_FILE": "/environment"}, paths.ActiveContractPath); value != "/option" {
		t.Fatal(value)
	}
	if value := ResolveContractSource("", map[string]string{"TWT_CONTRACT_FILE": "/environment"}, paths.ActiveContractPath); value != "/environment" {
		t.Fatal(value)
	}
}

func TestCapturedStatePersistence(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "nested")
	candidate := filepath.Join(directory, "contracts.candidate.json")
	authenticationPath := filepath.Join(directory, "authentication.json")
	capture := app.CapturedState{
		Contracts: app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{}},
		Authentication: app.AuthenticationState{
			Cookies: []app.AuthenticationCookie{{Name: "auth_token", Value: "session-secret"}}, Authorization: "Bearer captured-token",
		},
	}
	if err := SaveCaptured(candidate, authenticationPath, capture); err != nil {
		t.Fatal(err)
	}
	contractBytes, _ := os.ReadFile(candidate)
	authBytes, _ := os.ReadFile(authenticationPath)
	if strings.Contains(string(contractBytes), "session-secret") || !strings.Contains(string(authBytes), "session-secret") {
		t.Fatal("authentication was not stored separately")
	}
	info, _ := os.Stat(authenticationPath)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("authentication mode = %o", info.Mode().Perm())
	}
	loaded, err := LoadAuthentication(authenticationPath)
	if err != nil || !reflect.DeepEqual(loaded, capture.Authentication) {
		t.Fatalf("loaded = %#v, %v", loaded, err)
	}
}
