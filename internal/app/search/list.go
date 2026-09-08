package search

// listResult is the stable representation of a List returned in a search
// page. Fields that the source omits retain the contract's explicit null.
type listResult struct {
	Type        string      `json:"type"`
	ID          string      `json:"id"`
	Name        *string     `json:"name"`
	Description *string     `json:"description"`
	URL         *string     `json:"url"`
	Owner       *userRef    `json:"owner"`
	Private     *bool       `json:"private"`
	BannerURL   *string     `json:"banner_url"`
	Metrics     listMetrics `json:"metrics"`
}

type listMetrics struct {
	Members     *int64 `json:"members"`
	Subscribers *int64 `json:"subscribers"`
}

// decodeListResult translates the item content carried by a TimelineList into
// the flat list result contract. Callers are responsible for unwrapping the
// surrounding timeline module and preserving its item order.
func decodeListResult(itemContent map[string]any) (listResult, bool) {
	if itemContent["itemType"] != "TimelineTwitterList" {
		return listResult{}, false
	}
	rawList, ok := itemContent["list"].(map[string]any)
	if !ok {
		return listResult{}, false
	}
	id := stringAt(rawList, "id_str")
	if id == "" {
		id = stringAt(rawList, "rest_id")
	}
	if id == "" {
		return listResult{}, false
	}

	result := listResult{
		Type:        "list",
		ID:          id,
		Name:        optionalString(rawList, "name"),
		Description: optionalString(rawList, "description"),
		Owner:       decodeListOwner(rawList),
		Private:     decodeListPrivacy(rawList),
		BannerURL:   decodeListBanner(rawList),
		Metrics: listMetrics{
			Members:     countAt(rawList, "member_count"),
			Subscribers: countAt(rawList, "subscriber_count"),
		},
	}
	url := "https://x.com/i/lists/" + id
	result.URL = &url
	return result, true
}

func decodeListBanner(rawList map[string]any) *string {
	if value := optionalNestedString(rawList, "default_banner_media", "media_info", "original_img_url"); value != nil {
		return value
	}
	return optionalNestedString(rawList, "custom_banner_media", "media_info", "original_img_url")
}

func decodeListOwner(rawList map[string]any) *userRef {
	owner, ok := objectAt(rawList, "user_results", "result")
	if !ok {
		return nil
	}
	return decodeUserRef(owner)
}

// SearchTimeline evidence presently establishes only the public mapping. The
// contract reserves private-list support for a representative source shape.
func decodeListPrivacy(rawList map[string]any) *bool {
	if stringAt(rawList, "mode") != "Public" {
		return nil
	}
	public := false
	return &public
}
