package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ach968/x-twt-cli/internal/app/operations/search"
)

func TestDecodePageKeepsUnobservedAccountSignalsUnknown(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "test", "testdata", "search-timeline", "people-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}

	page, err := search.DecodePage(source, "example people", search.TabPeople)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	var normalized struct {
		Results []struct {
			ID                  string  `json:"id"`
			Verification        *string `json:"verification"`
			IdentityVerified    *bool   `json:"identity_verified"`
			ParodyCommentaryFan *string `json:"parody_commentary_fan"`
		} `json:"results"`
	}
	if err := json.Unmarshal(got, &normalized); err != nil {
		t.Fatal(err)
	}
	if len(normalized.Results) != 3 {
		t.Fatalf("results=%s", got)
	}
	partial := normalized.Results[2]
	if partial.ID != "102" || partial.Verification != nil || partial.IdentityVerified != nil || partial.ParodyCommentaryFan != nil {
		t.Fatalf("unobserved account signals = %#v", partial)
	}
}
