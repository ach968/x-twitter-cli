package view_test

import (
	"bytes"
	"context"
	"encoding/json"
	app "github.com/ach968/x-twitter-cli/internal/app"
	command "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/view"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	view "github.com/ach968/x-twitter-cli/internal/app/operations/view"
	"net/url"
	"strings"
	"testing"
)

type transport struct {
	calls   int
	request app.PreparedRequest
	body    string
	status  int
}

func TestViewRejectsMalformedInputBeforeContactingX(t *testing.T) {
	cases := [][]string{{}, {"100", "200"}, {"0"}, {"01"}, {"-1"}, {"+1"}, {"1.0"}, {"18446744073709551616"}, {" "}, {"abc"}, {"--user", "x", "--id", "100"},
		{"https://evil.example/x/status/100"}, {"https://x.com.evil.example/x/status/100"}, {"https://user@x.com/x/status/100"}, {"https://x.com:443/x/status/100"}, {"ftp://x.com/x/status/100"},
		{"https://x.com/x/status/"}, {"https://x.com/x/status/100/extra"}, {"https://x.com/x%2fstatus%2f100"}, {"https://x.com/x/status/100//"},
		{"100", "--cursor"}, {"100", "--cursor="}, {"100", "--cursor", " "}, {"100", "--cursor=a", "--cursor=b"}, {"100", "--unknown"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			upstream := &transport{body: initial, status: 200}
			var stdout, stderr bytes.Buffer
			code := command.New(requester(upstream)).Run(context.Background(), args, nil, &stdout, &stderr)
			if code == 0 || upstream.calls != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "INVALID_ARGUMENT") {
				t.Fatalf("code=%d calls=%d stderr=%s", code, upstream.calls, &stderr)
			}
		})
	}
}

func TestViewAcceptsSupportedLinksAndPreservesOpaqueCursor(t *testing.T) {
	cases := []string{"100", "18446744073709551615", "http://x.com/x/status/100", "https://www.x.com/x/status/100/", "https://mobile.x.com/x/status/100/photo/2", "https://twitter.com/i/web/status/100?s=20#fragment", "https://www.twitter.com/i/status/100", "https://mobile.twitter.com/x/status/100/video/1"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			upstream := &transport{body: `{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineAddEntries","entries":[]}]}}}`, status: 200}
			var stdout, stderr bytes.Buffer
			token := " opaque+/= token "
			code := command.New(requester(upstream)).Run(context.Background(), []string{"--cursor=" + token, input}, nil, &stdout, &stderr)
			if code != 0 || upstream.calls != 1 || stderr.Len() != 0 {
				t.Fatalf("code=%d calls=%d stderr=%s", code, upstream.calls, &stderr)
			}
			parsed, _ := url.Parse(upstream.request.URL)
			var variables map[string]any
			json.Unmarshal([]byte(parsed.Query().Get("variables")), &variables)
			wantID := "100"
			if input == "18446744073709551615" {
				wantID = input
			}
			if variables["focalTweetId"] != wantID || variables["cursor"] != token {
				t.Fatalf("variables=%#v", variables)
			}
		})
	}
}

func (s *transport) Execute(_ context.Context, _ app.OperationName, request app.PreparedRequest) (app.UpstreamResponse, error) {
	s.calls++
	s.request = request
	return app.UpstreamResponse{Status: s.status, Body: s.body}, nil
}
func requester(s *transport) *view.Operation {
	props := app.ContractProperties{Operations: map[app.OperationName]app.OperationContract{
		app.TweetDetail: {Host: "x.com", Path: "/i/api/graphql/test/TweetDetail", Method: "GET", Encoding: "query", Variables: map[string]any{"rankingMode": "Relevance", "focalTweetId": "999"}},
	}}
	return view.New(view.NewDirectSource(httpclient.New(props, app.AuthenticationState{}, s, nil)))
}

const initial = `{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineAddEntries","entries":[{"entryId":"tweet-100","content":{"entryType":"TimelineTimelineItem","itemContent":{"itemType":"TimelineTweet","tweet_results":{"result":{"rest_id":"100","legacy":{"full_text":"A synthetic post"}}}}}}]}]}}}`

func TestViewLinkReturnsCanonicalPostThroughOneDirectRequest(t *testing.T) {
	upstream := &transport{body: initial, status: 200}
	var stdout, stderr bytes.Buffer
	code := command.New(requester(upstream)).Run(context.Background(), []string{"https://twitter.com/OldHandle/status/100?s=20"}, nil, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || upstream.calls != 1 {
		t.Fatalf("code=%d calls=%d stderr=%s", code, upstream.calls, &stderr)
	}
	var page view.Page
	if err := json.Unmarshal(stdout.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.PostID != "100" || page.Post == nil || page.Post.Text != "A synthetic post" {
		t.Fatal("wrong canonical output")
	}
	parsed, _ := url.Parse(upstream.request.URL)
	var variables map[string]any
	json.Unmarshal([]byte(parsed.Query().Get("variables")), &variables)
	if variables["focalTweetId"] != "100" || variables["rankingMode"] != "Relevance" {
		t.Fatalf("wrong variables: %#v", variables)
	}
}

func TestViewFailuresUseStderrWithoutLeakingSource(t *testing.T) {
	for _, tc := range []struct {
		status     int
		body, code string
	}{
		{401, `secret-auth-body`, "AUTHENTICATION_FAILED"},
		{429, `secret-rate-body`, "RATE_LIMITED"},
		{200, `{"errors":[{"message":"secret-graphql-body"}],"data":{"threaded_conversation_with_injections_v2":{"instructions":[]}}}`, "UPSTREAM_REJECTED"},
		{200, `{"errors":[{"message":"PersistedQueryNotFound"}]}`, "CONTRACT_FAILED"},
	} {
		upstream := &transport{status: tc.status, body: tc.body}
		var stdout, stderr bytes.Buffer
		code := command.New(requester(upstream)).Run(context.Background(), []string{"100"}, nil, &stdout, &stderr)
		if code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.code) || strings.Contains(stderr.String(), "secret-") {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
		}
	}
}
