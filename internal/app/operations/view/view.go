// Package view translates one X conversation page into shared canonical posts.
package view

import (
	"context"
	"errors"
	"slices"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/models"
)

type Request struct {
	PostID string
	Cursor *string
}
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Page struct {
	PostID     string        `json:"post_id"`
	Post       *models.Post  `json:"post"`
	Ancestors  []models.Post `json:"ancestors"`
	Replies    []models.Post `json:"replies"`
	NextCursor *string       `json:"next_cursor"`
	Partial    bool          `json:"partial"`
	Warnings   []Warning     `json:"warnings"`
}
type Source interface {
	Fetch(context.Context, Request) (any, error)
}
type Operation struct{ source Source }

func New(source Source) *Operation { return &Operation{source: source} }
func (operation *Operation) Execute(ctx context.Context, request Request) (Page, error) {
	if operation == nil || operation.source == nil {
		return Page{}, failed()
	}
	payload, err := operation.source.Fetch(ctx, request)
	if err != nil {
		var failure *app.OperationFailure
		if errors.As(err, &failure) {
			return Page{}, failure
		}
		return Page{}, failed()
	}
	return DecodePage(payload, request)
}
func failed() *app.OperationFailure {
	return &app.OperationFailure{Code: "VIEW_FAILED", Message: "Unable to view the requested post"}
}
func shapeFailure() *app.OperationFailure {
	return &app.OperationFailure{Code: "RESPONSE_SHAPE_CHANGED", Message: "X returned an unrecognized conversation response"}
}
func objectValue(value any, keys ...string) any {
	for _, key := range keys {
		m, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = m[key]
	}
	return value
}

