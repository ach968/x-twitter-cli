// Package search translates X SearchTimeline source payloads into the stable
// search-page contract exposed by the application.
package search

import (
	"context"
	"errors"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
)

type Tab string

const (
	TabTop    Tab = "top"
	TabLatest Tab = "latest"
	TabPeople Tab = "people"
	TabMedia  Tab = "media"
	TabLists  Tab = "lists"
)

func ParseTab(value string) (Tab, bool) {
	switch strings.ToLower(value) {
	case string(TabTop):
		return TabTop, true
	case string(TabLatest):
		return TabLatest, true
	case string(TabPeople):
		return TabPeople, true
	case string(TabMedia):
		return TabMedia, true
	case string(TabLists):
		return TabLists, true
	default:
		return "", false
	}
}

func (tab Tab) Product() string {
	switch tab {
	case TabTop:
		return "Top"
	case TabLatest:
		return "Latest"
	case TabPeople:
		return "People"
	case TabMedia:
		return "Media"
	case TabLists:
		return "Lists"
	default:
		return ""
	}
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Page struct {
	Query      string    `json:"query"`
	Tab        Tab       `json:"tab"`
	Results    []any     `json:"results"`
	NextCursor *string   `json:"next_cursor"`
	Warnings   []Warning `json:"warnings"`
}

// Request is the stable caller input for one Search page.
type Request struct {
	Query  string
	Tab    Tab
	Cursor *string
}

// Source is the operation-private seam for retrieving a SearchTimeline source
// payload. It intentionally exposes no direct-request details to callers.
type Source interface {
	SearchTimeline(context.Context, Request) (any, error)
}

// Operation owns Search request policy, source interpretation, and normalized
// page construction.
type Operation struct {
	source Source
}

func New(source Source) *Operation {
	return &Operation{source: source}
}

func (operation *Operation) Execute(ctx context.Context, request Request) (Page, error) {
	if operation == nil || operation.source == nil {
		return Page{}, &app.OperationFailure{Code: "SEARCH_FAILED", Message: "Unable to execute search"}
	}
	source, err := operation.source.SearchTimeline(ctx, request)
	if err != nil {
		var failure *app.OperationFailure
		if errors.As(err, &failure) {
			return Page{}, failure
		}
		return Page{}, &app.OperationFailure{Code: "SEARCH_FAILED", Message: "Unable to execute search"}
	}
	page, err := DecodePage(source, request.Query, request.Tab)
	if err != nil {
		return Page{}, &app.OperationFailure{Code: "RESPONSE_SHAPE_CHANGED", Message: "X returned an unrecognized SearchTimeline response"}
	}
	return page, nil
}

// DecodePage is the normalization seam. Source payload details stay private to
// this package so callers only learn the stable Search page contract.
func DecodePage(source any, query string, tab Tab) (Page, error) {
	root, ok := source.(map[string]any)
	if !ok {
		return Page{}, errors.New("invalid SearchTimeline envelope")
	}
	instructions, ok := nestedSlice(root, "data", "search_by_raw_query", "search_timeline", "timeline", "instructions")
	if !ok {
		return Page{}, errors.New("invalid SearchTimeline envelope")
	}

	decodedEntries := []timelineDecodedEntry{}
	meaningful := false
	var nextCursor *string
	activeCursorEntryID := ""
	for _, rawInstruction := range instructions {
		instruction, ok := rawInstruction.(map[string]any)
		if !ok {
			return Page{}, errors.New("invalid SearchTimeline instruction")
		}
		switch instruction["type"] {
		case "TimelineAddEntries":
			entries, ok := instruction["entries"].([]any)
			if !ok {
				return Page{}, errors.New("invalid TimelineAddEntries instruction")
			}
			meaningful = true
			for _, entry := range entries {
				if cursor, ok := bottomCursor(entry); ok {
					nextCursor = &cursor
					activeCursorEntryID = timelineEntryID(entry)
				}
				decoded, err := decodeEntry(entry)
				if err != nil {
					return Page{}, err
				}
				decodedEntries = append(decodedEntries, timelineDecodedEntry{id: timelineEntryID(entry), decoded: decoded})
			}
		case "TimelineReplaceEntry":
			entry, ok := instruction["entry"].(map[string]any)
			if !ok {
				return Page{}, errors.New("invalid TimelineReplaceEntry instruction")
			}
			decoded, err := decodeEntry(entry)
			if err != nil {
				return Page{}, err
			}
			target, ok := instruction["entry_id_to_replace"].(string)
			if !ok || target == "" {
				return Page{}, errors.New("invalid TimelineReplaceEntry target")
			}
			meaningful = true
			if target == activeCursorEntryID {
				nextCursor = nil
				activeCursorEntryID = ""
			}
			if cursor, ok := bottomCursor(entry); ok {
				nextCursor = &cursor
				activeCursorEntryID = timelineEntryID(entry)
			}
			replacement := timelineDecodedEntry{id: timelineEntryID(entry), decoded: decoded}
			if index := findTimelineEntry(decodedEntries, target); index >= 0 {
				decodedEntries[index] = replacement
			} else if decoded.hasPublicEffect() {
				decodedEntries = append(decodedEntries, replacement)
			}
		}
	}
	if !meaningful {
		return Page{}, errors.New("no meaningful SearchTimeline instructions")
	}
	results := []any{}
	warnings := []Warning{}
	unsupportedResults := 0
	for _, entry := range decodedEntries {
		results = append(results, entry.decoded.results...)
		warnings = append(warnings, entry.decoded.warnings...)
		unsupportedResults += entry.decoded.unsupportedResults
	}
	if len(results) == 0 && unsupportedResults > 0 {
		return Page{}, errors.New("no meaningful SearchTimeline results")
	}
	return Page{Query: query, Tab: tab, Results: results, NextCursor: nextCursor, Warnings: warnings}, nil
}

type timelineDecodedEntry struct {
	id      string
	decoded decodedEntry
}

func findTimelineEntry(entries []timelineDecodedEntry, id string) int {
	for index, entry := range entries {
		if entry.id == id {
			return index
		}
	}
	return -1
}

func timelineEntryID(raw any) string {
	entry, _ := raw.(map[string]any)
	value, _ := entry["entryId"].(string)
	return value
}

type decodedEntry struct {
	results            []any
	warnings           []Warning
	unsupportedResults int
}

func (entry decodedEntry) hasPublicEffect() bool {
	return len(entry.results) > 0 || len(entry.warnings) > 0 || entry.unsupportedResults > 0
}

func decodeEntry(raw any) (decodedEntry, error) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return decodedEntry{}, errors.New("invalid SearchTimeline entry")
	}
	content, ok := entry["content"].(map[string]any)
	if !ok {
		return decodedEntry{}, errors.New("invalid SearchTimeline entry content")
	}
	if rawItem, present := content["itemContent"]; present {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return decodedEntry{}, errors.New("invalid SearchTimeline item content")
		}
		return decodeResultItem(item), nil
	}
	if content["entryType"] == "TimelineTimelineCursor" {
		return decodedEntry{}, nil
	}
	if content["entryType"] != "TimelineTimelineModule" {
		return decodedEntry{
			warnings:           []Warning{{Code: "UNKNOWN_SEARCH_ENTRY", Message: "Skipped an unsupported search entry"}},
			unsupportedResults: 1,
		}, nil
	}
	decoded := decodedEntry{}
	items, ok := content["items"].([]any)
	if !ok {
		return decodedEntry{}, errors.New("invalid SearchTimeline module items")
	}
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return decodedEntry{}, errors.New("invalid SearchTimeline module item")
		}
		itemContent, ok := objectAt(item, "item", "itemContent")
		if !ok {
			return decodedEntry{}, errors.New("invalid SearchTimeline module item content")
		}
		itemDecoded := decodeResultItem(itemContent)
		decoded.results = append(decoded.results, itemDecoded.results...)
		decoded.warnings = append(decoded.warnings, itemDecoded.warnings...)
		decoded.unsupportedResults += itemDecoded.unsupportedResults
	}
	return decoded, nil
}

