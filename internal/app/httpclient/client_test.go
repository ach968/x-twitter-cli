package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	app "github.com/ach968/x-twitter-cli3/internal/app"
)

func testContracts() app.ContractProperties {
	return app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{
		app.HomeTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/home-id/HomeTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20), "requestContext": "launch"}, Features: map[string]any{"timeline_enabled": true}, FieldToggles: map[string]any{"withArticleRichContentState": false},
		},
		app.SearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/search-id/SearchTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20), "querySource": "typed_query"}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
	}}
}

type transportFunc struct {
	home   func(app.PreparedRequest) (app.UpstreamResponse, error)
	search func(app.PreparedRequest) (app.UpstreamResponse, error)
}

type transactionIDFunc func(method, path string) (string, error)

func (function transactionIDFunc) Generate(method, path string) (string, error) {
	return function(method, path)
}

func (transport transportFunc) Execute(_ context.Context, operation app.OperationName, request app.PreparedRequest) (app.UpstreamResponse, error) {
	switch operation {
	case app.HomeTimeline:
		return transport.home(request)
	case app.SearchTimeline:
		return transport.search(request)
	default:
		return app.UpstreamResponse{}, errors.New("unsupported operation")
	}
}

func successfulTransport() transportFunc {
	success := app.UpstreamResponse{Status: 200, Body: `{"data":{"ok":true}}`}
	return transportFunc{
		home:   func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		search: func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
	}
}

func TestTransportPreservesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" || request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected request: %s %#v", request.Method, request.Header)
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(202)
		_, _ = io.WriteString(response, `{"data":{"ok":true}}`)
	}))
	defer server.Close()
	result, err := NewTransport(server.Client()).Execute(context.Background(), app.HomeTimeline, app.PreparedRequest{
		Method: "POST", URL: server.URL, Headers: map[string]string{"authorization": "Bearer token"}, Body: "body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != 202 || result.ContentType == nil || *result.ContentType != "application/json; charset=utf-8" || result.Body != `{"data":{"ok":true}}` {
		t.Fatalf("unexpected response: %#v", result)
	}
}

func TestTransportSendsPreparedSearchRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("x-client-transaction-id") != "prepared-id" {
			t.Errorf("transaction ID = %q", request.Header.Get("x-client-transaction-id"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"data":{"ok":true}}`)
	}))
	defer server.Close()
	result, err := NewTransport(server.Client()).Execute(context.Background(), app.SearchTimeline, app.PreparedRequest{
		Method: http.MethodGet, URL: server.URL + "/i/api/graphql/id/SearchTimeline", Headers: map[string]string{"authorization": "Bearer token", "x-client-transaction-id": "prepared-id"},
	})
	if err != nil || result.Status != 200 {
		t.Fatalf("unexpected response: %#v %v", result, err)
	}
}

func TestClientPreparesQueryOperations(t *testing.T) {
	var homeRequest, searchRequest app.PreparedRequest
	transport := transportFunc{
		home: func(request app.PreparedRequest) (app.UpstreamResponse, error) {
			homeRequest = request
			return app.UpstreamResponse{Status: 200, Body: `{"data":{"home":true}}`}, nil
		},
		search: func(request app.PreparedRequest) (app.UpstreamResponse, error) {
			searchRequest = request
			return app.UpstreamResponse{Status: 200, Body: `{"data":{"search":true}}`}, nil
		},
	}
	transactionIDs := transactionIDFunc(func(method, path string) (string, error) {
		if method != http.MethodGet || (path != "/i/api/graphql/search-id/SearchTimeline" && path != "/i/api/graphql/home-id/HomeTimeline") {
			t.Fatalf("generated for %s %s", method, path)
		}
		return "generated-id", nil
	})
	requestClient := New(testContracts(), app.AuthenticationState{
		Cookies: []app.AuthenticationCookie{{Name: "auth_token", Value: "session-token"}, {Name: "ct0", Value: "csrf-token"}}, Authorization: "Bearer public-web-token",
	}, transport, transactionIDs)
	home, err := requestClient.Execute(context.Background(), app.HomeTimeline, map[string]any{"cursor": "next-page"})
	if err != nil || !home.OK {
		t.Fatalf("home failed: %#v %v", home, err)
	}
	parsed, _ := url.Parse(homeRequest.URL)
	var variables map[string]any
	_ = json.Unmarshal([]byte(parsed.Query().Get("variables")), &variables)
	if variables["cursor"] != "next-page" || variables["requestContext"] != "launch" {
		t.Fatalf("unexpected variables: %#v", variables)
	}
	wantHeaders := map[string]string{
		"authorization": "Bearer public-web-token", "cookie": "auth_token=session-token; ct0=csrf-token", "x-csrf-token": "csrf-token",
		"x-twitter-active-user": "yes", "x-twitter-auth-type": "OAuth2Session", "x-client-transaction-id": "generated-id",
	}
	if !reflect.DeepEqual(homeRequest.Headers, wantHeaders) {
		t.Fatalf("headers = %#v", homeRequest.Headers)
	}
	search, err := requestClient.Execute(context.Background(), app.SearchTimeline, map[string]any{"rawQuery": "golang", "cursor": "page-2"})
	if err != nil || !search.OK {
		t.Fatalf("search failed: %#v %v", search, err)
	}
	parsed, _ = url.Parse(searchRequest.URL)
	_ = json.Unmarshal([]byte(parsed.Query().Get("variables")), &variables)
	if variables["rawQuery"] != "golang" || variables["cursor"] != "page-2" {
		t.Fatalf("unexpected search variables: %#v", variables)
	}
	if searchRequest.Headers["x-client-transaction-id"] != "generated-id" {
		t.Fatalf("transaction ID = %q", searchRequest.Headers["x-client-transaction-id"])
	}
}

func TestClientExecutesCapturedOperationWithSemanticOverrides(t *testing.T) {
	var prepared app.PreparedRequest
	transport := successfulTransport()
	transport.search = func(request app.PreparedRequest) (app.UpstreamResponse, error) {
		prepared = request
		return app.UpstreamResponse{Status: 200, Body: `{"data":{"search":true}}`}, nil
	}
	client := New(testContracts(), app.AuthenticationState{}, transport, transactionIDFunc(func(string, string) (string, error) { return "generated-id", nil }))

	result, err := client.Execute(context.Background(), app.SearchTimeline, map[string]any{"rawQuery": "golang", "product": "Latest"})
	if err != nil || !result.OK {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	parsed, _ := url.Parse(prepared.URL)
	var variables map[string]any
	_ = json.Unmarshal([]byte(parsed.Query().Get("variables")), &variables)
	if variables["rawQuery"] != "golang" || variables["product"] != "Latest" || prepared.Headers["x-client-transaction-id"] != "generated-id" {
		t.Fatalf("prepared request = %#v", prepared)
	}
}

func TestClientPreparesJSONPostContract(t *testing.T) {
	properties := testContracts()
	home := properties.Operations[app.HomeTimeline]
	home.Method = "POST"
	home.Encoding = "json"
	home.Variables = map[string]any{"count": float64(20), "includePromotedContent": true}
	home.Features = map[string]any{"timeline_enabled": true}
	home.FieldToggles = map[string]any{}
	home.Body = map[string]any{"queryId": "current-id", "variables": home.Variables, "features": home.Features}
	properties.Operations[app.HomeTimeline] = home
	var prepared app.PreparedRequest
	transport := successfulTransport()
	transport.home = func(request app.PreparedRequest) (app.UpstreamResponse, error) {
		prepared = request
		return app.UpstreamResponse{Status: 200, Body: `{"data":{"home":true}}`}, nil
	}
	result, err := New(properties, app.AuthenticationState{}, transport, nil).Execute(context.Background(), app.HomeTimeline, map[string]any{"cursor": "next-page"})
	if err != nil || !result.OK {
		t.Fatalf("home failed: %#v %v", result, err)
	}
	parsedURL, _ := url.Parse(prepared.URL)
	if parsedURL.RawQuery != "" || prepared.Headers["content-type"] != "application/json" {
		t.Fatalf("unexpected JSON POST request: %#v", prepared)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(prepared.Body), &body); err != nil {
		t.Fatal(err)
	}
	variables := body["variables"].(map[string]any)
	if body["queryId"] != "current-id" || variables["includePromotedContent"] != true || variables["cursor"] != "next-page" {
		t.Fatalf("unexpected JSON POST body: %#v", body)
	}
}

func TestClientClassifiesAndSanitizesFailures(t *testing.T) {
	cases := []struct {
		name     string
		response app.UpstreamResponse
		code     string
	}{
		{"authentication", app.UpstreamResponse{Status: 401}, "AUTHENTICATION_FAILED"},
		{"rate limit", app.UpstreamResponse{Status: 429}, "RATE_LIMITED"},
		{"response shape", app.UpstreamResponse{Status: 200, Body: "<html>challenge</html>"}, "RESPONSE_SHAPE_CHANGED"},
		{"contract", app.UpstreamResponse{Status: 200, Body: `{"errors":[{"message":"PersistedQueryNotFound"}]}`}, "CONTRACT_FAILED"},
		{"ambiguous", app.UpstreamResponse{Status: 403, Body: `{"auth_token":"session-secret","padding":"` + strings.Repeat("x", 5000) + `"}`}, "UPSTREAM_REJECTED"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			transport := successfulTransport()
			transport.home = func(app.PreparedRequest) (app.UpstreamResponse, error) { return test.response, nil }
			result, err := New(testContracts(), app.AuthenticationState{}, transport, nil).Execute(context.Background(), app.HomeTimeline, nil)
			if err != nil || result.OK || result.Error.Code != test.code {
				t.Fatalf("unexpected result: %#v %v", result, err)
			}
			if result.Upstream == nil || strings.Contains(result.Upstream.Body, "session-secret") || len(result.Upstream.Body) > maximumUpstreamBodyLength {
				t.Fatalf("unsafe upstream: %#v", result.Upstream)
			}
			if test.code == "CONTRACT_FAILED" && result.Error.RecoveryCommand != "twt contract refresh" {
				t.Fatalf("missing recovery command: %#v", result.Error)
			}
		})
	}
}

func TestTransportErrorPropagates(t *testing.T) {
	want := errors.New("network")
	transport := successfulTransport()
	transport.home = func(app.PreparedRequest) (app.UpstreamResponse, error) { return app.UpstreamResponse{}, want }
	_, err := New(testContracts(), app.AuthenticationState{}, transport, nil).Execute(context.Background(), app.HomeTimeline, nil)
	if !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
}
