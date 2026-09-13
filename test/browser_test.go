//go:build browser

package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/browser"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
)

type transportFunc struct {
	home           func(app.PreparedRequest) (app.UpstreamResponse, error)
	search         func(app.PreparedRequest) (app.UpstreamResponse, error)
	bookmarks      func(app.PreparedRequest) (app.UpstreamResponse, error)
	bookmarkSearch func(app.PreparedRequest) (app.UpstreamResponse, error)
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
	case app.Bookmarks:
		return transport.bookmarks(request)
	case app.BookmarkSearchTimeline:
		return transport.bookmarkSearch(request)
	default:
		return app.UpstreamResponse{}, errors.New("unsupported operation")
	}
}

func successfulTestTransport() transportFunc {
	success := app.UpstreamResponse{Status: 200, Body: `{"data":{"ok":true}}`}
	return transportFunc{
		home:           func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		search:         func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		bookmarks:      func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
		bookmarkSearch: func(app.PreparedRequest) (app.UpstreamResponse, error) { return success, nil },
	}
}

func testContracts() app.ContractProperties {
	return app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{
		app.HomeTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/home-id/HomeTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.SearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/search-id/SearchTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20), "querySource": "typed_query"}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.Bookmarks: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmarks-id/Bookmarks", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.BookmarkSearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmark-search-id/BookmarkSearchTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
	}}
}

func TestApplicationProfileHidesAutomationAndPersistsAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/login":
			http.SetCookie(response, &http.Cookie{Name: "x_session", Value: "recognized", Path: "/", HttpOnly: true, MaxAge: 3600})
			fmt.Fprint(response, "<title>login-complete</title>")
		case "/automation-status":
			fmt.Fprint(response, `<script>document.title = navigator.webdriver ? "automation-visible" : "automation-hidden";</script>`)
		default:
			_, err := request.Cookie("x_session")
			if err == nil {
				fmt.Fprint(response, "<title>session-recognized</title>")
			} else {
				fmt.Fprint(response, "<title>login-required</title>")
			}
		}
	}))
	defer server.Close()
	profile := filepath.Join(t.TempDir(), "profile")
	automation, err := browser.VisitWithApplicationProfile(browser.ApplicationProfileVisit{ProfilePath: profile, Headless: true, URL: server.URL + "/automation-status"})
	if err != nil || automation.Title != "automation-hidden" {
		t.Fatalf("automation signal: %#v %v", automation, err)
	}
	authenticated, err := browser.VisitWithApplicationProfile(browser.ApplicationProfileVisit{ProfilePath: profile, Headless: true, URL: server.URL + "/login"})
	if err != nil || authenticated.Title != "login-complete" {
		t.Fatalf("login: %#v %v", authenticated, err)
	}
	reused, err := browser.VisitWithApplicationProfile(browser.ApplicationProfileVisit{ProfilePath: profile, Headless: true, URL: server.URL + "/status"})
	if err != nil || reused.Title != "session-recognized" {
		t.Fatalf("profile reuse: %#v %v", reused, err)
	}
}

func TestApplicationProfileLockAndStaleRecovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(response, "<title>ready</title>")
	}))
	defer server.Close()
	profile := filepath.Join(t.TempDir(), "profile")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profile+".lock", []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := browser.VisitWithApplicationProfile(browser.ApplicationProfileVisit{ProfilePath: profile, Headless: true, URL: server.URL})
	var locked *browser.BrowserProfileLockedError
	if !errors.As(err, &locked) || locked.Code != "BROWSER_PROFILE_LOCKED" {
		t.Fatalf("expected profile lock error, got %v", err)
	}
	if err := os.Remove(profile + ".lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profile+".lock", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := browser.VisitWithApplicationProfile(browser.ApplicationProfileVisit{ProfilePath: profile, Headless: true, URL: server.URL}); err != nil {
		t.Fatalf("stale lock was not recovered: %v", err)
	}
	if err := os.WriteFile(profile+".lock", []byte("999999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := browser.VisitWithApplicationProfile(browser.ApplicationProfileVisit{ProfilePath: profile, Headless: true, URL: server.URL}); err != nil {
		t.Fatalf("dead-owner lock was not recovered: %v", err)
	}
}

func graphQLPath(id string, operation app.OperationName, variables map[string]any) string {
	encodedVariables, _ := json.Marshal(variables)
	encodedFeatures, _ := json.Marshal(map[string]any{"timeline_enabled": true})
	encodedToggles, _ := json.Marshal(map[string]any{"withArticleRichContentState": false})
	query := url.Values{}
	query.Set("variables", string(encodedVariables))
	query.Set("features", string(encodedFeatures))
	query.Set("fieldToggles", string(encodedToggles))
	return "/i/api/graphql/" + id + "/" + string(operation) + "?" + query.Encode()
}

