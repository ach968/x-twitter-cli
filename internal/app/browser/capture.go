package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

type ContractCaptureStep struct {
	URL                   string
	WaitFor               []app.OperationName
	TriggerBookmarkSearch bool
	TriggerPostView       bool
}

type captureAction struct {
	operation app.OperationName
	run       func(*rod.Page) error
}

func (step ContractCaptureStep) actions() []captureAction {
	var actions []captureAction
	if step.TriggerPostView {
		actions = append(actions, captureAction{app.TweetDetail, triggerPostView})
	}
	if step.TriggerBookmarkSearch {
		actions = append(actions, captureAction{app.BookmarkSearchTimeline, triggerBookmarkSearch})
	}
	return actions
}

// CapturedOperations lists the responses this step waits for, including actions.
func (step ContractCaptureStep) CapturedOperations() []app.OperationName {
	names := append([]app.OperationName(nil), step.WaitFor...)
	for _, action := range step.actions() {
		names = append(names, action.operation)
	}
	return names
}

type ContractCaptureOptions struct {
	ProfilePath string
	Headless    bool
	URLs        []string
	Steps       []ContractCaptureStep
	CaptureHost string
	Timeout     time.Duration
}

// bookmarkSearchValidationQuery is deliberately improbable so contract refresh
// proves request execution without relying on private bookmark content.
const bookmarkSearchValidationQuery = "x-twitter-cli-contract-validation-improbable-6d1e2f"

type AuthenticationRequiredError struct {
	URL string
}

func (err *AuthenticationRequiredError) Error() string {
	return "X authentication is required"
}

func authenticationRequiredURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	path := strings.ToLower(parsed.Path)
	return path == "/login" || strings.HasPrefix(path, "/i/flow/login") || strings.HasPrefix(path, "/account/access")
}

func headlessAuthenticationError(client *browserClient) error {
	info, err := client.page.Info()
	if err != nil {
		return nil
	}
	if authenticationRequiredURL(info.URL) {
		return &AuthenticationRequiredError{URL: info.URL}
	}
	parsed, err := url.Parse(info.URL)
	if err != nil {
		return nil
	}
	host := parsed.Hostname()
	if host != "x.com" && !strings.HasSuffix(host, ".x.com") {
		return nil
	}
	cookies, err := client.launched.browser.GetCookies()
	if err != nil {
		return err
	}
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" && cookie.Value != "" {
			return nil
		}
	}
	return &AuthenticationRequiredError{URL: info.URL}
}

func recordParameter(parsed *url.URL, name string) (map[string]any, error) {
	value := parsed.Query().Get(name)
	if value == "" {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(value), &result); err != nil || result == nil {
		return nil, fmt.Errorf("%s must be a JSON object", name)
	}
	return result, nil
}

func recordObject(value map[string]any, name string) (map[string]any, error) {
	child, present := value[name]
	if !present {
		return map[string]any{}, nil
	}
	result, ok := child.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a JSON object", name)
	}
	return result, nil
}

type capturedOperationResult struct {
	Name          app.OperationName
	Contract      app.OperationContract
	Authorization string
}

