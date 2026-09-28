package httpclient

import (
	"net/url"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

func TestTransactionSourceClientScopesAuthenticationCookies(t *testing.T) {
	auth := app.AuthenticationState{Cookies: []app.AuthenticationCookie{
		{Name: "auth_token", Value: "session-token"},
		{Name: "ct0", Value: "csrf-token"},
		{Name: "unrelated", Value: "other-cookie"},
	}}
	client, err := transactionSourceClient(auth)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		url       string
		wantNames []string
	}{
		{"https://x.com/home", []string{"auth_token", "ct0"}},
		{"https://abs.twimg.com/responsive-web/client-web/ondemand.s.test.js", nil},
		{"https://example.com/", nil},
	} {
		parsed, err := url.Parse(test.url)
		if err != nil {
			t.Fatal(err)
		}
		cookies := client.Jar.Cookies(parsed)
		if len(cookies) != len(test.wantNames) {
			t.Fatalf("%s: cookie count=%d, want %d", test.url, len(cookies), len(test.wantNames))
		}
		for index, cookie := range cookies {
			if cookie.Name != test.wantNames[index] {
				t.Fatalf("%s: cookie %d=%s, want %s", test.url, index, cookie.Name, test.wantNames[index])
			}
		}
	}
}
