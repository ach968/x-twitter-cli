package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ach968/x-twt-cli/internal/app/search"
)

func TestDecodePageKeepsKnownResultsInOrderAndReturnsUsablePartialPage(t *testing.T) {
	source := readPaginationFixture(t, "pagination-partial-source.json")

	for _, tab := range []search.Tab{search.TabTop, search.TabLatest, search.TabPeople, search.TabMedia, search.TabLists} {
		page, err := search.DecodePage(source, "query", tab)
		if err != nil {
			t.Fatalf("tab %s: %v", tab, err)
		}
		if page.NextCursor == nil || *page.NextCursor != "new-bottom" {
			t.Fatalf("tab %s: next cursor = %#v", tab, page.NextCursor)
		}
		if got := resultTypes(t, page); !reflect.DeepEqual(got, []string{"post", "user", "list"}) {
			t.Fatalf("tab %s: result types = %#v", tab, got)
		}
		if !reflect.DeepEqual(page.Warnings, []search.Warning{{Code: "UNKNOWN_SEARCH_ENTRY", Message: "Skipped an unsupported search entry"}}) {
			t.Fatalf("tab %s: warnings = %#v", tab, page.Warnings)
		}
	}
}

func TestDecodePageRejectsUnsupportedResultModule(t *testing.T) {
	itemContent := map[string]any{"itemType": "TimelineNewResult"}
	moduleItem := map[string]any{"item": map[string]any{"itemContent": itemContent}}
	content := map[string]any{
		"entryType":   "TimelineTimelineModule",
		"displayType": "NewResultModule",
		"items":       []any{moduleItem},
	}
	instruction := map[string]any{
		"type":    "TimelineAddEntries",
		"entries": []any{map[string]any{"content": content}},
	}
	source := sourceWithInstructions([]any{instruction})
	if _, err := search.DecodePage(source, "query", search.TabTop); err == nil {
		t.Fatal("expected an unsupported result module to fail")
	}
}

func TestDecodePageRejectsInstructionsWithNoMeaningfulContent(t *testing.T) {
	source := sourceWithInstructions([]any{})
	if _, err := search.DecodePage(source, "query", search.TabTop); err == nil {
		t.Fatal("expected an instructionless page to fail")
	}
}

func TestDecodePageRejectsMalformedKnownInstructions(t *testing.T) {
	for _, instruction := range []any{
		map[string]any{"type": "TimelineAddEntries"},
		map[string]any{"type": "TimelineAddEntries", "entries": "not an array"},
		map[string]any{"type": "TimelineReplaceEntry"},
		map[string]any{"type": "TimelineReplaceEntry", "entry": map[string]any{}},
		map[string]any{"type": "TimelineReplaceEntry", "entry": map[string]any{"content": "not an object"}},
		map[string]any{"type": "TimelineReplaceEntry", "entry": map[string]any{"content": map[string]any{"entryType": "TimelineTimelineCursor", "cursorType": "Top", "value": "top"}}},
		"not an instruction",
	} {
		if _, err := search.DecodePage(sourceWithInstructions([]any{instruction}), "query", search.TabTop); err == nil {
			t.Fatalf("expected malformed instruction %#v to fail", instruction)
		}
	}
}

func TestDecodePageRejectsMalformedTimelineEntries(t *testing.T) {
	for _, entry := range []any{
		"not an entry",
		map[string]any{"entryId": "missing-content"},
		map[string]any{"content": "not an object"},
		map[string]any{"content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": "not an object"}},
		map[string]any{"content": map[string]any{"entryType": "TimelineTimelineModule", "items": "not an array"}},
	} {
		instruction := map[string]any{"type": "TimelineAddEntries", "entries": []any{entry}}
		if _, err := search.DecodePage(sourceWithInstructions([]any{instruction}), "query", search.TabTop); err == nil {
			t.Fatalf("expected malformed entry %#v to fail", entry)
		}
	}
}

func TestDecodePageAcceptsPresentationOnlyEmptyPage(t *testing.T) {
	entries := []any{
		map[string]any{"content": map[string]any{"entryType": "TimelineTimelineCursor", "cursorType": "Top", "value": "top"}},
		map[string]any{"content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": map[string]any{"itemType": "TimelinePrompt"}}},
	}
	instruction := map[string]any{"type": "TimelineAddEntries", "entries": entries}
	page, err := search.DecodePage(sourceWithInstructions([]any{instruction}), "query", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 0 || len(page.Warnings) != 0 {
		t.Fatalf("presentation-only page = %#v", page)
	}
}

func TestDecodePageClearsReplacedActiveBottomCursor(t *testing.T) {
	add := map[string]any{
		"type": "TimelineAddEntries",
		"entries": []any{map[string]any{
			"entryId": "cursor-bottom-old",
			"content": map[string]any{"entryType": "TimelineTimelineCursor", "cursorType": "Bottom", "value": "obsolete"},
		}},
	}
	replace := map[string]any{
		"type":                "TimelineReplaceEntry",
		"entry_id_to_replace": "cursor-bottom-old",
		"entry": map[string]any{
			"entryId": "prompt-replacement",
			"content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": map[string]any{"itemType": "TimelinePrompt"}},
		},
	}
	page, err := search.DecodePage(sourceWithInstructions([]any{add, replace}), "query", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor != nil {
		t.Fatalf("next cursor = %q, want null", *page.NextCursor)
	}
}

func TestDecodePageAppliesResultReplacementInPlace(t *testing.T) {
	add := map[string]any{
		"type":    "TimelineAddEntries",
		"entries": []any{postTimelineEntry("a", "post-a"), postTimelineEntry("b", "post-b")},
	}
	replace := map[string]any{
		"type":                "TimelineReplaceEntry",
		"entry_id_to_replace": "a",
		"entry":               postTimelineEntry("a-new", "post-a-new"),
	}
	page, err := search.DecodePage(sourceWithInstructions([]any{add, replace}), "query", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	if got := resultIDs(t, page); !reflect.DeepEqual(got, []string{"post-a-new", "post-b"}) {
		t.Fatalf("result IDs = %#v", got)
	}
}

func postTimelineEntry(entryID, postID string) map[string]any {
	post := map[string]any{"rest_id": postID, "legacy": map[string]any{"full_text": postID}}
	item := map[string]any{"itemType": "TimelineTweet", "tweet_results": map[string]any{"result": post}}
	return map[string]any{"entryId": entryID, "content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": item}}
}

func sourceWithInstructions(instructions []any) any {
	timeline := map[string]any{"instructions": instructions}
	searchTimeline := map[string]any{"timeline": timeline}
	searchByRawQuery := map[string]any{"search_timeline": searchTimeline}
	return map[string]any{"data": map[string]any{"search_by_raw_query": searchByRawQuery}}
}

func TestDecodePageRejectsPageContainingOnlyUnsupportedResults(t *testing.T) {
	source := readPaginationFixture(t, "pagination-unknown-only-source.json")
	if _, err := search.DecodePage(source, "query", search.TabTop); err == nil {
		t.Fatal("expected an unsupported-result-only page to fail")
	}
}

func readPaginationFixture(t *testing.T, name string) any {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}
	return source
}

func resultTypes(t *testing.T, page search.Page) []string {
	t.Helper()
	types := make([]string, 0, len(page.Results))
	for _, result := range page.Results {
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		types = append(types, decoded.Type)
	}
	return types
}

func resultIDs(t *testing.T, page search.Page) []string {
	t.Helper()
	ids := make([]string, 0, len(page.Results))
	for _, result := range page.Results {
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, decoded.ID)
	}
	return ids
}