func captureOperation(request *proto.NetworkRequest) (*capturedOperationResult, error) {
	parsed, err := url.Parse(request.URL)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "i" || parts[1] != "api" || parts[2] != "graphql" {
		return nil, nil
	}
	name := app.OperationName(parts[4])
	if _, supported := app.LookupOperation(name); !supported {
		return nil, nil
	}
	if request.Method != "GET" && request.Method != "POST" {
		return nil, nil
	}
	var body map[string]any
	var variables, features, fieldToggles map[string]any
	encoding := "query"
	if request.Method == "POST" {
		encoding = "json"
		if request.PostData == "" {
			return nil, errors.New("GraphQL JSON POST has no captured body")
		}
		if err := json.Unmarshal([]byte(request.PostData), &body); err != nil || body == nil {
			return nil, errors.New("GraphQL POST body must be a JSON object")
		}
		variables, err = recordObject(body, "variables")
		if err != nil {
			return nil, err
		}
		features, err = recordObject(body, "features")
		if err != nil {
			return nil, err
		}
		fieldToggles, err = recordObject(body, "fieldToggles")
		if err != nil {
			return nil, err
		}
	} else {
		variables, err = recordParameter(parsed, "variables")
		if err != nil {
			return nil, err
		}
		features, err = recordParameter(parsed, "features")
		if err != nil {
			return nil, err
		}
		fieldToggles, err = recordParameter(parsed, "fieldToggles")
		if err != nil {
			return nil, err
		}
	}
	authorization, _ := networkHeader(request.Headers, "authorization")
	if name == app.TweetDetail {
		delete(variables, "cursor")
	}
	return &capturedOperationResult{Name: name, Contract: app.OperationContract{
		Family: "graphql", Host: parsed.Hostname(), Path: parsed.Path, Method: request.Method, Encoding: encoding,
		Body: body, Variables: variables, Features: features, FieldToggles: fieldToggles,
	}, Authorization: authorization}, nil
}

// triggerPostView opens a post actually present on the current search page,
// avoiding a permanent dependency on a hard-coded public post remaining available.
func triggerPostView(page *rod.Page) error {
	link, err := page.Element(`article a[href*="/status/"]:has(time)`)
	if err != nil {
		return errors.New("Unable to find a post for View contract capture")
	}
	href, err := link.Attribute("href")
	if err != nil || href == nil {
		return errors.New("Unable to read the post link for View contract capture")
	}
	info, err := page.Info()
	if err != nil {
		return err
	}
	base, err := url.Parse(info.URL)
	if err != nil {
		return err
	}
	target, err := base.Parse(*href)
	if err != nil || target.Scheme != base.Scheme || target.Host != base.Host || !strings.Contains(target.Path, "/status/") {
		return errors.New("Unexpected post link during View contract capture")
	}
	return page.Navigate(target.String())
}

// triggerBookmarkSearch contains the X-page interaction recipe. Callers only
// ask to capture operation contracts and never receive selectors or controls.
func triggerBookmarkSearch(page *rod.Page) error {
	const searchInputSelector = `input[placeholder="Search Bookmarks"]`
	hasInput, _, err := page.Has(searchInputSelector)
	if err != nil {
		return fmt.Errorf("inspect Search Bookmarks control: %w", err)
	}
	if !hasInput {
		launcher, err := page.Element(`button[aria-label="Search Bookmarks"]`)
		if err != nil {
			return fmt.Errorf("locate Search Bookmarks launcher: %w", err)
		}
		if err := launcher.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return fmt.Errorf("open Search Bookmarks control: %w", err)
		}
	}
	control, err := page.Element(searchInputSelector)
	if err != nil {
		return fmt.Errorf("locate Search Bookmarks control: %w", err)
	}
	if err := control.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("focus Search Bookmarks control: %w", err)
	}
	if err := control.Input(bookmarkSearchValidationQuery); err != nil {
		return fmt.Errorf("enter bookmark-search validation query: %w", err)
	}
	if err := control.Type(input.Enter); err != nil {
		return fmt.Errorf("submit bookmark-search validation query: %w", err)
	}
	return nil
}

