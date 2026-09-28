package httpclient

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twitter-cli/internal/app"
)

// NewTransactionIDGenerator reads X's current frontend using the account's
// stored login cookies. The cookies are scoped to x.com and are not sent to
// the frontend asset host.
func NewTransactionIDGenerator(ctx context.Context, authentication app.AuthenticationState) (app.TransactionIDGenerator, error) {
	client, err := transactionSourceClient(authentication)
	if err != nil {
		return nil, err
	}
	return transaction.New(ctx, client)
}

func transactionSourceClient(authentication app.AuthenticationState) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cookies := make([]*http.Cookie, 0, 2)
	for _, cookie := range authentication.Cookies {
		if cookie.Name == "auth_token" || cookie.Name == "ct0" {
			cookies = append(cookies, &http.Cookie{Name: cookie.Name, Value: cookie.Value})
		}
	}
	jar.SetCookies(&url.URL{Scheme: "https", Host: "x.com"}, cookies)
	return &http.Client{Jar: jar, Timeout: 20 * time.Second}, nil
}
