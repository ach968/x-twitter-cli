package view_test

import (
	"context"
	"encoding/json"
	"errors"
	command "github.com/ach968/x-twitter-cli/cmd/twt/dependencies/commands/view"
	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	view "github.com/ach968/x-twitter-cli/internal/app/operations/view"
	"os"
	"path/filepath"
	"testing"
)

func TestMissingViewContractPreservesExistingContractsAndRequestsRefresh(t *testing.T) {
	props := app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{}}
	for _, name := range []app.OperationName{app.SearchTimeline, app.Bookmarks, app.BookmarkSearchTimeline} {
		props.Operations[name] = app.OperationContract{Family: "graphql", Host: "x.com", Path: "/i/api/graphql/test/" + string(name), Method: "GET", Encoding: "query", Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{}}
	}
	path := filepath.Join(t.TempDir(), "contracts.json")
	data, _ := json.Marshal(props)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TWT_CONTRACT_FILE", path)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := contracts.Load(path); err != nil {
		t.Fatalf("legacy contracts invalid: %v", err)
	}
	_, err := command.New().Execute(context.Background(), view.Request{PostID: "100"})
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != "CONTRACT_FAILED" || failure.RecoveryCommand != "twt contract refresh" {
		t.Fatalf("err=%v", err)
	}
}
