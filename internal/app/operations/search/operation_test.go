package search_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/operations/search"
)

type controlledSource struct {
	payload any
	err     error
	request search.Request
}

func (source *controlledSource) SearchTimeline(_ context.Context, request search.Request) (any, error) {
	source.request = request
	return source.payload, source.err
}

func TestExecuteBuildsANormalizedPageFromTheSource(t *testing.T) {
	source := &controlledSource{payload: emptySearchSource()}
	operation := search.New(source)
	cursor := "opaque-next"

	page, err := operation.Execute(context.Background(), search.Request{Query: "golang", Tab: search.TabLatest, Cursor: &cursor})
	if err != nil {
		t.Fatal(err)
	}
	if page.Query != "golang" || page.Tab != search.TabLatest || page.Warnings == nil || len(page.Results) != 0 {
		t.Fatalf("page = %#v", page)
	}
	if source.request.Query != "golang" || source.request.Tab != search.TabLatest || source.request.Cursor != &cursor {
		t.Fatalf("source request = %#v", source.request)
	}
}

func TestExecutePreservesClassifiedFailures(t *testing.T) {
	want := &app.OperationFailure{Code: "RATE_LIMITED", Message: "X rate-limited the request"}
	operation := search.New(&controlledSource{err: want})

	_, err := operation.Execute(context.Background(), search.Request{Query: "golang", Tab: search.TabTop})
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure != want {
		t.Fatalf("error = %#v", err)
	}
}

func TestExecuteClassifiesMalformedSourceAsASafeFailure(t *testing.T) {
	operation := search.New(&controlledSource{payload: map[string]any{"data": map[string]any{}}})

	_, err := operation.Execute(context.Background(), search.Request{Query: "golang", Tab: search.TabTop})
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != "RESPONSE_SHAPE_CHANGED" || failure.Message != "X returned an unrecognized SearchTimeline response" {
		t.Fatalf("error = %#v", err)
	}
}

func emptySearchSource() any {
	return map[string]any{"data": map[string]any{"search_by_raw_query": map[string]any{"search_timeline": map[string]any{"timeline": map[string]any{"instructions": []any{
		map[string]any{"type": "TimelineAddEntries", "entries": []any{}},
	}}}}}}
}