// DecodePage normalizes an already-fetched conversation response.
func DecodePage(payload any, request Request) (Page, error) {
	if failures, ok := objectValue(payload, "errors").([]any); ok && len(failures) > 0 {
		return Page{}, &app.OperationFailure{Code: "UPSTREAM_REJECTED", Message: "X could not return the requested conversation page"}
	}
	page := Page{PostID: request.PostID, Ancestors: []models.Post{}, Replies: []models.Post{}, Warnings: []Warning{}}
	instructions, ok := objectValue(payload, "data", "threaded_conversation_with_injections_v2", "instructions").([]any)
	if !ok {
		return Page{}, shapeFailure()
	}
	posts := []models.Post{}
	unavailable := false
	appendItem := func(item any, entryID any) {
		if objectValue(item, "promotedMetadata") != nil {
			return
		}
		switch objectValue(item, "itemType") {
		case "TimelinePrompt", "TimelineMessagePrompt":
			return
		}
		result := objectValue(item, "tweet_results", "result")
		if objectValue(result, "__typename") == "TweetWithVisibilityResults" {
			result = objectValue(result, "tweet")
		}
		if kind := objectValue(result, "__typename"); kind == "TweetUnavailable" || kind == "TweetTombstone" || objectValue(item, "itemType") == "TimelineTombstone" {
			if entryID == "tweet-"+request.PostID || objectValue(result, "rest_id") == request.PostID {
				unavailable = true
			}
			page.warn("UNSUPPORTED_CONVERSATION_ITEM", "Some surrounding content or branch continuation could not be returned")
			return
		}
		if post, ok := models.DecodePostResult(result); ok {
			posts = append(posts, post)
		} else {
			page.warn("UNSUPPORTED_CONVERSATION_ITEM", "Some surrounding content or branch continuation could not be returned")
		}
	}
	meaningful := false
	ambiguousCursor := false
	for _, instruction := range instructions {
		switch objectValue(instruction, "type") {
		case "TimelineClearCache":
			continue
		case "TimelineTerminateTimeline":
			meaningful = true
			continue
		case "TimelineAddEntries":
			meaningful = true
		default:
			page.warn("UNSUPPORTED_CONVERSATION_INSTRUCTION", "Some conversation updates could not be returned")
			continue
		}
		entries, ok := objectValue(instruction, "entries").([]any)
		if !ok {
			return Page{}, shapeFailure()
		}
		for _, entry := range entries {
			content := objectValue(entry, "content")
			if objectValue(content, "promotedMetadata") != nil {
				continue
			}
			if objectValue(content, "entryType") == "TimelineTimelineCursor" {
				if kind := objectValue(content, "cursorType"); kind == "ShowMoreThreads" || kind == "Bottom" {
					if value, ok := objectValue(content, "value").(string); ok && value != "" {
						if page.NextCursor != nil && *page.NextCursor != value {
							ambiguousCursor = true
						}
						page.NextCursor = &value
					} else {
						page.warn("UNSUPPORTED_CONVERSATION_CURSOR", "Some forward pagination could not be represented")
					}
				} else if objectValue(content, "cursorType") != "Top" {
					page.warn("UNSUPPORTED_CONVERSATION_CURSOR", "Some forward pagination could not be represented")
				}
				continue
			}
			switch objectValue(content, "entryType") {
			case "TimelineTimelineModule":
				items, valid := objectValue(content, "items").([]any)
				if !valid {
					page.warn("UNSUPPORTED_CONVERSATION_ITEM", "Some surrounding content or branch continuation could not be returned")
				}
				for _, item := range items {
					appendItem(objectValue(item, "item", "itemContent"), objectValue(item, "entryId"))
				}
			case "TimelineTimelineItem":
				appendItem(objectValue(content, "itemContent"), objectValue(entry, "entryId"))
			default:
				page.warn("UNSUPPORTED_CONVERSATION_ITEM", "Some surrounding content or branch continuation could not be returned")
			}
		}
	}
	if !meaningful {
		return Page{}, shapeFailure()
	}
	if unavailable {
		return Page{}, &app.OperationFailure{Code: "POST_UNAVAILABLE", Message: "The requested post is unavailable to the authenticated identity"}
	}
	if ambiguousCursor {
		page.NextCursor = nil
		page.warn("UNSUPPORTED_CONVERSATION_CURSOR", "Some forward pagination could not be represented")
	}
	byID := map[string]models.Post{}
	for _, post := range posts {
		byID[post.ID] = post
		if post.ID == request.PostID {
			page.Post = &post
		}
	}
	if page.Post == nil && request.Cursor == nil {
		return Page{}, shapeFailure()
	}
	ancestors := map[string]bool{}
	if page.Post != nil {
		current := *page.Post
		for parentID(current) != "" {
			parent, ok := byID[parentID(current)]
			if !ok || ancestors[parent.ID] || parent.ID == request.PostID {
				page.warn("INCOMPLETE_PARENT_CHAIN", "Some of the requested post's parent chain could not be returned")
				break
			}
			ancestors[parent.ID] = true
			page.Ancestors = append(page.Ancestors, parent)
			current = parent
		}
		slices.Reverse(page.Ancestors)
	}
	for _, post := range posts {
		if post.ID == request.PostID || ancestors[post.ID] {
			continue
		}
		current := post
		seen := map[string]bool{}
		for parentID(current) != "" {
			if seen[current.ID] {
				page.warn("UNKNOWN_REPLY_RELATIONSHIP", "Some posts could not be placed in the requested conversation")
				break
			}
			seen[current.ID] = true
			if parentID(current) == request.PostID {
				page.Replies = append(page.Replies, post)
				break
			}
			parent, ok := byID[parentID(current)]
			if !ok {
				page.warn("UNKNOWN_REPLY_RELATIONSHIP", "Some posts could not be placed in the requested conversation")
				break
			}
			current = parent
		}
	}
	return page, nil
}

func (page *Page) warn(code, message string) {
	page.Partial = true
	for _, warning := range page.Warnings {
		if warning.Code == code {
			return
		}
	}
	page.Warnings = append(page.Warnings, Warning{Code: code, Message: message})
}

func parentID(post models.Post) string {
	if post.ReplyTo != nil && post.ReplyTo.PostID != nil {
		return *post.ReplyTo.PostID
	}
	return ""
}
