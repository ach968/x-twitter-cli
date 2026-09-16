package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
)

type transactionIDFunc func(method, path string) (string, error)

func (function transactionIDFunc) Generate(method, path string) (string, error) {
	return function(method, path)
}

var testTransactionIDs = transactionIDFunc(func(string, string) (string, error) {
	return "generated-id", nil
})

type transportFunc struct {
	home           func(app.PreparedRequest) (app.UpstreamResponse, error)
	search         func(app.PreparedRequest) (app.UpstreamResponse, error)
	bookmarks      func(app.PreparedRequest) (app.UpstreamResponse, error)
	bookmarkSearch func(app.PreparedRequest) (app.UpstreamResponse, error)
	view           func(app.PreparedRequest) (app.UpstreamResponse, error)
}

func (transport transportFunc) Execute(_ context.Context, operation app.OperationName, request app.PreparedRequest) (app.UpstreamResponse, error) {
	switch operation {
	case app.HomeTimeline:
		return transport.home(request)
	case app.SearchTimeline:
		return transport.search(request)
	case app.Bookmarks:
		return transport.bookmarks(request)
	case app.BookmarkSearchTimeline:
		return transport.bookmarkSearch(request)
	case app.TweetDetail:
		return transport.view(request)
	default:
		return app.UpstreamResponse{}, nil
	}
}

func TestViewCandidateMustReturnRequestedPostBeforeActivation(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			directory := t.TempDir()
			active := filepath.Join(directory, "active.json")
			candidate := filepath.Join(directory, "candidate.json")
			old := testContracts()
			writeContracts(t, active, old)
			before, _ := os.ReadFile(active)
			newer := testContracts()
			newer.Operations[app.TweetDetail] = app.OperationContract{Family: "graphql", Host: "x.com", Path: "/i/api/graphql/detail/TweetDetail", Method: "GET", Encoding: "query", Variables: map[string]any{"focalTweetId": "100"}, Features: map[string]any{}, FieldToggles: map[string]any{}}
			writeContracts(t, candidate, newer)
			transport := successfulTransport()
			calls := 0
			transport.view = func(request app.PreparedRequest) (app.UpstreamResponse, error) {
				calls++
				body := `{"data":{}}`
				if valid {
					body = `{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineAddEntries","entries":[{"entryId":"tweet-100","content":{"entryType":"TimelineTimelineItem","itemContent":{"itemType":"TimelineTweet","tweet_results":{"result":{"rest_id":"100","legacy":{"full_text":"synthetic"}}}}}}]}]}}}`
				}
				return app.UpstreamResponse{Status: 200, Body: body}, nil
			}
			result, err := ValidateAndActivateCandidate(context.Background(), candidate, active, app.AuthenticationState{}, transport, testTransactionIDs)
			if err != nil || result.Activated != valid || calls != 1 {
				t.Fatalf("activated=%t calls=%d err=%v", result.Activated, calls, err)
			}
			if !valid {
				after, _ := os.ReadFile(active)
				if string(after) != string(before) {
					t.Fatal("invalid View replaced active contracts")
				}
			}
		})
	}
}

func successfulTransport() transportFunc {
	success := app.UpstreamResponse{Status: 200, Body: `{"data":{"ok":true}}`}
	return transportFunc{
		home:           func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		search:         func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		bookmarks:      func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		bookmarkSearch: func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
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
		app.Bookmarks: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmarks-id/Bookmarks", Method: "GET", Encoding: "query",
			Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.BookmarkSearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmark-search-id/BookmarkSearchTimeline", Method: "GET", Encoding: "query",
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

	writeContracts(t, active, old)
	writeContracts(t, candidate, testContracts())
	rejected = successfulTransport()
	rejected.bookmarkSearch = func(request app.PreparedRequest) (app.UpstreamResponse, error) {
		parsed, err := url.Parse(request.URL)
		if err != nil {
			t.Fatal(err)
		}
		var variables map[string]any
		if err := json.Unmarshal([]byte(parsed.Query().Get("variables")), &variables); err != nil || variables["rawQuery"] != "x-twitter-cli-contract-validation-improbable-6d1e2f" {
			t.Fatalf("bookmark-search validation query was not preserved: %s", request.URL)
		}
		return app.UpstreamResponse{Status: 403}, nil
	}
	result, err = ValidateAndActivateCandidate(context.Background(), candidate, active, app.AuthenticationState{}, rejected, testTransactionIDs)
	if err != nil || result.Activated {
		t.Fatalf("bookmark-search rejection activated candidate: %#v %v", result, err)
	}
	contents, _ = os.ReadFile(active)
	if !strings.Contains(string(contents), "old-id") {
		t.Fatal("active contracts changed after bookmark-search rejection")
	}
}

func TestValidateAndActivateCandidateDoesNotRequireHomeTimeline(t *testing.T) {
	directory := t.TempDir()
	active := filepath.Join(directory, "contracts.json")
	candidate := filepath.Join(directory, "contracts.candidate.json")
	properties := testContracts()
	delete(properties.Operations, app.HomeTimeline)
	writeContracts(t, candidate, properties)

	transport := successfulTransport()
	transport.home = func(app.PreparedRequest) (app.UpstreamResponse, error) {
		t.Fatal("optional HomeTimeline was executed")
		return app.UpstreamResponse{}, nil
	}
	result, err := ValidateAndActivateCandidate(context.Background(), candidate, active, app.AuthenticationState{}, transport, testTransactionIDs)
	if err != nil || !result.Activated {
		t.Fatalf("activation failed: %#v %v", result, err)
	}
	loaded, err := contracts.Load(active)
	if err != nil || len(loaded.Operations) != 3 {
		t.Fatalf("active contracts = %#v, %v", loaded, err)
	}
}
