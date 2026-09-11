package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDecodeListResultNormalizesVerticalModuleItems(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("testdata", "lists-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	var module map[string]any
	if err := json.Unmarshal(contents, &module); err != nil {
		t.Fatal(err)
	}

	source := map[string]any{"data": map[string]any{"search_by_raw_query": map[string]any{"search_timeline": map[string]any{"timeline": map[string]any{"instructions": []any{map[string]any{"type": "TimelineAddEntries", "entries": []any{module}}}}}}}}
	page, err := DecodePage(source, "golang", TabLists)
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "lists-golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results any
	if err := json.Unmarshal(want, &results); err != nil {
		t.Fatal(err)
	}
	expected := Page{Query: "golang", Tab: TabLists, Results: results.([]any), Warnings: []Warning{}}
	actual, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatal(err)
	}
	expectedJSON, _ := json.Marshal(expected)
	var expectedValue any
	if err := json.Unmarshal(expectedJSON, &expectedValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("list page =\n%s\nwant\n%s", actual, expectedJSON)
	}
}
