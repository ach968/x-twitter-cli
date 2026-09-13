package search_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/operations/search"
)

type controlledDirectExecutor struct {
	operation app.OperationName
	overrides map[string]any
	result    app.OperationResult
	err       error
}

func (executor *controlledDirectExecutor) Execute(_ context.Context, operation app.OperationName, overrides map[string]any) (app.OperationResult, error) {
	executor.operation = operation
	executor.overrides = overrides
	return executor.result, executor.err
}

func TestDirectSourceAppliesSearchRequestPolicy(t *testing.T) {
	executor := &controlledDirectExecutor{result: app.OperationResult{OK: true, Payload: emptySearchSource()}}
	source := search.NewDirectSource(executor)
	cursor := "opaque cursor"

	_, err := source.SearchTimeline(context.Background(), search.Request{Query: "query", Tab: search.TabMedia, Cursor: &cursor})
	if err != nil {
		t.Fatal(err)
	}
	if executor.operation != app.SearchTimeline {
		t.Fatalf("operation = %q", executor.operation)
	}
	if executor.overrides["rawQuery"] != "query" || executor.overrides["product"] != "Media" || executor.overrides["cursor"] != cursor {
		t.Fatalf("overrides = %#v", executor.overrides)
	}
}

func TestDirectSourceReturnsClassifiedFailure(t *testing.T) {
	want := &app.OperationFailure{Code: "CONTRACT_FAILED", Message: "contract", RecoveryCommand: "twt contract refresh"}
	source := search.NewDirectSource(&controlledDirectExecutor{result: app.OperationResult{Error: want}})

	_, err := source.SearchTimeline(context.Background(), search.Request{Query: "query", Tab: search.TabTop})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}
