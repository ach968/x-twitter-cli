// Package bookmarks translates X bookmark timelines into the stable Bookmarks
// page contract.
package bookmarks

import (
	"context"
	"errors"

	app "github.com/ach968/x-twitter-cli3/internal/app"
	"github.com/ach968/x-twitter-cli3/internal/app/models"
)

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Page struct {
	Query      *string       `json:"query"`
	Bookmarks  []models.Post `json:"bookmarks"`
	NextCursor *string       `json:"next_cursor"`
	Warnings   []Warning     `json:"warnings"`
}

type Request struct {
	Query  *string
	Cursor *string
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
	source, err := operation.source.Fetch(ctx, request)
	if err != nil {
		var failure *app.OperationFailure
		if errors.As(err, &failure) {
			return Page{}, failure
		}
		return Page{}, failed()
	}
	page, err := decodePage(source, request.Query)
	if err != nil {
		return Page{}, &app.OperationFailure{Code: "RESPONSE_SHAPE_CHANGED", Message: "X returned an unrecognized Bookmarks response"}
	}
	return page, nil
}

func failed() *app.OperationFailure {
	return &app.OperationFailure{Code: "BOOKMARKS_FAILED", Message: "Unable to execute bookmarks"}
}

func decodePage(source any, query *string) (Page, error) {
	root, ok := source.(map[string]any)
	if !ok {
		return Page{}, errors.New("invalid Bookmarks envelope")
	}
	instructions, ok := bookmarkInstructions(root, query != nil)
	if !ok {
		return Page{}, errors.New("invalid Bookmarks envelope")
	}
	posts := []models.Post{}
	warnings := []Warning{}
	var cursor *string
	meaningful := false
	unsupported := 0
	for _, raw := range instructions {
		instruction, ok := raw.(map[string]any)
		if !ok {
			return Page{}, errors.New("invalid instruction")
		}
		if instruction["type"] != "TimelineAddEntries" {
			continue
		}
		entries, ok := instruction["entries"].([]any)
		if !ok {
			return Page{}, errors.New("invalid entries")
		}
		meaningful = true
		for _, rawEntry := range entries {
			post, next, warning, unknown, err := decodeEntry(rawEntry)
			if err != nil {
				return Page{}, err
			}
			if post != nil {
				posts = append(posts, *post)
			}
			if next != nil {
				cursor = next
			}
			if warning != nil {
				warnings = append(warnings, *warning)
			}
			unsupported += unknown
		}
	}
	if !meaningful || (len(posts) == 0 && unsupported > 0) {
		return Page{}, errors.New("nothing meaningful decoded")
	}
	return Page{Query: query, Bookmarks: posts, NextCursor: cursor, Warnings: warnings}, nil
}

func bookmarkInstructions(root map[string]any, search bool) ([]any, bool) {
	var current any = root["data"]
	if search {
		current = objectValue(current, "search_by_raw_query", "bookmarks_search_timeline", "timeline", "instructions")
	} else {
		current = objectValue(current, "bookmark_timeline_v2", "timeline", "instructions")
	}
	values, ok := current.([]any)
	return values, ok
}

func objectValue(value any, keys ...string) any {
	for _, key := range keys {
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = object[key]
	}
	return value
}

func decodeEntry(raw any) (*models.Post, *string, *Warning, int, error) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return nil, nil, nil, 0, errors.New("invalid entry")
	}
	content, ok := entry["content"].(map[string]any)
	if !ok {
		return nil, nil, nil, 0, errors.New("invalid entry content")
	}
	if content["entryType"] == "TimelineTimelineCursor" {
		if content["cursorType"] == "Bottom" {
			if value, ok := content["value"].(string); ok {
				return nil, &value, nil, 0, nil
			}
		}
		return nil, nil, nil, 0, nil
	}
	item, ok := content["itemContent"].(map[string]any)
	if !ok {
		return nil, nil, &Warning{Code: "UNKNOWN_BOOKMARK_ENTRY", Message: "Skipped an unsupported bookmark entry"}, 1, nil
	}
	if item["itemType"] == "TimelinePrompt" || item["itemType"] == "TimelineMessagePrompt" {
		return nil, nil, nil, 0, nil
	}
	result := objectValue(item, "tweet_results", "result")
	if post, ok := models.DecodePostResult(result); ok {
		return &post, nil, nil, 0, nil
	}
	return nil, nil, &Warning{Code: "UNKNOWN_BOOKMARK_ENTRY", Message: "Skipped an unsupported bookmark entry"}, 1, nil
}