func CaptureOperationContracts(options ContractCaptureOptions) (result app.CapturedState, err error) {
	if options.Timeout == 0 {
		options.Timeout = 10 * time.Second
	}
	client, err := newBrowserClient(options.ProfilePath, options.Headless, options.Timeout)
	if err != nil {
		return result, err
	}
	defer func() {
		if closeErr := client.Close(); err == nil {
			err = closeErr
		}
	}()
	page := client.page
	var mutex sync.Mutex
	operations := map[app.OperationName]app.OperationContract{}
	authorization := ""
	var captureErr error
	notify := make(chan struct{}, 1)
	waitEvents := page.EachEvent(func(event *proto.NetworkRequestWillBeSent) {
		parsed, parseErr := url.Parse(event.Request.URL)
		allowed := parseErr == nil && (parsed.Hostname() == "x.com" || strings.HasSuffix(parsed.Hostname(), ".x.com"))
		if options.CaptureHost != "" {
			allowed = parseErr == nil && parsed.Hostname() == options.CaptureHost
		}
		if !allowed {
			return
		}
		captured, eventErr := captureOperation(event.Request)
		mutex.Lock()
		defer mutex.Unlock()
		if eventErr != nil && captureErr == nil {
			captureErr = eventErr
		}
		if captured != nil {
			operations[captured.Name] = captured.Contract
			if authorization == "" {
				authorization = captured.Authorization
			}
		}
		select {
		case notify <- struct{}{}:
		default:
		}
	})
	go waitEvents()
	waitForOperations := func(names []app.OperationName) error {
		timer := time.NewTimer(options.Timeout)
		defer timer.Stop()
		authenticationTicker := time.NewTicker(500 * time.Millisecond)
		defer authenticationTicker.Stop()
		for {
			if options.Headless {
				if authErr := headlessAuthenticationError(client); authErr != nil {
					return authErr
				}
			}
			mutex.Lock()
			ready := true
			for _, name := range names {
				if _, ok := operations[name]; !ok {
					ready = false
					break
				}
			}
			currentErr := captureErr
			mutex.Unlock()
			if currentErr != nil {
				return currentErr
			}
			if ready {
				return nil
			}
			select {
			case <-notify:
			case <-authenticationTicker.C:
			case <-timer.C:
				if options.Headless {
					if info, infoErr := page.Info(); infoErr == nil && authenticationRequiredURL(info.URL) {
						return &AuthenticationRequiredError{URL: info.URL}
					}
				}
				values := make([]string, len(names))
				for i, name := range names {
					values[i] = string(name)
				}
				return fmt.Errorf("Timed out waiting for operation contracts: %s", strings.Join(values, ", "))
			}
		}
	}
	steps := options.Steps
	if len(steps) == 0 {
		for index, value := range options.URLs {
			waitFor := []app.OperationName{}
			if index == len(options.URLs)-1 {
				waitFor = []app.OperationName{app.SearchTimeline}
			}
			steps = append(steps, ContractCaptureStep{URL: value, WaitFor: waitFor})
		}
	}
	if len(steps) == 0 {
		return result, errors.New("At least one capture URL or step is required")
	}
	for _, step := range steps {
		waitForNavigation := page.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
		if err = page.Navigate(step.URL); err != nil {
			return result, err
		}
		waitForNavigation()
		if options.Headless {
			if authErr := headlessAuthenticationError(client); authErr != nil {
				return result, authErr
			}
		}
		if err = waitForOperations(step.WaitFor); err != nil {
			return result, err
		}
		for _, action := range step.actions() {
			if err = action.run(page); err != nil {
				return result, err
			}
			if err = waitForOperations([]app.OperationName{action.operation}); err != nil {
				return result, err
			}
		}
	}
	var requiredNames []app.OperationName
	for _, policy := range app.OperationPolicies() {
		if policy.ContractFile == app.Required {
			requiredNames = append(requiredNames, policy.Name)
		}
	}
	if err = waitForOperations(requiredNames); err != nil {
		return result, err
	}
	mutex.Lock()
	defer mutex.Unlock()
	if authorization == "" {
		return result, errors.New("Required operation contracts or authentication were not captured")
	}
	cookies, err := client.launched.browser.GetCookies()
	if err != nil {
		return result, err
	}
	authCookies := make([]app.AuthenticationCookie, len(cookies))
	for i, cookie := range cookies {
		authCookies[i] = app.AuthenticationCookie{Name: cookie.Name, Value: cookie.Value}
	}
	capturedContracts := make(map[app.OperationName]app.OperationContract, len(operations))
	for name, operation := range operations {
		capturedContracts[name] = operation
	}
	return app.CapturedState{
		Contracts:      app.ContractProperties{Version: 1, Operations: capturedContracts},
		Authentication: app.AuthenticationState{Cookies: authCookies, Authorization: authorization},
	}, nil
}
