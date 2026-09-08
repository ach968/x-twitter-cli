package search_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ach968/x-twt-cli/internal/app/search"
)

func TestDecodePageNormalizesObservedParodyAndFanSignals(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "testdata", "search-timeline", "people-pcf-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source any
	if err := json.Unmarshal(contents, &source); err != nil {
		t.Fatal(err)
	}

	page, err := search.DecodePage(source, "account labels", search.TabPeople)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":"account labels","tab":"people","results":[{"type":"user","id":"301","name":"Parody Example","username":"parody_example","url":"https://x.com/parody_example","bio":null,"avatar_url":null,"banner_url":null,"website_url":null,"verification":null,"identity_verified":null,"protected":null,"affiliation":null,"automated_by":null,"parody_commentary_fan":"parody","professional":null,"metrics":{"followers":null,"following":null,"posts":null,"media":null}},{"type":"user","id":"302","name":"Fan Example","username":"fan_example","url":"https://x.com/fan_example","bio":null,"avatar_url":null,"banner_url":null,"website_url":null,"verification":null,"identity_verified":null,"protected":null,"affiliation":null,"automated_by":null,"parody_commentary_fan":"fan","professional":null,"metrics":{"followers":null,"following":null,"posts":null,"media":null}},{"type":"user","id":"303","name":"Unsupported Example","username":"unsupported_example","url":"https://x.com/unsupported_example","bio":null,"avatar_url":null,"banner_url":null,"website_url":null,"verification":null,"identity_verified":null,"protected":null,"affiliation":null,"automated_by":null,"parody_commentary_fan":null,"professional":null,"metrics":{"followers":null,"following":null,"posts":null,"media":null}}],"next_cursor":null,"warnings":[]}`
	if string(got) != want {
		t.Fatalf("page:\n got %s\nwant %s", got, want)
	}
}
