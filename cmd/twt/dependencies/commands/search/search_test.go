package search

import (
	"context"
	"errors"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
	searchcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/search"
)

type controlledSearchClient struct {
	result    app.OperationResult
	calls     int
	query     string
	overrides map[string]any
}

func (client *controlledSearchClient) SearchTimeline(_ context.Context, query string, overrides map[string]any) (app.OperationResult, error) {
	client.calls++
	client.query = query
	client.overrides = overrides
	return client.result, nil
}

func TestClientDefersAndReusesInitialization(t *testing.T) {
	client := &controlledSearchClient{result: app.OperationResult{OK: true}}
	loads := 0
	dependency := newLazyClient(func(context.Context) (searchcommand.Requester, error) {
		loads++
		return client, nil
	})

	if loads != 0 {
		t.Fatalf("search client loaded during runtime construction")
	}
	overrides := map[string]any{"product": "Latest"}
	for range 2 {
		result, err := dependency.SearchTimeline(context.Background(), "golang", overrides)
		if err != nil || !result.OK {
			t.Fatalf("SearchTimeline() result = %#v, error = %v", result, err)
		}
	}
	if loads != 1 {
		t.Fatalf("search client loads = %d, want 1", loads)
	}
	if client.calls != 2 || client.query != "golang" || client.overrides["product"] != "Latest" {
		t.Fatalf("search calls = %d, query = %q, overrides = %#v", client.calls, client.query, client.overrides)
	}
}

func TestClientReusesInitializationFailure(t *testing.T) {
	want := errors.New("load search client")
	loads := 0
	dependency := newLazyClient(func(context.Context) (searchcommand.Requester, error) {
		loads++
		return nil, want
	})

	for range 2 {
		result, err := dependency.SearchTimeline(context.Background(), "golang", nil)
		if !errors.Is(err, want) || result != (app.OperationResult{}) {
			t.Fatalf("SearchTimeline() result = %#v, error = %v", result, err)
		}
	}
	if loads != 1 {
		t.Fatalf("search client loads = %d, want 1", loads)
	}
}
