package search

type mediaResult struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	URL        *string `json:"url"`
	PreviewURL *string `json:"preview_url"`
	Width      *int64  `json:"width"`
	Height     *int64  `json:"height"`
	AltText    *string `json:"alt_text"`
	DurationMS *int64  `json:"duration_ms"`
}

func decodeMedia(legacy map[string]any) []mediaResult {
	extended, _ := legacy["extended_entities"].(map[string]any)
	media := []mediaResult{}
	for _, raw := range sliceAt(extended, "media") {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		decoded, ok := decodeMediaItem(item)
		if ok {
			media = append(media, decoded)
		}
	}
	return media
}

func decodeMediaItem(item map[string]any) (mediaResult, bool) {
	id := stringAt(item, "id_str")
	if id == "" {
		return mediaResult{}, false
	}
	mediaType, ok := normalizedMediaType(stringAt(item, "type"))
	if !ok {
		return mediaResult{}, false
	}
	original, _ := item["original_info"].(map[string]any)
	result := mediaResult{
		ID:      id,
		Type:    mediaType,
		Width:   countAt(original, "width"),
		Height:  countAt(original, "height"),
		AltText: optionalString(item, "ext_alt_text"),
	}
	thumbnail := optionalString(item, "media_url_https")
	if mediaType == "image" {
		result.URL = thumbnail
		return result, true
	}
	result.PreviewURL = thumbnail
	videoInfo, _ := item["video_info"].(map[string]any)
	result.URL = highestBitrateMP4(videoInfo)
	result.DurationMS = countAt(videoInfo, "duration_millis")
	return result, true
}

func normalizedMediaType(source string) (string, bool) {
	switch source {
	case "photo":
		return "image", true
	case "video":
		return "video", true
	case "animated_gif":
		return "animated_gif", true
	default:
		return "", false
	}
}

func highestBitrateMP4(videoInfo map[string]any) *string {
	var selected *string
	var selectedBitrate *int64
	for _, raw := range sliceAt(videoInfo, "variants") {
		variant, ok := raw.(map[string]any)
		if !ok || stringAt(variant, "content_type") != "video/mp4" {
			continue
		}
		url := optionalString(variant, "url")
		if url == nil {
			continue
		}
		bitrate := countAt(variant, "bitrate")
		if selected == nil || (bitrate != nil && (selectedBitrate == nil || *bitrate > *selectedBitrate)) {
			selected = url
			selectedBitrate = bitrate
		}
	}
	return selected
}
