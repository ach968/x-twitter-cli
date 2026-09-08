package search

import (
	"fmt"
	"strconv"
	"time"
)

type postResult struct {
	Type              string         `json:"type"`
	ID                string         `json:"id"`
	URL               *string        `json:"url"`
	Text              string         `json:"text"`
	Author            *userRef       `json:"author"`
	CreatedAt         *string        `json:"created_at"`
	Language          *string        `json:"language"`
	ConversationID    *string        `json:"conversation_id"`
	ReplyTo           *replyTo       `json:"reply_to"`
	Metrics           postMetrics    `json:"metrics"`
	PossiblySensitive *bool          `json:"possibly_sensitive"`
	Links             []link         `json:"links"`
	Mentions          []userRef      `json:"mentions"`
	Hashtags          []string       `json:"hashtags"`
	Cashtags          []string       `json:"cashtags"`
	Media             []mediaResult  `json:"media"`
	QuotedPost        *postResult    `json:"quoted_post"`
	CommunityNote     *communityNote `json:"community_note"`
}

type userRef struct {
	ID           *string `json:"id"`
	Name         *string `json:"name"`
	Username     *string `json:"username"`
	URL          *string `json:"url"`
	AvatarURL    *string `json:"avatar_url"`
	Verification *string `json:"verification"`
	Protected    *bool   `json:"protected"`
}

type replyTo struct {
	PostID   *string `json:"post_id"`
	UserID   *string `json:"user_id"`
	Username *string `json:"username"`
}

type postMetrics struct {
	Replies   *int64 `json:"replies"`
	Reposts   *int64 `json:"reposts"`
	Quotes    *int64 `json:"quotes"`
	Likes     *int64 `json:"likes"`
	Bookmarks *int64 `json:"bookmarks"`
	Views     *int64 `json:"views"`
}

type link struct {
	URL        string  `json:"url"`
	DisplayURL *string `json:"display_url"`
}

func decodePostResult(raw any) (postResult, bool) {
	return decodePostResultWithGuard(raw, map[string]struct{}{}, 0)
}

func decodePostResultWithGuard(raw any, ancestors map[string]struct{}, depth int) (postResult, bool) {
	tweet, ok := raw.(map[string]any)
	if !ok {
		return postResult{}, false
	}
	legacy, _ := tweet["legacy"].(map[string]any)
	id := stringAt(tweet, "rest_id")
	if id == "" {
		id = stringAt(legacy, "id_str")
	}
	if id == "" {
		return postResult{}, false
	}
	if depth >= maxQuoteDepth {
		return postResult{}, false
	}
	if _, seen := ancestors[id]; seen {
		return postResult{}, false
	}
	ancestors[id] = struct{}{}
	defer delete(ancestors, id)

	text := stringAt(legacy, "full_text")
	entities, _ := legacy["entities"].(map[string]any)
	if note, ok := objectAt(tweet, "note_tweet", "note_tweet_results", "result"); ok {
		if noteText, present := note["text"].(string); present {
			text = noteText
		}
		if noteEntities, ok := note["entity_set"].(map[string]any); ok {
			entities = noteEntities
		}
	}

	author := decodeTweetAuthor(tweet)
	result := postResult{
		Type:              "post",
		ID:                id,
		Text:              text,
		Author:            author,
		CreatedAt:         normalizeTime(stringAt(legacy, "created_at")),
		Language:          optionalString(legacy, "lang"),
		ConversationID:    optionalString(legacy, "conversation_id_str"),
		ReplyTo:           decodeReplyTo(legacy),
		Metrics:           decodePostMetrics(legacy, tweet),
		PossiblySensitive: optionalBool(legacy, "possibly_sensitive"),
		Links:             decodeLinks(entities),
		Mentions:          decodeMentions(entities),
		Hashtags:          decodeTextEntities(entities, "hashtags"),
		Cashtags:          decodeTextEntities(entities, "symbols"),
		Media:             decodeMedia(legacy),
		QuotedPost:        decodeQuotedPost(tweet, ancestors, depth),
		CommunityNote:     decodeCommunityNote(tweet),
	}
	if author != nil && author.Username != nil {
		url := fmt.Sprintf("https://x.com/%s/status/%s", *author.Username, id)
		result.URL = &url
	}
	return result, true
}

