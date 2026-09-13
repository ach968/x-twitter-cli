package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
)

type BrowserTransport struct {
	ProfilePath string
	Origin      string
	Timeout     time.Duration
	Direct      app.XTransport
}

func NewBrowserTransport(profilePath, origin string, timeout time.Duration, direct ...app.XTransport) *BrowserTransport {
	if origin == "" {
		origin = "https://x.com"
	}
	directTransport := app.XTransport(httpclient.NewTransport(nil))
	if len(direct) > 0 {
		directTransport = direct[0]
	}
	return &BrowserTransport{ProfilePath: profilePath, Origin: origin, Timeout: timeout, Direct: directTransport}
}

func searchQuery(request app.PreparedRequest) (string, error) {
	parsed, err := url.Parse(request.URL)
	if err != nil {
		return "", err
	}
	variables := parsed.Query().Get("variables")
	if variables == "" {
		return "", errors.New("SearchTimeline request has no variables")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(variables), &decoded); err != nil {
		return "", err
	}
	query, ok := decoded["rawQuery"].(string)
	if !ok {
		return "", errors.New("SearchTimeline request has no rawQuery")
	}
	return query, nil
}

func isMatchingSearchResponse(responseURL, query string) bool {
	parsed, err := url.Parse(responseURL)
	if err != nil || !strings.HasSuffix(parsed.Path, "/SearchTimeline") {
		return false
	}
	variables := parsed.Query().Get("variables")
	if variables == "" {
		return false
	}
	var decoded map[string]any
	return json.Unmarshal([]byte(variables), &decoded) == nil && decoded["rawQuery"] == query
}

func (transport *BrowserTransport) Execute(ctx context.Context, operation app.OperationName, request app.PreparedRequest) (app.UpstreamResponse, error) {
	if operation == app.HomeTimeline {
		return transport.Direct.Execute(ctx, operation, request)
	}
	if operation != app.SearchTimeline {
		return app.UpstreamResponse{}, fmt.Errorf("unsupported browser transport operation: %s", operation)
	}
	query, err := searchQuery(request)
	if err != nil {
		return app.UpstreamResponse{}, err
	}
	origin, err := url.Parse(transport.Origin)
	if err != nil {
		return app.UpstreamResponse{}, err
	}
	navigation := origin.ResolveReference(&url.URL{Path: "/search"})
	values := navigation.Query()
	values.Set("q", query)
	values.Set("src", "typed_query")
	navigation.RawQuery = values.Encode()
	return ExecuteBrowserOperation(BrowserOperationExecution{
		ProfilePath: transport.ProfilePath, NavigationURL: navigation.String(), Timeout: transport.Timeout,
		MatchesResponse: func(responseURL string) bool { return isMatchingSearchResponse(responseURL, query) },
	})
}
