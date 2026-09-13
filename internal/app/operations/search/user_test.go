package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ach968/x-twitter-cli3/internal/app/operations/search"
)

func TestDecodePageNormalizesPeopleResults(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "test", "testdata", "search-timeline", "people-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}

	page, err := search.DecodePage(source, "example people", search.TabPeople)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":"example people","tab":"people","results":[{"type":"user","id":"100","name":"Example Person","username":"example","url":"https://x.com/example","bio":"Example biography","avatar_url":"https://example.test/avatar.jpg","banner_url":"https://example.test/banner.jpg","website_url":"https://example.test/","verification":"premium","identity_verified":null,"protected":false,"affiliation":{"name":"Example Organization","url":"https://x.com/example_org","badge_url":"https://example.test/badge.png"},"automated_by":null,"parody_commentary_fan":null,"professional":{"type":"business","categories":["Science \u0026 Technology","Software Company"]},"metrics":{"followers":0,"following":12,"posts":34,"media":0}},{"type":"user","id":"101","name":"Automated Account","username":"automation","url":"https://x.com/automation","bio":null,"avatar_url":"https://example.test/automation.jpg","banner_url":null,"website_url":null,"verification":null,"identity_verified":null,"protected":true,"affiliation":null,"automated_by":{"id":"200","name":null,"username":"operator","url":"https://x.com/operator","avatar_url":null,"verification":null,"protected":null},"parody_commentary_fan":null,"professional":null,"metrics":{"followers":1,"following":2,"posts":3,"media":4}},{"type":"user","id":"102","name":"Partial Account","username":"partial","url":"https://x.com/partial","bio":null,"avatar_url":null,"banner_url":null,"website_url":null,"verification":null,"identity_verified":null,"protected":false,"affiliation":null,"automated_by":null,"parody_commentary_fan":null,"professional":null,"metrics":{"followers":null,"following":null,"posts":null,"media":null}}],"next_cursor":null,"warnings":[]}`
	if string(got) != want {
		t.Fatalf("page:\n got %s\nwant %s", got, want)
	}
}
