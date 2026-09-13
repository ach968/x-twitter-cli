package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ach968/x-twitter-cli/internal/app/operations/search"
)

func TestDecodePageReturnsCompleteEmptySearchPage(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "test", "testdata", "search-timeline", "empty-initial.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}

	page, err := search.DecodePage(source, "quiet query", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":"quiet query","tab":"top","results":[],"next_cursor":"cursor-bottom-empty-example","warnings":[]}`
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("page = %s, want %s", got, want)
	}
}

func TestDecodePageRejectsInvalidEnvelope(t *testing.T) {
	if _, err := search.DecodePage(map[string]any{"data": map[string]any{}}, "query", search.TabTop); err == nil {
		t.Fatal("expected invalid envelope to fail")
	}
}
