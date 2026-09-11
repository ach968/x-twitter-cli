package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestBookmarksGoldenPagesConformToSchema(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	for _, resource := range []struct {
		uri  string
		path string
	}{
		{uri: "urn:x-twt-cli:schema:shared-post", path: filepath.Join("..", "docs", "shared-post.schema.json")},
		{uri: "urn:x-twt-cli:schema:bookmarks-page", path: filepath.Join("..", "docs", "bookmarks-page.schema.json")},
	} {
		contents, err := os.ReadFile(resource.path)
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if err := json.Unmarshal(contents, &document); err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource(resource.uri, document); err != nil {
			t.Fatal(err)
		}
	}
	contract, err := compiler.Compile("urn:x-twt-cli:schema:bookmarks-page")
	if err != nil {
		t.Fatal(err)
	}

	fixtures, err := filepath.Glob(filepath.Join("..", "internal", "app", "operations", "bookmarks", "testdata", "*-golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no bookmark golden fixtures found")
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			contents, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			var page any
			if err := json.Unmarshal(contents, &page); err != nil {
				t.Fatal(err)
			}
			if err := contract.Validate(page); err != nil {
				t.Fatalf("bookmark page does not conform to schema: %v", err)
			}
		})
	}
}
