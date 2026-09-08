package search_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
	searchcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/search"
)

type searchCall struct {
	query     string
	overrides map[string]any
}

type controlledRequester struct {
	result app.OperationResult
	err    error
	calls  []searchCall
}

func (requester *controlledRequester) SearchTimeline(_ context.Context, query string, overrides map[string]any) (app.OperationResult, error) {
	requester.calls = append(requester.calls, searchCall{query: query, overrides: overrides})
	return requester.result, requester.err
}

func emptyPayload(cursor string) any {
	var source any
	text := `{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[]}]}}}}}`
	if cursor != "" {
		text = `{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[{"content":{"entryType":"TimelineTimelineCursor","cursorType":"Bottom","value":"` + cursor + `"}}]}]}}}}}`
	}
	if err := json.Unmarshal([]byte(text), &source); err != nil {
		panic(err)
	}
	return source
}

func run(t *testing.T, requester searchcommand.Requester, arguments ...string) (int, string, string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	status := searchcommand.New(requester).Run(context.Background(), arguments, strings.NewReader(""), &stdout, &stderr)
	return status, stdout.String(), stderr.String()
}

func TestHelpDoesNotExecuteRequest(t *testing.T) {
	requester := &controlledRequester{}
	status, stdout, stderr := run(t, requester, "--help")
	if status != 0 || stderr != "" || !strings.Contains(stdout, "Usage:") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
	if len(requester.calls) != 0 {
		t.Fatalf("help executed %d requests", len(requester.calls))
	}
}

func TestForwardsCanonicalTabAndOpaqueCursor(t *testing.T) {
	requester := &controlledRequester{result: app.OperationResult{OK: true, Payload: emptyPayload("next-value")}}
	status, stdout, stderr := run(t, requester, "exact query", "--tab", "lAtEsT", "--cursor", "opaque+/= value")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%s", status, stderr)
	}
	if stdout != "{\"query\":\"exact query\",\"tab\":\"latest\",\"results\":[],\"next_cursor\":\"next-value\",\"warnings\":[]}\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	if len(requester.calls) != 1 || requester.calls[0].query != "exact query" {
		t.Fatalf("calls = %#v", requester.calls)
	}
	if got := requester.calls[0].overrides["product"]; got != "Latest" {
		t.Fatalf("product = %#v", got)
	}
	if got := requester.calls[0].overrides["cursor"]; got != "opaque+/= value" {
		t.Fatalf("cursor = %#v", got)
	}
}

func TestDefaultsToTopWithoutCursor(t *testing.T) {
	requester := &controlledRequester{result: app.OperationResult{OK: true, Payload: emptyPayload("")}}
	status, _, _ := run(t, requester, "query")
	if status != 0 {
		t.Fatalf("status = %d", status)
	}
	if got := requester.calls[0].overrides["product"]; got != "Top" {
		t.Fatalf("product = %#v", got)
	}
	if _, ok := requester.calls[0].overrides["cursor"]; ok {
		t.Fatalf("cursor unexpectedly forwarded: %#v", requester.calls[0].overrides)
	}
}

func TestMalformedArgumentsFailWithoutRequest(t *testing.T) {
	tests := [][]string{{}, {""}, {"   "}, {"one", "two"}, {"query", "--unknown"}, {"query", "--tab", "photos"}, {"query", "--tab"}, {"query", "--cursor"}}
	requester := &controlledRequester{}
	for _, arguments := range tests {
		status, stdout, stderr := run(t, requester, arguments...)
		if status == 0 || stdout != "" || !strings.Contains(stderr, `"code":"INVALID_ARGUMENT"`) {
			t.Fatalf("arguments %q: status=%d stdout=%q stderr=%q", arguments, status, stdout, stderr)
		}
	}
	if len(requester.calls) != 0 {
		t.Fatalf("malformed invocations executed %d requests", len(requester.calls))
	}
}

func TestEmptyQueryExplainsHowToPassALiteralDollarSign(t *testing.T) {
	requester := &controlledRequester{}
	status, stdout, stderr := run(t, requester, "")
	if status == 0 || stdout != "" {
		t.Fatalf("status=%d stdout=%q", status, stdout)
	}
	want := "{\"code\":\"INVALID_ARGUMENT\",\"message\":\"search query is empty; shells expand $NAME inside double quotes, so use single quotes (for example: twt search '$NVDA') or escape the dollar sign\"}\n"
	if stderr != want {
		t.Fatalf("stderr=%q, want %q", stderr, want)
	}
	if len(requester.calls) != 0 {
		t.Fatalf("empty query executed %d requests", len(requester.calls))
	}
}

func TestOutputDecodesHTMLEntitiesWithoutJSONHTMLEscapes(t *testing.T) {
	tweet := map[string]any{"rest_id": "html", "legacy": map[string]any{"full_text": "S&amp;P &gt; noise"}}
	item := map[string]any{"itemType": "TimelineTweet", "tweet_results": map[string]any{"result": tweet}}
	entry := map[string]any{"content": map[string]any{"itemContent": item}}
	instruction := map[string]any{"type": "TimelineAddEntries", "entries": []any{entry}}
	timeline := map[string]any{"instructions": []any{instruction}}
	payload := map[string]any{"data": map[string]any{"search_by_raw_query": map[string]any{"search_timeline": map[string]any{"timeline": timeline}}}}
	requester := &controlledRequester{result: app.OperationResult{OK: true, Payload: payload}}
	status, stdout, stderr := run(t, requester, "markets")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%q", status, stderr)
	}
	if !strings.Contains(stdout, `"text":"S&P > noise"`) || strings.Contains(stdout, `\u0026`) || strings.Contains(stdout, `&amp;`) {
		t.Fatalf("stdout retains HTML encoding: %s", stdout)
	}
}

func TestOperationFailureUsesStableErrorDocument(t *testing.T) {
	requester := &controlledRequester{result: app.OperationResult{OK: false, Error: &app.OperationFailure{Code: "RATE_LIMITED", Message: "X rate-limited the request"}}}
	status, stdout, stderr := run(t, requester, "query")
	if status == 0 || stdout != "" {
		t.Fatalf("status=%d stdout=%q", status, stdout)
	}
	if stderr != "{\"code\":\"RATE_LIMITED\",\"message\":\"X rate-limited the request\"}\n" {
		t.Fatalf("stderr=%q", stderr)
	}
}

func TestInternalRequestFailureIsStable(t *testing.T) {
	requester := &controlledRequester{err: errors.New("secret internal detail")}
	status, _, stderr := run(t, requester, "query")
	if status == 0 || stderr != "{\"code\":\"SEARCH_FAILED\",\"message\":\"Unable to execute search\"}\n" {
		t.Fatalf("status=%d stderr=%q", status, stderr)
	}
}
