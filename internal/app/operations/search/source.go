package search

import "strconv"

// These helpers are intentionally source-shape helpers for Search-only user
// and list normalization. Shared post construction has equivalent private
// helpers in models; they are not part of the normalized-model API.
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
func boolAt(root map[string]any, key string) bool { value, _ := root[key].(bool); return value }
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
