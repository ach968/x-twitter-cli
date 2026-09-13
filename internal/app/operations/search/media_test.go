package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ach968/x-twitter-cli3/internal/app/operations/search"
)

func TestDecodePageNormalizesMediaGridAndAttachedMedia(t *testing.T) {
	sourceText, err := os.ReadFile(filepath.Join("testdata", "media-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(sourceText, &source); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "media-golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	page, err := search.DecodePage(source, "cats", search.TabMedia)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != strings.TrimSpace(string(want)) {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}
