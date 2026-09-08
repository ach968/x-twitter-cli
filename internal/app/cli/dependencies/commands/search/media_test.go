package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func TestWritesNormalizedMediaPage(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "search", "testdata", "media-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload any
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatal(err)
	}

	requester := &controlledRequester{result: app.OperationResult{OK: true, Payload: payload}}
	status, stdout, stderr := run(t, requester, "cats", "--tab", "media")
	if status != 0 || stderr != "" {
		t.Fatalf("status=%d stderr=%q", status, stderr)
	}
	var page struct {
		Tab     string `json:"tab"`
		Results []struct {
			Type  string `json:"type"`
			Media []struct {
				Type string `json:"type"`
				URL  string `json:"url"`
			} `json:"media"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &page); err != nil {
		t.Fatal(err)
	}
	if page.Tab != "media" || len(page.Results) != 2 || page.Results[0].Type != "post" || len(page.Results[0].Media) != 3 || page.Results[0].Media[1].URL != "https://video.test/high.mp4" {
		t.Fatalf("page = %s", stdout)
	}
}
