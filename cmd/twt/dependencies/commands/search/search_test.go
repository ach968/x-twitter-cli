package search

import (
	"context"
	"errors"
	app "github.com/ach968/x-twitter-cli/internal/app"
	operation "github.com/ach968/x-twitter-cli/internal/app/operations/search"
	"path/filepath"
	"testing"
)

func TestClientClassifiesInvalidContractState(t *testing.T) {
	t.Setenv("TWT_CONTRACT_FILE", filepath.Join(t.TempDir(), "missing-contracts.json"))
	_, err := New().Execute(context.Background(), operation.Request{})
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != "CONTRACT_FAILED" || failure.RecoveryCommand != "twt contract refresh" {
		t.Fatalf("unexpected error: %v", err)
	}
}