func decodeTweetAuthor(tweet map[string]any) *userRef {
	user, ok := objectAt(tweet, "core", "user_results", "result")
	if !ok {
		return nil
	}
	return decodeUserRef(user)
}

func decodeUserRef(raw map[string]any) *userRef {
	core, _ := raw["core"].(map[string]any)
	username := optionalString(core, "screen_name")
	ref := &userRef{
		ID:           optionalString(raw, "rest_id"),
		Name:         optionalString(core, "name"),
		Username:     username,
		AvatarURL:    optionalNestedString(raw, "avatar", "image_url"),
		Verification: semanticVerification(raw),
		Protected:    optionalNestedBool(raw, "privacy", "protected"),
	}
	if username != nil {
		url := "https://x.com/" + *username
		ref.URL = &url
	}
	return ref
}

func decodeReplyTo(legacy map[string]any) *replyTo {
	postID := optionalString(legacy, "in_reply_to_status_id_str")
	userID := optionalString(legacy, "in_reply_to_user_id_str")
	username := optionalString(legacy, "in_reply_to_screen_name")
	if postID == nil && userID == nil && username == nil {
		return nil
	}
	return &replyTo{PostID: postID, UserID: userID, Username: username}
}

func decodePostMetrics(legacy, tweet map[string]any) postMetrics {
	views, _ := tweet["views"].(map[string]any)
	return postMetrics{
		Replies:   countAt(legacy, "reply_count"),
		Reposts:   countAt(legacy, "retweet_count"),
		Quotes:    countAt(legacy, "quote_count"),
		Likes:     countAt(legacy, "favorite_count"),
		Bookmarks: countAt(legacy, "bookmark_count"),
		Views:     countAt(views, "count"),
	}
}

func decodeLinks(entities map[string]any) []link {
	links := []link{}
	for _, raw := range sliceAt(entities, "urls") {
		entity, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		url := stringAt(entity, "expanded_url")
		if url == "" {
			url = stringAt(entity, "url")
		}
		if url == "" {
			continue
		}
		links = append(links, link{URL: url, DisplayURL: optionalString(entity, "display_url")})
	}
	return links
}

func decodeMentions(entities map[string]any) []userRef {
	mentions := []userRef{}
	for _, raw := range sliceAt(entities, "user_mentions") {
		entity, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		username := optionalString(entity, "screen_name")
		ref := userRef{ID: optionalString(entity, "id_str"), Name: optionalString(entity, "name"), Username: username}
		if username != nil {
			url := "https://x.com/" + *username
			ref.URL = &url
		}
		mentions = append(mentions, ref)
	}
	return mentions
}

func decodeTextEntities(entities map[string]any, key string) []string {
	values := []string{}
	for _, raw := range sliceAt(entities, key) {
		entity, ok := raw.(map[string]any)
		if text := stringAt(entity, "text"); ok && text != "" {
			values = append(values, text)
		}
	}
	return values
}

func normalizeTime(value string) *string {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", value)
	if err != nil {
		return nil
	}
	normalized := parsed.UTC().Format(time.RFC3339)
	return &normalized
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
	object, ok := current.(map[string]any)
	return object, ok
}

func stringAt(root map[string]any, key string) string {
	value, _ := root[key].(string)
	return value
}

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

func boolAt(root map[string]any, key string) bool {
	value, _ := root[key].(bool)
	return value
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

func sliceAt(root map[string]any, key string) []any {
	values, _ := root[key].([]any)
	return values
}

func countAt(root map[string]any, key string) *int64 {
	value, ok := root[key]
	if !ok {
		return nil
	}
	var number int64
	switch typed := value.(type) {
	case float64:
		if typed < 0 || typed != float64(int64(typed)) {
			return nil
		}
		number = int64(typed)
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil || parsed < 0 {
			return nil
		}
		number = parsed
	default:
		return nil
	}
	return &number
}
