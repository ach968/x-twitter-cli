package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ach968/x-twitter-cli/internal/app/operations/search"
)

func TestDecodePageNormalizesQuotesNotesAndConversationModule(t *testing.T) {
	sourceText, err := os.ReadFile(filepath.Join("testdata", "quotes-notes-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(sourceText, &source); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "quotes-notes-golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	page, err := search.DecodePage(source, "context", search.TabTop)
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

func TestDecodePageBoundsCyclicQuotedPosts(t *testing.T) {
	tweet := map[string]any{"rest_id": "cycle", "legacy": map[string]any{"full_text": "Cycle"}}
	tweet["quoted_status_result"] = map[string]any{"result": tweet}
	source := map[string]any{"data": map[string]any{"search_by_raw_query": map[string]any{"search_timeline": map[string]any{"timeline": map[string]any{"instructions": []any{
		map[string]any{"type": "TimelineAddEntries", "entries": []any{map[string]any{"content": map[string]any{"itemContent": map[string]any{"itemType": "TimelineTweet", "tweet_results": map[string]any{"result": tweet}}}}}},
	}}}}}}

	page, err := search.DecodePage(source, "cycle", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":"cycle","tab":"top","results":[{"type":"post","id":"cycle","url":null,"text":"Cycle","author":null,"created_at":null,"language":null,"conversation_id":null,"reply_to":null,"metrics":{"replies":null,"reposts":null,"quotes":null,"likes":null,"bookmarks":null,"views":null},"possibly_sensitive":null,"links":[],"mentions":[],"hashtags":[],"cashtags":[],"media":[],"quoted_post":null,"community_note":null}],"next_cursor":null,"warnings":[]}`
	if string(got) != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}
