package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ach968/x-twitter-cli3/internal/app/operations/search"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestNormalizedSearchPagesConformToSchema(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	for _, resource := range []struct {
		uri  string
		path string
	}{
		{uri: "urn:x-twitter-cli3:schema:shared-post", path: filepath.Join("..", "docs", "shared-post.schema.json")},
		{uri: "urn:x-twitter-cli3:schema:search-timeline", path: filepath.Join("..", "docs", "search-timeline.schema.json")},
	} {
		schemaContents, err := os.ReadFile(resource.path)
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if err := json.Unmarshal(schemaContents, &document); err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource(resource.uri, document); err != nil {
			t.Fatal(err)
		}
	}
	contract, err := compiler.Compile("urn:x-twitter-cli3:schema:search-timeline")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		query      string
		tab        search.Tab
		listModule bool
	}{
		{name: "empty", path: filepath.Join("testdata", "search-timeline", "empty-initial.json"), query: "quiet query", tab: search.TabTop},
		{name: "posts", path: filepath.Join("testdata", "search-timeline", "populated-initial.json"), query: "from:example", tab: search.TabTop},
		{name: "later page", path: filepath.Join("testdata", "search-timeline", "populated-page-2.json"), query: "page two", tab: search.TabLatest},
		{name: "people", path: filepath.Join("testdata", "search-timeline", "people-results.json"), query: "example people", tab: search.TabPeople},
		{name: "parody fan", path: filepath.Join("testdata", "search-timeline", "people-pcf-results.json"), query: "account labels", tab: search.TabPeople},
		{name: "lists", path: filepath.Join("..", "internal", "app", "operations", "search", "testdata", "lists-source.json"), query: "golang", tab: search.TabLists, listModule: true},
		{name: "media", path: filepath.Join("..", "internal", "app", "operations", "search", "testdata", "media-source.json"), query: "cats", tab: search.TabMedia},
		{name: "quotes and notes", path: filepath.Join("..", "internal", "app", "operations", "search", "testdata", "quotes-notes-source.json"), query: "context", tab: search.TabTop},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contents, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			var source any
			if err := json.Unmarshal(contents, &source); err != nil {
				t.Fatal(err)
			}
			if test.listModule {
				source = searchSourceWithListModule(source)
			}
			page, err := search.DecodePage(source, test.query, test.tab)
			if err != nil {
				t.Fatal(err)
			}
			normalized, err := json.Marshal(page)
			if err != nil {
				t.Fatal(err)
			}
			var value any
			if err := json.Unmarshal(normalized, &value); err != nil {
				t.Fatal(err)
			}
			if err := contract.Validate(value); err != nil {
				t.Fatalf("normalized page does not conform to schema: %v", err)
			}
		})
	}
}

func searchSourceWithListModule(module any) any {
	return map[string]any{
		"data": map[string]any{
			"search_by_raw_query": map[string]any{
				"search_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{map[string]any{
							"type":    "TimelineAddEntries",
							"entries": []any{module},
						}},
					},
				},
			},
		},
	}
}
