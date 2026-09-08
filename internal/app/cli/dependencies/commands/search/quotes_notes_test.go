package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func TestWritesQuotedPostAndCommunityNote(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "search", "testdata", "quotes-notes-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload any
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatal(err)
	}

	status, stdout, stderr := run(t, &controlledRequester{result: app.OperationResult{OK: true, Payload: payload}}, "context")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%q", status, stderr)
	}
	var page struct {
		Results []struct {
			QuotedPost    *struct{} `json:"quoted_post"`
			CommunityNote *struct{} `json:"community_note"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 3 || page.Results[0].QuotedPost == nil || page.Results[0].CommunityNote == nil {
		t.Fatalf("page = %s", stdout)
	}
}
