package bookmarks

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	app "github.com/ach968/x-twitter-cli3/internal/app"
)

type controlledSource struct {
	payload any
	err     error
}

func (source controlledSource) Fetch(_ context.Context, _ Request) (any, error) {
	return source.payload, source.err
}

func TestOperationExecuteFixtures(t *testing.T) {
	tests := []struct {
		source string
		golden string
		query  string
	}{
		{source: "populated-list-source.json", golden: "populated-list-golden.json"},
		{source: "later-list-source.json", golden: "later-list-golden.json"},
		{source: "empty-list-source.json", golden: "empty-list-golden.json"},
		{source: "partial-unknown-source.json", golden: "partial-unknown-golden.json"},
		{source: "nested-post-context-source.json", golden: "nested-post-context-golden.json"},
		{source: "populated-search-source.json", golden: "populated-search-golden.json", query: "synthetic query"},
		{source: "empty-search-with-cursor-source.json", golden: "empty-search-with-cursor-golden.json", query: "no synthetic matches"},
		{source: "later-search-source.json", golden: "later-search-golden.json", query: "later query"},
		{source: "partial-unknown-search-source.json", golden: "partial-unknown-search-golden.json", query: "partial query"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			var source any
			readJSON(t, test.source, &source)
			var query *string
			if test.query != "" {
				query = &test.query
			}
			actual, err := New(controlledSource{payload: source}).Execute(context.Background(), Request{Query: query})
			if err != nil {
				t.Fatal(err)
			}
			var expected Page
			readJSON(t, test.golden, &expected)
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("page mismatch\nactual: %#v\nexpected: %#v", actual, expected)
			}
		})
	}
}

func TestOperationExecutePreservesDuplicatePostsInSourceOrder(t *testing.T) {
	result := map[string]any{
		"rest_id": "duplicate-post",
		"legacy":  map[string]any{"full_text": "Repeated upstream result"},
	}
	entry := func(id string) any {
		return map[string]any{
			"entryId": id,
			"content": map[string]any{
				"entryType": "TimelineTimelineItem",
				"itemContent": map[string]any{
					"itemType":      "TimelineTweet",
					"tweet_results": map[string]any{"result": result},
				},
			},
		}
	}
	payload := map[string]any{"data": map[string]any{
		"bookmark_timeline_v2": map[string]any{"timeline": map[string]any{
			"instructions": []any{map[string]any{
				"type":    "TimelineAddEntries",
				"entries": []any{entry("first"), entry("second")},
			}},
		}},
	}}

	page, err := New(controlledSource{payload: payload}).Execute(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Bookmarks) != 2 || page.Bookmarks[0].ID != "duplicate-post" || page.Bookmarks[1].ID != "duplicate-post" {
		t.Fatalf("bookmarks = %#v", page.Bookmarks)
	}
}
func TestOperationExecuteMapsInvalidEnvelopeToSafeFailure(t *testing.T) {
	var source any
	readJSON(t, "invalid-envelope-source.json", &source)
	query := "synthetic invalid"
	if _, err := New(controlledSource{payload: source}).Execute(context.Background(), Request{Query: &query}); err == nil {
		t.Fatal("expected safe shape failure")
	} else if failure, ok := err.(*app.OperationFailure); !ok || failure.Code != "RESPONSE_SHAPE_CHANGED" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestOperationExecutePreservesClassifiedAndHidesUnexpectedErrors(t *testing.T) {
	classified := &app.OperationFailure{Code: "RATE_LIMITED", Message: "safe"}
	query := "synthetic failure"
	if _, err := New(controlledSource{err: classified}).Execute(context.Background(), Request{Query: &query}); err != classified {
		t.Fatalf("classified error changed: %#v", err)
	}
	if _, err := New(controlledSource{err: errors.New("private transport detail")}).Execute(context.Background(), Request{}); err == nil {
		t.Fatal("expected safe failure")
	} else if failure := err.(*app.OperationFailure); failure.Code != "BOOKMARKS_FAILED" || failure.Message != "Unable to execute bookmarks" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
func readJSON(t *testing.T, name string, destination any) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(contents, destination); err != nil {
		t.Fatal(err)
	}
}

type directExecutor struct {
	operation app.OperationName
	overrides map[string]any
	result    app.OperationResult
	resultSet bool
	err       error
}

func (executor *directExecutor) Execute(_ context.Context, operation app.OperationName, overrides map[string]any) (app.OperationResult, error) {
	executor.operation, executor.overrides = operation, overrides
	if executor.err != nil {
		return app.OperationResult{}, executor.err
	}
	if executor.resultSet {
		return executor.result, nil
	}
	return app.OperationResult{OK: true}, nil
}

func TestDirectSourceSelectsOperationAndPreservesInputs(t *testing.T) {
	executor := &directExecutor{}
	source := NewDirectSource(executor)
	query, cursor := "exact synthetic query", "opaque synthetic cursor"
	if _, err := source.Fetch(context.Background(), Request{Query: &query, Cursor: &cursor}); err != nil {
		t.Fatal(err)
	}
	if executor.operation != app.BookmarkSearchTimeline || executor.overrides["rawQuery"] != query || executor.overrides["cursor"] != cursor {
		t.Fatalf("unexpected request: %#v %#v", executor.operation, executor.overrides)
	}
	if _, err := source.Fetch(context.Background(), Request{}); err != nil {
		t.Fatal(err)
	}
	if executor.operation != app.Bookmarks || len(executor.overrides) != 0 {
		t.Fatalf("unexpected list request: %#v %#v", executor.operation, executor.overrides)
	}
}

func TestDirectSourcePreservesClassifiedFailuresAndTransportErrors(t *testing.T) {
	classified := &app.OperationFailure{Code: "CONTRACT_FAILED", Message: "safe", RecoveryCommand: "twt contract refresh"}
	source := NewDirectSource(&directExecutor{result: app.OperationResult{Error: classified}, resultSet: true})
	if _, err := source.Fetch(context.Background(), Request{}); err != classified {
		t.Fatalf("classified error changed: %#v", err)
	}

	transportErr := errors.New("transport failed")
	source = NewDirectSource(&directExecutor{err: transportErr})
	if _, err := source.Fetch(context.Background(), Request{}); !errors.Is(err, transportErr) {
		t.Fatalf("transport error changed: %#v", err)
	}

	source = NewDirectSource(&directExecutor{result: app.OperationResult{}, resultSet: true})
	if _, err := source.Fetch(context.Background(), Request{}); err == nil {
		t.Fatal("expected defensive failure for an unsuccessful result without a classified error")
	}
}