func TestOperationContractCapture(t *testing.T) {
	homePath := graphQLPath("home-id", app.HomeTimeline, map[string]any{"count": 20, "requestContext": "launch"})
	searchPath := graphQLPath("search-id", app.SearchTimeline, map[string]any{"count": 20, "rawQuery": "x"})
	bookmarksPath := graphQLPath("bookmarks-id", app.Bookmarks, map[string]any{"count": 20})
	bookmarkSearchPath := graphQLPath("bookmark-search-id", app.BookmarkSearchTimeline, map[string]any{"count": 20, "rawQuery": "x-twitter-cli-contract-validation-improbable-6d1e2f"})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html")
		switch request.URL.Path {
		case "/capture":
			http.SetCookie(response, &http.Cookie{Name: "ct0", Value: "csrf-secret", Path: "/", MaxAge: 3600})
			fmt.Fprintf(response, `<script>Promise.all([fetch(%s,{headers:{authorization:"Bearer captured-token"}}),fetch(%s,{headers:{authorization:"Bearer captured-token"}})]).then(()=>document.title="captured")</script>`, strconv.Quote(homePath), strconv.Quote(searchPath))
		case "/home":
			http.SetCookie(response, &http.Cookie{Name: "ct0", Value: "csrf-secret", Path: "/", MaxAge: 3600})
			fmt.Fprintf(response, `<script>fetch(%s,{headers:{authorization:"Bearer captured-token"}})</script>`, strconv.Quote(homePath))
		case "/search":
			fmt.Fprintf(response, `<script>fetch(%s,{headers:{authorization:"Bearer captured-token"}})</script>`, strconv.Quote(searchPath))
		case "/bookmarks":
			fmt.Fprintf(response, `<button aria-label="Search Bookmarks" onclick='const control=document.createElement("input");control.placeholder="Search Bookmarks";control.oninput=()=>fetch(%s,{headers:{authorization:"Bearer captured-token"}});this.after(control)'>Search</button><script>fetch(%s,{headers:{authorization:"Bearer captured-token"}})</script>`, strconv.Quote(bookmarkSearchPath), strconv.Quote(bookmarksPath))
		default:
			response.Header().Set("Content-Type", "application/json")
			fmt.Fprint(response, `{"data":{"ok":true}}`)
		}
	}))
	defer server.Close()
	host := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")[0]
	capture, err := browser.CaptureOperationContracts(browser.ContractCaptureOptions{
		ProfilePath: filepath.Join(t.TempDir(), "profile"), Headless: true, CaptureHost: host,
		Steps: []browser.ContractCaptureStep{{URL: server.URL + "/capture", WaitFor: []app.OperationName{app.HomeTimeline, app.SearchTimeline}}, {URL: server.URL + "/bookmarks", WaitFor: []app.OperationName{app.Bookmarks}, TriggerBookmarkSearch: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	home := capture.Contracts.Operations[app.HomeTimeline]
	if home.Path != "/i/api/graphql/home-id/HomeTimeline" || home.Variables["requestContext"] != "launch" || capture.Authentication.Authorization != "Bearer captured-token" {
		t.Fatalf("unexpected capture: %#v", capture)
	}
	serialized, _ := json.Marshal(capture.Contracts)
	if strings.Contains(string(serialized), "csrf-secret") || strings.Contains(string(serialized), "captured-token") {
		t.Fatal("secrets leaked into contract properties")
	}

	stepped, err := browser.CaptureOperationContracts(browser.ContractCaptureOptions{
		ProfilePath: filepath.Join(t.TempDir(), "profile"), Headless: true, CaptureHost: host,
		Steps: []browser.ContractCaptureStep{{URL: server.URL + "/capture", WaitFor: []app.OperationName{app.HomeTimeline, app.SearchTimeline}}, {URL: server.URL + "/bookmarks", WaitFor: []app.OperationName{app.Bookmarks}, TriggerBookmarkSearch: true}},
	})
	if err != nil || len(stepped.Contracts.Operations) != 4 {
		t.Fatalf("stepped capture: %#v %v", stepped, err)
	}
}

func TestOperationContractCapturePreservesJSONPostBody(t *testing.T) {
	searchPath := graphQLPath("search-id", app.SearchTimeline, map[string]any{"count": 20, "rawQuery": "x"})
	bookmarksPath := graphQLPath("bookmarks-id", app.Bookmarks, map[string]any{"count": 20})
	bookmarkSearchPath := graphQLPath("bookmark-search-id", app.BookmarkSearchTimeline, map[string]any{"count": 20, "rawQuery": "validation"})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html")
		if request.URL.Path == "/capture" {
			fmt.Fprintf(response, `<script>Promise.all([
				fetch("/i/api/graphql/home-id/HomeTimeline", {
					method: "POST",
					headers: {authorization: "Bearer captured-token", "content-type": "application/json"},
					body: JSON.stringify({queryId:"home-id",variables:{count:20,includePromotedContent:true},features:{timeline_enabled:true}})
				}),
				fetch(%s,{headers:{authorization:"Bearer captured-token"}}),
				fetch(%s,{headers:{authorization:"Bearer captured-token"}}),
				fetch(%s,{headers:{authorization:"Bearer captured-token"}})
			])</script>`, strconv.Quote(searchPath), strconv.Quote(bookmarksPath), strconv.Quote(bookmarkSearchPath))
			return
		}
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprint(response, `{"data":{"ok":true}}`)
	}))
	defer server.Close()
	host := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")[0]

	capture, err := browser.CaptureOperationContracts(browser.ContractCaptureOptions{
		ProfilePath: filepath.Join(t.TempDir(), "profile"),
		Headless:    true,
		URLs:        []string{server.URL + "/capture"},
		CaptureHost: host,
	})
	if err != nil {
		t.Fatal(err)
	}
	home := capture.Contracts.Operations[app.HomeTimeline]
	variables, _ := home.Body["variables"].(map[string]any)
	if home.Encoding != "json" || home.Body["queryId"] != "home-id" || variables["includePromotedContent"] != true {
		t.Fatalf("JSON POST body was not captured: %#v", home)
	}
}

