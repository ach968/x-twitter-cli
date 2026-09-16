package test

import (
	"context"
	"encoding/json"
	view "github.com/ach968/x-twitter-cli/internal/app/operations/view"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"os"
	"testing"
)

type viewFixture struct{ raw string }

func (f viewFixture) Fetch(context.Context, view.Request) (any, error) {
	var payload any
	err := json.Unmarshal([]byte(f.raw), &payload)
	return payload, err
}

func TestViewPagesConformToPublishedSchema(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	for uri, path := range map[string]string{"urn:x-twitter-cli:schema:shared-post": "../docs/shared-post.schema.json", "urn:x-twitter-cli:schema:view-page": "../docs/view-page.schema.json"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource(uri, doc); err != nil {
			t.Fatal(err)
		}
	}
	schema, err := compiler.Compile("urn:x-twitter-cli:schema:view-page")
	if err != nil {
		t.Fatal(err)
	}
	token := "next"
	for _, tc := range []struct {
		raw    string
		cursor *string
	}{
		{`{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineAddEntries","entries":[{"entryId":"tweet-100","content":{"entryType":"TimelineTimelineItem","itemContent":{"itemType":"TimelineTweet","tweet_results":{"result":{"rest_id":"100","legacy":{"full_text":"Synthetic","in_reply_to_status_id_str":"50"}}}}}}]}]}}}`, nil},
		{`{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineAddEntries","entries":[]}]}}}`, &token},
	} {
		page, err := view.New(viewFixture{tc.raw}).Execute(context.Background(), view.Request{PostID: "100", Cursor: tc.cursor})
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		json.Unmarshal(data, &value)
		if err := schema.Validate(value); err != nil {
			t.Fatal(err)
		}
	}
}
