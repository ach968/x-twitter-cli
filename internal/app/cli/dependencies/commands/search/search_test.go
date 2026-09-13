package search_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	app "github.com/ach968/x-twitter-cli3/internal/app"
	searchcommand "github.com/ach968/x-twitter-cli3/internal/app/cli/dependencies/commands/search"
	searchoperation "github.com/ach968/x-twitter-cli3/internal/app/operations/search"
)

type controlledRequester struct {
	page  searchoperation.Page
	err   error
	calls []searchoperation.Request
}

func (requester *controlledRequester) Execute(_ context.Context, request searchoperation.Request) (searchoperation.Page, error) {
	requester.calls = append(requester.calls, request)
	return requester.page, requester.err
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
	next := "next-value"
	requester := &controlledRequester{page: searchoperation.Page{Query: "exact query", Tab: searchoperation.TabLatest, Results: []any{}, NextCursor: &next, Warnings: []searchoperation.Warning{}}}
	status, stdout, stderr := run(t, requester, "exact query", "--tab", "lAtEsT", "--cursor", "opaque+/= value")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%s", status, stderr)
	}
	if stdout != "{\"query\":\"exact query\",\"tab\":\"latest\",\"results\":[],\"next_cursor\":\"next-value\",\"warnings\":[]}\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	if len(requester.calls) != 1 || requester.calls[0].Query != "exact query" {
		t.Fatalf("calls = %#v", requester.calls)
	}
	if requester.calls[0].Tab != searchoperation.TabLatest {
		t.Fatalf("tab = %#v", requester.calls[0].Tab)
	}
	if requester.calls[0].Cursor == nil || *requester.calls[0].Cursor != "opaque+/= value" {
		t.Fatalf("cursor = %#v", requester.calls[0].Cursor)
	}
}

func TestDefaultsToTopWithoutCursor(t *testing.T) {
	requester := &controlledRequester{page: searchoperation.Page{Query: "query", Tab: searchoperation.TabTop, Results: []any{}, Warnings: []searchoperation.Warning{}}}
	status, _, _ := run(t, requester, "query")
	if status != 0 {
		t.Fatalf("status = %d", status)
	}
	if requester.calls[0].Tab != searchoperation.TabTop {
		t.Fatalf("tab = %#v", requester.calls[0].Tab)
	}
	if requester.calls[0].Cursor != nil {
		t.Fatalf("cursor unexpectedly forwarded: %#v", requester.calls[0].Cursor)
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

func TestOperationFailureUsesStableErrorDocument(t *testing.T) {
	requester := &controlledRequester{err: &app.OperationFailure{Code: "RATE_LIMITED", Message: "X rate-limited the request"}}
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
