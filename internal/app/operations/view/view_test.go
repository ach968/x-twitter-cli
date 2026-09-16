package view_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
	view "github.com/ach968/x-twitter-cli/internal/app/operations/view"
)

type source struct{ payload any }

func (s source) Fetch(context.Context, view.Request) (any, error) { return s.payload, nil }

func tweet(id, parent string) map[string]any {
	legacy := map[string]any{"full_text": "Post " + id}
	if parent != "" {
		legacy["in_reply_to_status_id_str"] = parent
	}
	return map[string]any{"__typename": "Tweet", "rest_id": id, "legacy": legacy}
}

func TestContinuationCanOmitContextAndRetainCursorOnAnEmptyPage(t *testing.T) {
	token := "input"
	for _, entries := range [][]any{
		{entry(tweet("200", "100")), cursor("Top", "back"), cursor("Bottom", "forward")},
		{cursor("Bottom", "forward")},
		{},
	} {
		page, err := view.New(source{timeline(entries...)}).Execute(context.Background(), view.Request{PostID: "100", Cursor: &token})
		if err != nil {
			t.Fatal(err)
		}
		if page.Post != nil || page.PostID != "100" || len(page.Ancestors) != 0 || page.Partial {
			t.Fatalf("invalid continuation: %#v", page)
		}
		if len(entries) > 0 && (page.NextCursor == nil || *page.NextCursor != "forward") {
			t.Fatal("lost forward cursor")
		}
	}
}

func TestKnownSurroundingGapsReturnPartialPage(t *testing.T) {
	for name, payload := range map[string]any{
		"missing parent":  timeline(entry(tweet("100", "50"))),
		"parent cycle":    timeline(entry(tweet("100", "50")), entry(tweet("50", "100"))),
		"unknown entry":   timeline(entry(tweet("100", "")), map[string]any{"content": map[string]any{"entryType": "NewEntry"}}),
		"branch cursor":   timeline(entry(tweet("100", "")), map[string]any{"content": map[string]any{"entryType": "TimelineTimelineModule", "items": []any{map[string]any{"item": map[string]any{"itemContent": map[string]any{"itemType": "TimelineTimelineCursor", "cursorType": "ShowMore", "value": "branch"}}}}}}),
		"ambiguous reply": timeline(entry(tweet("100", "")), entry(tweet("201", "200"))),
	} {
		t.Run(name, func(t *testing.T) {
			page, err := view.New(source{payload}).Execute(context.Background(), view.Request{PostID: "100"})
			if err != nil {
				t.Fatal(err)
			}
			if page.Post == nil || !page.Partial || len(page.Warnings) == 0 {
				t.Fatalf("expected usable partial result: %#v", page)
			}
			if name == "branch cursor" && page.NextCursor != nil {
				t.Fatal("branch cursor became page cursor")
			}
		})
	}
}

func TestUnrecognizedContinuationEnvelopeFails(t *testing.T) {
	token := "input"
	payload := map[string]any{"data": map[string]any{"threaded_conversation_with_injections_v2": map[string]any{"instructions": []any{map[string]any{"type": "NewInstruction"}}}}}
	_, err := view.New(source{payload}).Execute(context.Background(), view.Request{PostID: "100", Cursor: &token})
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != "RESPONSE_SHAPE_CHANGED" {
		t.Fatalf("err=%v", err)
	}
}

func TestExplicitRequestedPostUnavailabilityIsNotAnEmptyPage(t *testing.T) {
	token := "continuation"
	for _, c := range []*string{nil, &token} {
		payload := timeline(map[string]any{"entryId": "tweet-100", "content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": map[string]any{"itemType": "TimelineTweet", "tweet_results": map[string]any{"result": map[string]any{"__typename": "TweetUnavailable", "reason": "Unavailable"}}}}})
		_, err := view.New(source{payload}).Execute(context.Background(), view.Request{PostID: "100", Cursor: c})
		var failure *app.OperationFailure
		if !errors.As(err, &failure) || failure.Code != "POST_UNAVAILABLE" {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestVisibilityWrapperAndSurroundingUnavailablePost(t *testing.T) {
	wrapped := map[string]any{"entryId": "tweet-100", "content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": item(map[string]any{"__typename": "TweetWithVisibilityResults", "tweet": tweet("100", "")})}}
	payload := timeline(wrapped, map[string]any{"entryId": "tweet-200", "content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": map[string]any{"itemType": "TimelineTombstone"}}})
	page, err := view.New(source{payload}).Execute(context.Background(), view.Request{PostID: "100"})
	if err != nil || page.Post == nil || !page.Partial {
		t.Fatalf("page=%#v err=%v", page, err)
	}
}

