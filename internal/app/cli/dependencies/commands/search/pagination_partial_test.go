package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func TestListsReturnsUsablePartialPage(t *testing.T) {
	payload := readPaginationPayload(t, "pagination-partial-source.json")
	requester := &controlledRequester{result: app.OperationResult{OK: true, Payload: payload}}

	status, stdout, stderr := run(t, requester, "mixed", "--tab", "LiStS", "--cursor", "opaque-cursor")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%q", status, stderr)
	}
	if len(requester.calls) != 1 || requester.calls[0].overrides["product"] != "Lists" || requester.calls[0].overrides["cursor"] != "opaque-cursor" {
		t.Fatalf("calls=%#v", requester.calls)
	}

	var page struct {
		Tab     string `json:"tab"`
		Results []struct {
			Type string `json:"type"`
		} `json:"results"`
		NextCursor string `json:"next_cursor"`
		Warnings   []struct {
			Code string `json:"code"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(stdout), &page); err != nil {
		t.Fatal(err)
	}
	if page.Tab != "lists" || page.NextCursor != "new-bottom" || len(page.Results) != 3 || page.Results[2].Type != "list" || len(page.Warnings) != 1 || page.Warnings[0].Code != "UNKNOWN_SEARCH_ENTRY" {
		t.Fatalf("page=%s", stdout)
	}
}

func TestRejectsUnsupportedResultOnlyPage(t *testing.T) {
	payload := readPaginationPayload(t, "pagination-unknown-only-source.json")
	status, stdout, stderr := run(t, &controlledRequester{result: app.OperationResult{OK: true, Payload: payload}}, "unknown")
	if status == 0 || stdout != "" || stderr != "{\"code\":\"RESPONSE_SHAPE_CHANGED\",\"message\":\"X returned an unrecognized SearchTimeline response\"}\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
}

func readPaginationPayload(t *testing.T, name string) any {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "search", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var payload any
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
