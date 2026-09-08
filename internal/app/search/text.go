package search

import (
	"html"
	"strings"
	"unicode/utf8"
)

// cleanReaderText removes presentation whitespace from user-authored text and
// replaces straight double quotes with typographic quotes. The latter keeps
// quotes readable in the one-line JSON interface without exposing JSON escape
// sequences as apparent content to callers that inspect the output directly.
func cleanReaderText(value string) string {
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

func cleanOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := cleanReaderText(*value)
	return &cleaned
}
