package search

import "fmt"

const maxQuoteDepth = 128

type communityNote struct {
	ID       string  `json:"id"`
	Text     string  `json:"text"`
	Language *string `json:"language"`
	Sources  []link  `json:"sources"`
	URL      *string `json:"url"`
}

func decodeQuotedPost(tweet map[string]any, ancestors map[string]struct{}, depth int) *postResult {
	quoted, ok := objectAt(tweet, "quoted_status_result", "result")
	if !ok {
		return nil
	}
	post, ok := decodePostResultWithGuard(quoted, ancestors, depth+1)
	if !ok {
		return nil
	}
	return &post
}

func decodeCommunityNote(tweet map[string]any) *communityNote {
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
	return &communityNote{
		ID:       id,
		Text:     text,
		Language: optionalString(note, "language"),
		Sources:  decodeCommunityNoteLinks(subtitle),
		URL:      &url,
	}
}

func decodeCommunityNoteLinks(subtitle map[string]any) []link {
	links := []link{}
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
		displayURL := optionalString(ref, "displayUrl")
		if displayURL == nil {
			displayURL = optionalString(ref, "display_url")
		}
		links = append(links, link{URL: url, DisplayURL: displayURL})
	}
	return links
}
