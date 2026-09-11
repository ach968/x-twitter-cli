// Package models owns the stable normalized objects shared by read operations.
// Operation packages own their page envelopes and source timeline grammar.
package models

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxQuoteDepth = 128

type Post struct {
	Type              string         `json:"type"`
	ID                string         `json:"id"`
	URL               *string        `json:"url"`
	Text              string         `json:"text"`
	Author            *UserRef       `json:"author"`
	CreatedAt         *string        `json:"created_at"`
	Language          *string        `json:"language"`
	ConversationID    *string        `json:"conversation_id"`
	ReplyTo           *ReplyTo       `json:"reply_to"`
	Metrics           PostMetrics    `json:"metrics"`
	PossiblySensitive *bool          `json:"possibly_sensitive"`
	Links             []Link         `json:"links"`
	Mentions          []UserRef      `json:"mentions"`
	Hashtags          []string       `json:"hashtags"`
	Cashtags          []string       `json:"cashtags"`
	Media             []Media        `json:"media"`
	QuotedPost        *Post          `json:"quoted_post"`
	CommunityNote     *CommunityNote `json:"community_note"`
}

type UserRef struct {
	ID           *string `json:"id"`
	Name         *string `json:"name"`
	Username     *string `json:"username"`
	URL          *string `json:"url"`
	AvatarURL    *string `json:"avatar_url"`
	Verification *string `json:"verification"`
	Protected    *bool   `json:"protected"`
}
type ReplyTo struct {
	PostID   *string `json:"post_id"`
	UserID   *string `json:"user_id"`
	Username *string `json:"username"`
}
type PostMetrics struct {
	Replies   *int64 `json:"replies"`
	Reposts   *int64 `json:"reposts"`
	Quotes    *int64 `json:"quotes"`
	Likes     *int64 `json:"likes"`
	Bookmarks *int64 `json:"bookmarks"`
	Views     *int64 `json:"views"`
}
type Link struct {
	URL        string  `json:"url"`
	DisplayURL *string `json:"display_url"`
}
type Media struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	URL        *string `json:"url"`
	PreviewURL *string `json:"preview_url"`
	Width      *int64  `json:"width"`
	Height     *int64  `json:"height"`
	AltText    *string `json:"alt_text"`
	DurationMS *int64  `json:"duration_ms"`
}
type CommunityNote struct {
	ID       string  `json:"id"`
	Text     string  `json:"text"`
	Language *string `json:"language"`
	Sources  []Link  `json:"sources"`
	URL      *string `json:"url"`
}

// DecodePostResult constructs the shared post contract from one X post result.
// Source envelope traversal remains the responsibility of the operation.
func DecodePostResult(raw any) (Post, bool) { return decodePost(raw, map[string]struct{}{}, 0) }

func decodePost(raw any, ancestors map[string]struct{}, depth int) (Post, bool) {
	tweet, ok := raw.(map[string]any)
	if !ok {
		return Post{}, false
	}
	legacy, _ := tweet["legacy"].(map[string]any)
	id := stringAt(tweet, "rest_id")
	if id == "" {
		id = stringAt(legacy, "id_str")
	}
	if id == "" || depth >= maxQuoteDepth {
		return Post{}, false
	}
	if _, seen := ancestors[id]; seen {
		return Post{}, false
	}
	ancestors[id] = struct{}{}
	defer delete(ancestors, id)
	text := stringAt(legacy, "full_text")
	entities, _ := legacy["entities"].(map[string]any)
	if note, ok := objectAt(tweet, "note_tweet", "note_tweet_results", "result"); ok {
		if value, present := note["text"].(string); present {
			text = value
		}
		if value, ok := note["entity_set"].(map[string]any); ok {
			entities = value
		}
	}
	author := decodeAuthor(tweet)
	result := Post{Type: "post", ID: id, Text: CleanReaderText(text), Author: author, CreatedAt: normalizeTime(stringAt(legacy, "created_at")), Language: optionalString(legacy, "lang"), ConversationID: optionalString(legacy, "conversation_id_str"), ReplyTo: decodeReplyTo(legacy), Metrics: decodeMetrics(legacy, tweet), PossiblySensitive: optionalBool(legacy, "possibly_sensitive"), Links: decodeLinks(entities), Mentions: decodeMentions(entities), Hashtags: decodeTextEntities(entities, "hashtags"), Cashtags: decodeTextEntities(entities, "symbols"), Media: decodeMedia(legacy), QuotedPost: decodeQuoted(tweet, ancestors, depth), CommunityNote: decodeCommunityNote(tweet)}
	if author != nil && author.Username != nil {
		value := fmt.Sprintf("https://x.com/%s/status/%s", *author.Username, id)
		result.URL = &value
	}
	return result, true
}

