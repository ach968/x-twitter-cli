package workflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
)

type transactionIDFunc func(method, path string) (string, error)

func (function transactionIDFunc) Generate(method, path string) (string, error) {
	return function(method, path)
}

var testTransactionIDs = transactionIDFunc(func(string, string) (string, error) {
	return "generated-id", nil
})

type transportFunc struct {
	home   func(app.PreparedRequest) (app.UpstreamResponse, error)
	search func(app.PreparedRequest) (app.UpstreamResponse, error)
}

func (transport transportFunc) Execute(_ context.Context, operation app.OperationName, request app.PreparedRequest) (app.UpstreamResponse, error) {
	switch operation {
	case app.HomeTimeline:
		return transport.home(request)
	case app.SearchTimeline:
		return transport.search(request)
	default:
		return app.UpstreamResponse{}, nil
	}
}

func successfulTransport() transportFunc {
	success := app.UpstreamResponse{Status: 200, Body: `{"data":{"ok":true}}`}
	return transportFunc{
		home:   func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		search: func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
	}
}

func testContracts() app.ContractProperties {
	return app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{
		app.HomeTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/home-id/HomeTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.SearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/search-id/SearchTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
	}}
}

func writeContracts(t *testing.T, path string, value app.ContractProperties) {
	t.Helper()
	contents, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAndActivateCandidate(t *testing.T) {
	directory := t.TempDir()
	active := filepath.Join(directory, "contracts.json")
	candidate := filepath.Join(directory, "contracts.candidate.json")
	old := testContracts()
	operation := old.Operations[app.HomeTimeline]
	operation.Path = "/i/api/graphql/old-id/HomeTimeline"
	old.Operations[app.HomeTimeline] = operation
	writeContracts(t, active, old)
	writeContracts(t, candidate, testContracts())
	result, err := ValidateAndActivateCandidate(context.Background(), candidate, active, app.AuthenticationState{}, successfulTransport(), testTransactionIDs)
	if err != nil || !result.Activated {
		t.Fatalf("activation failed: %#v %v", result, err)
	}
	contents, _ := os.ReadFile(active)
	if !strings.Contains(string(contents), "home-id") {
		t.Fatal("candidate was not activated")
	}

	writeContracts(t, active, old)
	writeContracts(t, candidate, testContracts())
	rejected := successfulTransport()
	rejected.search = func(app.PreparedRequest) (app.UpstreamResponse, error) {
		return app.UpstreamResponse{Status: 403}, nil
	}
	result, err = ValidateAndActivateCandidate(context.Background(), candidate, active, app.AuthenticationState{}, rejected, testTransactionIDs)
	if err != nil || result.Activated {
		t.Fatalf("rejected candidate activated: %#v %v", result, err)
	}
	contents, _ = os.ReadFile(active)
	if !strings.Contains(string(contents), "old-id") {
		t.Fatal("active contracts changed after rejection")
	}
}
