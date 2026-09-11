package search

import (
	"strings"

	"github.com/ach968/x-twt-cli/internal/app/models"
)

type userResult struct {
	Type                string        `json:"type"`
	ID                  string        `json:"id"`
	Name                *string       `json:"name"`
	Username            *string       `json:"username"`
	URL                 *string       `json:"url"`
	Bio                 *string       `json:"bio"`
	AvatarURL           *string       `json:"avatar_url"`
	BannerURL           *string       `json:"banner_url"`
	WebsiteURL          *string       `json:"website_url"`
	Verification        *string       `json:"verification"`
	IdentityVerified    *bool         `json:"identity_verified"`
	Protected           *bool         `json:"protected"`
	Affiliation         *affiliation  `json:"affiliation"`
	AutomatedBy         *userRef      `json:"automated_by"`
	ParodyCommentaryFan *string       `json:"parody_commentary_fan"`
	Professional        *professional `json:"professional"`
	Metrics             userMetrics   `json:"metrics"`
}

type affiliation struct {
	Name     *string `json:"name"`
	URL      *string `json:"url"`
	BadgeURL *string `json:"badge_url"`
}

type professional struct {
	Type       *string  `json:"type"`
	Categories []string `json:"categories"`
}

type userMetrics struct {
	Followers *int64 `json:"followers"`
	Following *int64 `json:"following"`
	Posts     *int64 `json:"posts"`
	Media     *int64 `json:"media"`
}

func semanticVerification(user map[string]any) *string {
	return models.SemanticVerification(user)
}

// decodeUserResult normalizes one direct TimelineUser itemContent. Callers
// intentionally remain responsible for traversing the private timeline grammar.
func decodeUserResult(itemContent map[string]any) (userResult, bool) {
	if itemContent["itemType"] != "TimelineUser" {
		return userResult{}, false
	}
	user, ok := objectAt(itemContent, "user_results", "result")
	if !ok {
		return userResult{}, false
	}
	id := stringAt(user, "rest_id")
	if id == "" {
		return userResult{}, false
	}

	core, _ := user["core"].(map[string]any)
	username := optionalString(core, "screen_name")
	result := userResult{
		Type:                "user",
		ID:                  id,
		Name:                cleanOptionalText(optionalString(core, "name")),
		Username:            username,
		Bio:                 cleanOptionalText(optionalNestedString(user, "profile_bio", "description")),
		AvatarURL:           nonEmptyNestedString(user, "avatar", "image_url"),
		BannerURL:           nonEmptyNestedString(user, "banner", "image_url"),
		WebsiteURL:          expandedWebsiteURL(user),
		Verification:        semanticVerification(user),
		IdentityVerified:    nil,
		Protected:           optionalNestedBool(user, "privacy", "protected"),
		Affiliation:         decodeAffiliation(user),
		AutomatedBy:         decodeAutomatedBy(user),
		ParodyCommentaryFan: decodeObservedPCF(user),
		Professional:        decodeProfessional(user),
		Metrics:             decodeUserMetrics(user),
	}
	if username != nil {
		url := "https://x.com/" + *username
		result.URL = &url
	}
	return result, true
}

// decodeObservedPCF deliberately recognizes only source values with reviewed
// representative evidence. Other labels remain unknown until then.
func decodeObservedPCF(user map[string]any) *string {
	var normalized string
	switch stringAt(user, "parody_commentary_fan_label") {
	case "Parody":
		normalized = "parody"
	case "Fan":
		normalized = "fan"
	default:
		return nil
	}
	return &normalized
}

func decodeAffiliation(user map[string]any) *affiliation {
	label, ok := objectAt(user, "affiliates_highlighted_label", "label")
	if !ok || label["userLabelType"] != "BusinessLabel" {
		return nil
	}
	affiliation := &affiliation{
		Name:     cleanOptionalText(optionalString(label, "description")),
		URL:      canonicalXURL(optionalNestedString(label, "url", "url")),
		BadgeURL: nonEmptyNestedString(label, "badge", "url"),
	}
	if affiliation.Name == nil && affiliation.URL == nil && affiliation.BadgeURL == nil {
		return nil
	}
	return affiliation
}

func decodeAutomatedBy(user map[string]any) *userRef {
	label, ok := objectAt(user, "affiliates_highlighted_label", "label")
	if !ok || label["userLabelType"] != "AutomatedLabel" {
		return nil
	}
	entities := sliceAtObject(label, "longDescription", "entities")
	for _, rawEntity := range entities {
		entity, ok := rawEntity.(map[string]any)
		if !ok {
			continue
		}
		operator, ok := objectAt(entity, "ref", "mention_results", "result")
		if ok {
			return decodeUserRef(operator)
		}
	}
	return nil
}

func decodeProfessional(user map[string]any) *professional {
	source, ok := user["professional"].(map[string]any)
	if !ok {
		return nil
	}
	value := strings.ToLower(stringAt(source, "professional_type"))
	var kind *string
	if value != "" {
		kind = &value
	}
	categories := make([]string, 0)
	for _, rawCategory := range sliceAt(source, "category") {
		category, ok := rawCategory.(map[string]any)
		if !ok {
			continue
		}
		if name := cleanReaderText(stringAt(category, "name")); name != "" {
			categories = append(categories, name)
		}
	}
	if kind == nil && len(categories) == 0 {
		return nil
	}
	return &professional{Type: kind, Categories: categories}
}

func decodeUserMetrics(user map[string]any) userMetrics {
	relationships, _ := user["relationship_counts"].(map[string]any)
	tweets, _ := user["tweet_counts"].(map[string]any)
	return userMetrics{
		Followers: countAt(relationships, "followers"),
		Following: countAt(relationships, "following"),
		Posts:     countAt(tweets, "tweets"),
		Media:     countAt(tweets, "media_tweets"),
	}
}

func expandedWebsiteURL(user map[string]any) *string {
	websiteURL := nonEmptyNestedString(user, "website", "url")
	urls := sliceAtObject(user, "profile_bio", "entities", "url", "urls")
	for _, raw := range urls {
		entity, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if websiteURL == nil || stringAt(entity, "url") == *websiteURL {
			if expanded := nonEmptyString(entity, "expanded_url"); expanded != nil {
				return expanded
			}
		}
	}
	if websiteURL != nil && !strings.HasPrefix(*websiteURL, "https://t.co/") && !strings.HasPrefix(*websiteURL, "http://t.co/") {
		return websiteURL
	}
	return nil
}

func nonEmptyString(root map[string]any, key string) *string {
	value := optionalString(root, key)
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func nonEmptyNestedString(root map[string]any, keys ...string) *string {
	value := optionalNestedString(root, keys...)
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func canonicalXURL(value *string) *string {
	if value == nil {
		return nil
	}
	canonical := strings.Replace(*value, "https://twitter.com/", "https://x.com/", 1)
	canonical = strings.Replace(canonical, "http://twitter.com/", "https://x.com/", 1)
	return &canonical
}

func sliceAtObject(root map[string]any, keys ...string) []any {
	object, ok := objectAt(root, keys[:len(keys)-1]...)
	if !ok {
		return nil
	}
	return sliceAt(object, keys[len(keys)-1])
}