func TestUnavailableSurroundingResultWithAnIDIsNotFabricatedAsAPost(t *testing.T) {
	unavailable := tweet("200", "100")
	unavailable["__typename"] = "TweetUnavailable"
	page, err := view.New(source{timeline(entry(tweet("100", "")), entry(unavailable))}).Execute(context.Background(), view.Request{PostID: "100"})
	if err != nil || !page.Partial || len(page.Replies) != 0 {
		t.Fatalf("replies=%d partial=%t err=%v", len(page.Replies), page.Partial, err)
	}
}

func TestUnknownOrConflictingForwardCursorsWarnInsteadOfGuessing(t *testing.T) {
	for _, extra := range [][]any{{cursor("NewForward", "new")}, {cursor("Bottom", "one"), cursor("Bottom", "two")}} {
		page, err := view.New(source{timeline(append([]any{entry(tweet("100", ""))}, extra...)...)}).Execute(context.Background(), view.Request{PostID: "100"})
		if err != nil || !page.Partial || page.NextCursor != nil {
			t.Fatalf("page=%#v err=%v", page, err)
		}
	}
}

func module(posts ...map[string]any) any {
	items := []any{}
	for _, post := range posts {
		items = append(items, map[string]any{"item": map[string]any{"itemContent": item(post)}})
	}
	return map[string]any{"entryId": "conversationthread-test", "content": map[string]any{"entryType": "TimelineTimelineModule", "items": items}}
}

func TestReplyTargetSeparatesAncestorsAndNestedRepliesWithoutFlatteningQuotes(t *testing.T) {
	quoted := tweet("900", "")
	target := tweet("100", "50")
	target["quoted_status_result"] = map[string]any{"result": quoted}
	page, err := view.New(source{timeline(
		entry(tweet("50", "25")), entry(tweet("25", "")), entry(target),
		module(tweet("300", "100"), tweet("301", "300")),
		module(tweet("200", "100")), entry(tweet("777", "50")),
	)}).Execute(context.Background(), view.Request{PostID: "100"})
	if err != nil {
		t.Fatal(err)
	}
	ancestors := []string{}
	for _, post := range page.Ancestors {
		ancestors = append(ancestors, post.ID)
	}
	replies := []string{}
	for _, post := range page.Replies {
		replies = append(replies, post.ID)
	}
	if !reflect.DeepEqual(ancestors, []string{"25", "50"}) || !reflect.DeepEqual(replies, []string{"300", "301", "200"}) {
		t.Fatalf("ancestors=%v replies=%v", ancestors, replies)
	}
	if page.Post.QuotedPost == nil || page.Post.QuotedPost.ID != "900" || page.Partial {
		t.Fatalf("quote or completeness wrong: %#v", page)
	}
}
func item(post map[string]any) map[string]any {
	return map[string]any{"itemType": "TimelineTweet", "tweet_results": map[string]any{"result": post}}
}
func entry(post map[string]any) any {
	return map[string]any{"entryId": "tweet-" + post["rest_id"].(string), "content": map[string]any{"entryType": "TimelineTimelineItem", "itemContent": item(post)}}
}
func cursor(kind, value string) any {
	return map[string]any{"entryId": "cursor-" + kind, "content": map[string]any{"entryType": "TimelineTimelineCursor", "cursorType": kind, "value": value}}
}
func timeline(entries ...any) any {
	return map[string]any{"data": map[string]any{"threaded_conversation_with_injections_v2": map[string]any{"instructions": []any{
		map[string]any{"type": "TimelineAddEntries", "entries": entries},
	}}}}
}

func TestInitialPageReturnsRequestedPostAndNativeCursor(t *testing.T) {
	page, err := view.New(source{timeline(entry(tweet("100", "")), cursor("ShowMoreThreads", "opaque-first"))}).Execute(context.Background(), view.Request{PostID: "100"})
	if err != nil {
		t.Fatal(err)
	}
	if page.PostID != "100" || page.Post == nil || page.Post.ID != "100" || page.Post.Text != "Post 100" {
		t.Fatalf("wrong requested post: %#v", page)
	}
	if page.NextCursor == nil || *page.NextCursor != "opaque-first" || page.Partial || len(page.Warnings) != 0 {
		t.Fatalf("wrong pagination: %#v", page)
	}
	if page.Ancestors == nil || page.Replies == nil || page.Warnings == nil {
		t.Fatal("empty collections must encode as arrays")
	}
}