func decodeAuthor(tweet map[string]any) *UserRef {
	user, ok := objectAt(tweet, "core", "user_results", "result")
	if !ok {
		return nil
	}
	return DecodeUserRef(user)
}
func DecodeUserRef(raw map[string]any) *UserRef {
	core, _ := raw["core"].(map[string]any)
	username := optionalString(core, "screen_name")
	ref := &UserRef{ID: optionalString(raw, "rest_id"), Name: CleanOptionalText(optionalString(core, "name")), Username: username, AvatarURL: optionalNestedString(raw, "avatar", "image_url"), Verification: SemanticVerification(raw), Protected: optionalNestedBool(raw, "privacy", "protected")}
	if username != nil {
		value := "https://x.com/" + *username
		ref.URL = &value
	}
	return ref
}
func decodeReplyTo(legacy map[string]any) *ReplyTo {
	result := &ReplyTo{PostID: optionalString(legacy, "in_reply_to_status_id_str"), UserID: optionalString(legacy, "in_reply_to_user_id_str"), Username: optionalString(legacy, "in_reply_to_screen_name")}
	if result.PostID == nil && result.UserID == nil && result.Username == nil {
		return nil
	}
	return result
}
func decodeMetrics(legacy, tweet map[string]any) PostMetrics {
	views, _ := tweet["views"].(map[string]any)
	return PostMetrics{Replies: countAt(legacy, "reply_count"), Reposts: countAt(legacy, "retweet_count"), Quotes: countAt(legacy, "quote_count"), Likes: countAt(legacy, "favorite_count"), Bookmarks: countAt(legacy, "bookmark_count"), Views: countAt(views, "count")}
}
func decodeLinks(entities map[string]any) []Link {
	values := []Link{}
	for _, raw := range sliceAt(entities, "urls") {
		entity, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		url := stringAt(entity, "expanded_url")
		if url == "" {
			url = stringAt(entity, "url")
		}
		if url != "" {
			values = append(values, Link{URL: url, DisplayURL: optionalString(entity, "display_url")})
		}
	}
	return values
}
func decodeMentions(entities map[string]any) []UserRef {
	values := []UserRef{}
	for _, raw := range sliceAt(entities, "user_mentions") {
		entity, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		username := optionalString(entity, "screen_name")
		ref := UserRef{ID: optionalString(entity, "id_str"), Name: CleanOptionalText(optionalString(entity, "name")), Username: username}
		if username != nil {
			url := "https://x.com/" + *username
			ref.URL = &url
		}
		values = append(values, ref)
	}
	return values
}
func decodeTextEntities(entities map[string]any, key string) []string {
	values := []string{}
	for _, raw := range sliceAt(entities, key) {
		entity, ok := raw.(map[string]any)
		if ok {
			if text := stringAt(entity, "text"); text != "" {
				values = append(values, text)
			}
		}
	}
	return values
}
func decodeMedia(legacy map[string]any) []Media {
	extended, _ := legacy["extended_entities"].(map[string]any)
	values := []Media{}
	for _, raw := range sliceAt(extended, "media") {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		decoded, ok := decodeMediaItem(item)
		if ok {
			values = append(values, decoded)
		}
	}
	return values
}
func decodeMediaItem(item map[string]any) (Media, bool) {
	id := stringAt(item, "id_str")
	kind, ok := mediaType(stringAt(item, "type"))
	if id == "" || !ok {
		return Media{}, false
	}
	original, _ := item["original_info"].(map[string]any)
	result := Media{ID: id, Type: kind, Width: countAt(original, "width"), Height: countAt(original, "height"), AltText: CleanOptionalText(optionalString(item, "ext_alt_text"))}
	thumbnail := optionalString(item, "media_url_https")
	if kind == "image" {
		result.URL = thumbnail
		return result, true
	}
	result.PreviewURL = thumbnail
	video, _ := item["video_info"].(map[string]any)
	result.URL = highestBitrateMP4(video)
	result.DurationMS = countAt(video, "duration_millis")
	return result, true
}
func mediaType(value string) (string, bool) {
	switch value {
	case "photo":
		return "image", true
	case "video":
		return "video", true
	case "animated_gif":
		return "animated_gif", true
	}
	return "", false
}
func highestBitrateMP4(video map[string]any) *string {
	var selected *string
	var bitrate *int64
	for _, raw := range sliceAt(video, "variants") {
		variant, ok := raw.(map[string]any)
		if !ok || stringAt(variant, "content_type") != "video/mp4" {
			continue
		}
		url := optionalString(variant, "url")
		if url == nil {
			continue
		}
		candidate := countAt(variant, "bitrate")
		if selected == nil || (candidate != nil && (bitrate == nil || *candidate > *bitrate)) {
			selected = url
			bitrate = candidate
		}
	}
	return selected
}
func decodeQuoted(tweet map[string]any, ancestors map[string]struct{}, depth int) *Post {
	quoted, ok := objectAt(tweet, "quoted_status_result", "result")
	if !ok {
		return nil
	}
	result, ok := decodePost(quoted, ancestors, depth+1)
	if !ok {
		return nil
	}
	return &result
}
func decodeCommunityNote(tweet map[string]any) *CommunityNote {
	pivot, ok := tweet["birdwatch_pivot"].(map[string]any)
	if !ok {
		return nil
	}
	note, ok := pivot["note"].(map[string]any)
	if !ok {
		return nil
	}
	id := stringAt(note, "rest_id")
	if id == "" {
		return nil
	}
	subtitle, _ := pivot["subtitle"].(map[string]any)
	text := stringAt(note, "text")
	if text == "" {
		text = stringAt(subtitle, "text")
	}
	if text == "" {
		return nil
	}
	url := fmt.Sprintf("https://x.com/i/communitynotes/n/%s", id)
	return &CommunityNote{ID: id, Text: CleanReaderText(text), Language: optionalString(note, "language"), Sources: decodeNoteLinks(subtitle), URL: &url}
}
func decodeNoteLinks(subtitle map[string]any) []Link {
	values := []Link{}
	for _, raw := range sliceAt(subtitle, "entities") {
		entity, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ref, _ := entity["ref"].(map[string]any)
		url := stringAt(ref, "expandedUrl")
		if url == "" {
			url = stringAt(ref, "expanded_url")
		}
		if url == "" {
			url = stringAt(ref, "url")
		}
		if url == "" {
			continue
		}
		display := optionalString(ref, "displayUrl")
		if display == nil {
			display = optionalString(ref, "display_url")
		}
		values = append(values, Link{URL: url, DisplayURL: display})
	}
	return values
}
func CleanReaderText(value string) string {
	value = html.UnescapeString(value)
	value = strings.Join(strings.Fields(value), " ")
	if !strings.ContainsRune(value, '"') {
		return value
	}
	var output strings.Builder
	output.Grow(len(value))
	opening := true
	for len(value) > 0 {
		r, size := utf8.DecodeRuneInString(value)
		value = value[size:]
		if r != '"' {
			output.WriteRune(r)
			continue
		}
		if opening {
			output.WriteRune('“')
		} else {
			output.WriteRune('”')
		}
		opening = !opening
	}
	return output.String()
}
func CleanOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := CleanReaderText(*value)
	return &cleaned
}
func normalizeTime(value string) *string {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", value)
	if err != nil {
		return nil
	}
	result := parsed.UTC().Format(time.RFC3339)
	return &result
}
func SemanticVerification(user map[string]any) *string {
	if value, _ := user["is_blue_verified"].(bool); value {
		result := "premium"
		return &result
	}
	verification, _ := user["verification"].(map[string]any)
	if value, _ := verification["verified"].(bool); value && stringAt(verification, "verified_type") == "Blue" {
		result := "premium"
		return &result
	}
	return nil
}
func objectAt(root map[string]any, keys ...string) (map[string]any, bool) {
	var current any = root
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current = object[key]
	}
	result, ok := current.(map[string]any)
	return result, ok
}
func stringAt(root map[string]any, key string) string { value, _ := root[key].(string); return value }
func optionalString(root map[string]any, key string) *string {
	value, ok := root[key].(string)
	if !ok {
		return nil
	}
	return &value
}
func optionalNestedString(root map[string]any, keys ...string) *string {
	value, ok := objectAt(root, keys[:len(keys)-1]...)
	if !ok {
		return nil
	}
	return optionalString(value, keys[len(keys)-1])
}
func optionalBool(root map[string]any, key string) *bool {
	value, ok := root[key].(bool)
	if !ok {
		return nil
	}
	return &value
}
func optionalNestedBool(root map[string]any, keys ...string) *bool {
	value, ok := objectAt(root, keys[:len(keys)-1]...)
	if !ok {
		return nil
	}
	return optionalBool(value, keys[len(keys)-1])
}
func sliceAt(root map[string]any, key string) []any { values, _ := root[key].([]any); return values }
func countAt(root map[string]any, key string) *int64 {
	value, ok := root[key]
	if !ok {
		return nil
	}
	var result int64
	switch typed := value.(type) {
	case float64:
		if typed < 0 || typed != float64(int64(typed)) {
			return nil
		}
		result = int64(typed)
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil || parsed < 0 {
			return nil
		}
		result = parsed
	default:
		return nil
	}
	return &result
}