func TestOperationContractCaptureRejectsAPartialRequiredSet(t *testing.T) {
	homePath := graphQLPath("home-id", app.HomeTimeline, map[string]any{"count": 20})
	searchPath := graphQLPath("search-id", app.SearchTimeline, map[string]any{"count": 20, "rawQuery": "x"})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html")
		if request.URL.Path == "/capture" {
			fmt.Fprintf(response, `<script>Promise.all([fetch(%s,{headers:{authorization:"Bearer captured-token"}}),fetch(%s,{headers:{authorization:"Bearer captured-token"}})])</script>`, strconv.Quote(homePath), strconv.Quote(searchPath))
			return
		}
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprint(response, `{"data":{"ok":true}}`)
	}))
	defer server.Close()
	host := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")[0]

	_, err := browser.CaptureOperationContracts(browser.ContractCaptureOptions{
		ProfilePath: filepath.Join(t.TempDir(), "profile"),
		Headless:    true,
		URLs:        []string{server.URL + "/capture"},
		CaptureHost: host,
		Timeout:     500 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "Bookmarks") {
		t.Fatalf("expected missing required bookmark operations, got %v", err)
	}
}

func TestBrowserTransportRoutesOnlySearchThroughThePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html")
		switch request.URL.Path {
		case "/home":
			fmt.Fprint(response, `<script>fetch("/i/api/graphql/current-id/HomeTimeline",{method:"POST",headers:{"content-type":"application/json","x-client-transaction-id":"generated-by-page"},body:JSON.stringify({variables:{count:20}})})</script>`)
		case "/search":
			query := request.URL.Query().Get("q")
			variables, _ := json.Marshal(map[string]any{"rawQuery": query, "querySource": "typed_query"})
			path := "/i/api/graphql/current-id/SearchTimeline?variables=" + url.QueryEscape(string(variables))
			fmt.Fprintf(response, `<script>fetch(%s,{headers:{"x-client-transaction-id":"generated-by-page"}})</script>`, strconv.Quote(path))
		case "/i/api/graphql/current-id/SearchTimeline":
			response.Header().Set("Content-Type", "application/json")
			if request.Header.Get("x-client-transaction-id") == "" {
				response.WriteHeader(404)
				return
			}
			var variables map[string]any
			json.Unmarshal([]byte(request.URL.Query().Get("variables")), &variables)
			json.NewEncoder(response).Encode(map[string]any{"data": map[string]any{"search": map[string]any{"query": variables["rawQuery"]}}})
		case "/i/api/graphql/current-id/HomeTimeline":
			response.Header().Set("Content-Type", "application/json")
			fmt.Fprint(response, `{"data":{"home":{"count":20}}}`)
		default:
			response.WriteHeader(404)
		}
	}))
	defer server.Close()
	direct := successfulTestTransport()
	direct.home = func(app.PreparedRequest) (app.UpstreamResponse, error) {
		return app.UpstreamResponse{Status: 200, Body: `{"data":{"home":{"source":"direct"}}}`}, nil
	}
	transport := browser.NewBrowserTransport(filepath.Join(t.TempDir(), "profile"), server.URL, 30*time.Second, direct)
	transactionIDs := transactionIDFunc(func(string, string) (string, error) {
		return "generated-for-operation", nil
	})
	requestClient := httpclient.New(testContracts(), app.AuthenticationState{}, transport, transactionIDs)
	home, err := requestClient.Execute(context.Background(), app.HomeTimeline, nil)
	if err != nil || !home.OK {
		t.Fatalf("home: %#v %v", home, err)
	}
	homePayload, _ := json.Marshal(home.Payload)
	if !strings.Contains(string(homePayload), `"source":"direct"`) {
		t.Fatalf("home did not use direct HTTP: %s", homePayload)
	}
	search, err := requestClient.Execute(context.Background(), app.SearchTimeline, map[string]any{"rawQuery": "golang", "product": "Top"})
	if err != nil || !search.OK {
		t.Fatalf("search: %#v %v", search, err)
	}
}
