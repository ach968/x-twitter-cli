package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ach968/x-twitter-cli/internal/app/models"
	"github.com/ach968/x-twitter-cli/internal/app/operations/search"
)

// The Search seam deliberately remains the behavioral proof for shared post
// normalization. This type assertion prevents Search from growing a divergent
// second post contract while Bookmarks is introduced.
func TestDecodePageUsesSharedPostContract(t *testing.T) {
	page, err := search.DecodePage(postSource(map[string]any{
		"rest_id": "shared-post",
		"legacy":  map[string]any{"full_text": "Shared"},
	}), "shared", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := page.Results[0].(models.Post); !ok {
		t.Fatalf("result type = %T, want models.Post", page.Results[0])
	}
}

func TestDecodePageNormalizesDirectTopAndLatestPosts(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "test", "testdata", "search-timeline", "populated-initial.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}

	for _, tab := range []search.Tab{search.TabTop, search.TabLatest} {
		page, err := search.DecodePage(source, "from:example", tab)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"query":"from:example","tab":"` + string(tab) + `","results":[{"type":"post","id":"100","url":"https://x.com/example/status/100","text":"Example post with a photo","author":{"id":"10","name":"Example User","username":"example","url":"https://x.com/example","avatar_url":"https://example.com/avatar.jpg","verification":null,"protected":false},"created_at":"1970-01-01T00:00:00Z","language":null,"conversation_id":"100","reply_to":null,"metrics":{"replies":0,"reposts":0,"quotes":0,"likes":1,"bookmarks":null,"views":12},"possibly_sensitive":null,"links":[],"mentions":[],"hashtags":[],"cashtags":[],"media":[],"quoted_post":null,"community_note":null},{"type":"post","id":"101","url":"https://x.com/longform/status/101","text":"Example complete long-form post","author":{"id":"11","name":"Long Form User","username":"longform","url":"https://x.com/longform","avatar_url":"https://example.com/longform-avatar.jpg","verification":"premium","protected":false},"created_at":"1970-01-01T00:00:00Z","language":null,"conversation_id":"101","reply_to":null,"metrics":{"replies":0,"reposts":0,"quotes":0,"likes":2,"bookmarks":null,"views":24},"possibly_sensitive":null,"links":[],"mentions":[],"hashtags":[],"cashtags":[],"media":[],"quoted_post":null,"community_note":null},{"type":"post","id":"102","url":null,"text":"Example tweet nested in a module","author":null,"created_at":null,"language":null,"conversation_id":null,"reply_to":null,"metrics":{"replies":null,"reposts":null,"quotes":null,"likes":null,"bookmarks":null,"views":null},"possibly_sensitive":null,"links":[],"mentions":[],"hashtags":[],"cashtags":[],"media":[],"quoted_post":null,"community_note":null}],"next_cursor":"cursor-bottom-example","warnings":[]}`
		if string(got) != want {
			t.Fatalf("tab %s:\n got %s\nwant %s", tab, got, want)
		}
	}
}

func TestDecodePageNormalizesPostEntitiesReplyAndLargeID(t *testing.T) {
	source := postSource(map[string]any{
		"rest_id": "2095005406548341158",
		"core": map[string]any{"user_results": map[string]any{"result": map[string]any{
			"rest_id": "123", "core": map[string]any{"name": "Author", "screen_name": "author"}, "is_blue_verified": true,
		}}},
		"legacy": map[string]any{
			"created_at": "Mon Sep 01 21:26:00 +0000 2026", "full_text": "Replying to @other about #Go and $X: https://t.co/abc", "lang": "en", "conversation_id_str": "2095000000000000000",
			"in_reply_to_status_id_str": "2095000000000000001", "in_reply_to_user_id_str": "456", "in_reply_to_screen_name": "other", "possibly_sensitive": false,
			"reply_count": float64(0), "retweet_count": float64(4), "quote_count": float64(2), "favorite_count": float64(99), "bookmark_count": float64(0),
			"entities": map[string]any{
				"urls":          []any{map[string]any{"url": "https://t.co/abc", "expanded_url": "https://example.test/article", "display_url": "example.test/article"}},
				"user_mentions": []any{map[string]any{"id_str": "456", "name": "Other", "screen_name": "other"}}, "hashtags": []any{map[string]any{"text": "Go"}}, "symbols": []any{map[string]any{"text": "X"}},
			},
		},
	})

	page, err := search.DecodePage(source, "entities", search.TabLatest)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":"entities","tab":"latest","results":[{"type":"post","id":"2095005406548341158","url":"https://x.com/author/status/2095005406548341158","text":"Replying to @other about #Go and $X: https://t.co/abc","author":{"id":"123","name":"Author","username":"author","url":"https://x.com/author","avatar_url":null,"verification":"premium","protected":null},"created_at":"2026-09-01T21:26:00Z","language":"en","conversation_id":"2095000000000000000","reply_to":{"post_id":"2095000000000000001","user_id":"456","username":"other"},"metrics":{"replies":0,"reposts":4,"quotes":2,"likes":99,"bookmarks":0,"views":null},"possibly_sensitive":false,"links":[{"url":"https://example.test/article","display_url":"example.test/article"}],"mentions":[{"id":"456","name":"Other","username":"other","url":"https://x.com/other","avatar_url":null,"verification":null,"protected":null}],"hashtags":["Go"],"cashtags":["X"],"media":[],"quoted_post":null,"community_note":null}],"next_cursor":null,"warnings":[]}`
	if string(got) != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestDecodePageCleansReaderFacingText(t *testing.T) {
	source := postSource(map[string]any{
		"rest_id": "reader-friendly",
		"core": map[string]any{"user_results": map[string]any{"result": map[string]any{
			"rest_id": "author", "core": map[string]any{"name": "\"Agent\"\nName", "screen_name": "agent"},
		}}},
		"legacy": map[string]any{
			"full_text": "First line\n\n\"Tweet\"    with spacing",
			"extended_entities": map[string]any{"media": []any{map[string]any{
				"id_str": "media", "type": "photo", "media_url_https": "https://img.test/photo.jpg", "ext_alt_text": "Chart\n  called \"Revenue\"",
			}}},
		},
		"birdwatch_pivot": map[string]any{
			"note":     map[string]any{"rest_id": "note"},
			"subtitle": map[string]any{"text": "Context says \"Incorrect\"\nSee source"},
		},
	})

	page, err := search.DecodePage(source, "formatting", search.TabTop)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{
		`"text":"First line “Tweet” with spacing"`,
		`"name":"“Agent” Name"`,
		`"alt_text":"Chart called “Revenue”"`,
		`"text":"Context says “Incorrect” See source"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, `\n`) || strings.Contains(text, `\"`) {
		t.Fatalf("output retains noisy escaped text: %s", text)
	}
}

func postSource(result map[string]any) any {
	return map[string]any{"data": map[string]any{"search_by_raw_query": map[string]any{"search_timeline": map[string]any{"timeline": map[string]any{"instructions": []any{
		map[string]any{"type": "TimelineAddEntries", "entries": []any{map[string]any{"content": map[string]any{"itemContent": map[string]any{"itemType": "TimelineTweet", "tweet_results": map[string]any{"result": result}}}}}},
	}}}}}}
}
