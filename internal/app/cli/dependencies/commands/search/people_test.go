package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func TestRendersNormalizedUsers(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "..", "test", "testdata", "search-timeline", "people-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}

	requester := &controlledRequester{result: app.OperationResult{OK: true, Payload: source}}
	status, stdout, stderr := run(t, requester, "example people", "--tab", "people")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%q", status, stderr)
	}
	if len(requester.calls) != 1 || requester.calls[0].overrides["product"] != "People" {
		t.Fatalf("calls=%#v", requester.calls)
	}

	var page struct {
		Tab     string `json:"tab"`
		Results []struct {
			Type     string `json:"type"`
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &page); err != nil {
		t.Fatalf("stdout was not JSON: %v", err)
	}
	if page.Tab != "people" || len(page.Results) != 3 || page.Results[0].Type != "user" || page.Results[0].Username != "example" || page.Results[2].Username != "partial" {
		t.Fatalf("page=%s", stdout)
	}
}
