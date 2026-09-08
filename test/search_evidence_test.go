package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	app "github.com/ach968/x-twt-cli/internal/app"
)

const evidenceVersion = 1

var (
	evidenceScenarioPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	evidenceCookieSecretPattern  = regexp.MustCompile(`(?i)(auth_token|ct0|csrf_token)=([^;\s&]+)`)
	evidenceAuthorizationPattern = regexp.MustCompile(`(?i)(authorization:\s*)([^\r\n]+)`)
	evidenceBearerPattern        = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/-]+=*`)
	evidenceSearchProducts       = map[string]struct{}{"Top": {}, "Latest": {}, "People": {}, "Media": {}, "Lists": {}}
)

type searchTimelineSample struct {
	Scenario       string
	Query          string
	Product        string
	Page           int
	CursorProvided bool
	Payload        any
}

type searchTimelineEvidence struct {
	Version        int               `json:"version"`
	ReviewStatus   string            `json:"reviewStatus"`
	CapturedAt     string            `json:"capturedAt"`
	Operation      app.OperationName `json:"operation"`
	Scenario       string            `json:"scenario"`
	Query          string            `json:"query"`
	Product        string            `json:"product"`
	Page           int               `json:"page"`
	CursorProvided bool              `json:"cursorProvided"`
	SourcePayload  any               `json:"sourcePayload"`
}

func saveSearchTimelineEvidence(directory string, capturedAt time.Time, sample searchTimelineSample) (string, error) {
	if directory == "" {
		return "", errors.New("evidence directory is required")
	}
	if !evidenceScenarioPattern.MatchString(sample.Scenario) {
		return "", errors.New("scenario must contain only lowercase letters, numbers, and hyphens")
	}
	if strings.TrimSpace(sample.Query) == "" {
		return "", errors.New("search query is required")
	}
	if _, ok := evidenceSearchProducts[sample.Product]; !ok {
		return "", errors.New("search product must be Top, Latest, People, Media, or Lists")
	}
	if sample.Page < 1 {
		return "", errors.New("page must be at least 1")
	}
	if sample.Payload == nil {
		return "", errors.New("source payload is required")
	}

	capturedAt = capturedAt.UTC()
	evidence := searchTimelineEvidence{
		Version:        evidenceVersion,
		ReviewStatus:   "pending",
		CapturedAt:     capturedAt.Format(time.RFC3339Nano),
		Operation:      app.SearchTimeline,
		Scenario:       sample.Scenario,
		Query:          redactEvidenceString(sample.Query),
		Product:        sample.Product,
		Page:           sample.Page,
		CursorProvided: sample.CursorProvided,
		SourcePayload:  redactEvidenceValue(sample.Payload),
	}
	contents, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode search evidence: %w", err)
	}

	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create evidence directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", fmt.Errorf("secure evidence directory: %w", err)
	}

	filename := fmt.Sprintf("%s-search-timeline-%s.json", capturedAt.Format("20060102T150405.000000000Z"), sample.Scenario)
	destination := filepath.Join(directory, filename)
	temporary, err := os.CreateTemp(directory, ".search-evidence-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary evidence file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return "", fmt.Errorf("secure temporary evidence file: %w", err)
	}
	if _, err := temporary.Write(append(contents, '\n')); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write temporary evidence file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary evidence file: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return "", fmt.Errorf("activate evidence file: %w", err)
	}
	return destination, nil
}

func redactEvidenceValue(value any) any {
	switch typed := value.(type) {
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = redactEvidenceValue(child)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			normalized := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(key))
			switch normalized {
			case "authorization", "cookie", "authtoken", "ct0", "csrftoken":
				result[key] = "[REDACTED]"
			default:
				result[key] = redactEvidenceValue(child)
			}
		}
		return result
	case string:
		return redactEvidenceString(typed)
	default:
		return value
	}
}

func redactEvidenceString(value string) string {
	value = evidenceCookieSecretPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = evidenceAuthorizationPattern.ReplaceAllString(value, "$1[REDACTED]")
	return evidenceBearerPattern.ReplaceAllString(value, "$1[REDACTED]")
}

func TestSaveSearchTimelineEvidence(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "evidence")
	payload := map[string]any{
		"data": map[string]any{
			"authorization": "Bearer payload-secret",
			"csrfToken":     "camel-case-secret",
			"text":          "public post",
			"url":           "https://x.com/?ct0=csrf-secret&safe=yes",
		},
	}

	path, err := saveSearchTimelineEvidence(directory, time.Date(2026, time.September, 4, 12, 34, 56, 123, time.UTC), searchTimelineSample{
		Scenario: "populated-initial",
		Query:    "golang",
		Product:  "Top",
		Page:     1,
		Payload:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "20260904T123456.000000123Z-search-timeline-populated-initial.json" {
		t.Fatalf("unexpected evidence path: %s", path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	for _, secret := range []string{"payload-secret", "camel-case-secret", "csrf-secret"} {
		if strings.Contains(text, secret) {
			t.Fatalf("evidence contains secret %q: %s", secret, text)
		}
	}
	for _, expected := range []string{`"reviewStatus": "pending"`, `"operation": "SearchTimeline"`, `"query": "golang"`, `"product": "Top"`, `"text": "public post"`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("evidence does not contain %s: %s", expected, text)
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("evidence is not valid JSON: %v", err)
	}
	if payload["data"].(map[string]any)["authorization"] != "Bearer payload-secret" {
		t.Fatal("recording mutated the source payload")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("evidence permissions = %o, want 600", info.Mode().Perm())
	}
	directoryInfo, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("evidence directory permissions = %o, want 700", directoryInfo.Mode().Perm())
	}
}

func TestSaveSearchTimelineEvidenceRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name   string
		sample searchTimelineSample
	}{
		{name: "scenario", sample: searchTimelineSample{Scenario: "Has Spaces", Query: "go", Product: "Top", Page: 1, Payload: map[string]any{}}},
		{name: "query", sample: searchTimelineSample{Scenario: "initial", Product: "Top", Page: 1, Payload: map[string]any{}}},
		{name: "product", sample: searchTimelineSample{Scenario: "initial", Query: "go", Product: "Photos", Page: 1, Payload: map[string]any{}}},
		{name: "page", sample: searchTimelineSample{Scenario: "initial", Query: "go", Product: "Top", Payload: map[string]any{}}},
		{name: "payload", sample: searchTimelineSample{Scenario: "initial", Query: "go", Product: "Top", Page: 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := saveSearchTimelineEvidence(t.TempDir(), time.Now(), test.sample); err == nil {
				t.Fatal("expected invalid sample to be rejected")
			}
		})
	}
}