func decodeResultItem(item map[string]any) decodedEntry {
	if result, ok := decodeItem(item); ok {
		return decodedEntry{results: []any{result}}
	}
	if presentationItem(item) {
		return decodedEntry{}
	}
	return decodedEntry{
		warnings:           []Warning{{Code: "UNKNOWN_SEARCH_ENTRY", Message: "Skipped an unsupported search entry"}},
		unsupportedResults: 1,
	}
}

func presentationItem(item map[string]any) bool {
	switch item["itemType"] {
	case "TimelinePrompt", "TimelineMessagePrompt", "TimelineShowAlert":
		return true
	default:
		return false
	}
}

func decodeItem(item map[string]any) (any, bool) {
	if user, ok := decodeUserResult(item); ok {
		return user, true
	}
	if list, ok := decodeListResult(item); ok {
		return list, true
	}
	if item["itemType"] != "TimelineTweet" {
		return nil, false
	}
	result, ok := objectAt(item, "tweet_results", "result")
	if !ok {
		return nil, false
	}
	post, ok := decodePostResult(result)
	return post, ok
}

func nestedSlice(root map[string]any, keys ...string) ([]any, bool) {
	var current any = root
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[key]
		if !ok {
			return nil, false
		}
	}
	value, ok := current.([]any)
	return value, ok
}

func bottomCursor(raw any) (string, bool) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return "", false
	}
	content, ok := entry["content"].(map[string]any)
	if !ok || content["cursorType"] != "Bottom" {
		return "", false
	}
	value, ok := content["value"].(string)
	return value, ok
}
