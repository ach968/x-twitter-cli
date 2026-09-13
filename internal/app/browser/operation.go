package browser

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/go-rod/rod/lib/proto"
)

type BrowserOperationExecution struct {
	ProfilePath     string
	NavigationURL   string
	MatchesResponse func(string) bool
	Timeout         time.Duration
}

type capturedResponse struct {
	status      int
	contentType *string
	body        string
	err         error
}

func networkHeader(headers proto.NetworkHeaders, name string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value.Str(), true
		}
	}
	return "", false
}

func ExecuteBrowserOperation(options BrowserOperationExecution) (result app.UpstreamResponse, err error) {
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	client, err := newBrowserClient(options.ProfilePath, true, options.Timeout)
	if err != nil {
		return result, err
	}
	defer func() {
		if closeErr := client.Close(); err == nil {
			err = closeErr
		}
	}()
	page := client.page
	responses := make(chan capturedResponse, 1)
	var mutex sync.Mutex
	var target *proto.NetworkResponseReceived
	wait := page.EachEvent(
		func(event *proto.NetworkResponseReceived) {
			if options.MatchesResponse(event.Response.URL) {
				mutex.Lock()
				copy := *event
				target = &copy
				mutex.Unlock()
			}
		},
		func(event *proto.NetworkLoadingFinished) bool {
			mutex.Lock()
			matched := target
			mutex.Unlock()
			if matched == nil || event.RequestID != matched.RequestID {
				return false
			}
			body, bodyErr := (proto.NetworkGetResponseBody{RequestID: event.RequestID}).Call(page)
			var text string
			if bodyErr == nil {
				text = body.Body
				if body.Base64Encoded {
					decoded, decodeErr := base64.StdEncoding.DecodeString(text)
					if decodeErr != nil {
						bodyErr = decodeErr
					} else {
						text = string(decoded)
					}
				}
			}
			var contentType *string
			if value, ok := networkHeader(matched.Response.Headers, "content-type"); ok {
				contentType = &value
			}
			responses <- capturedResponse{status: matched.Response.Status, contentType: contentType, body: text, err: bodyErr}
			return true
		},
	)
	go wait()
	if err := page.Navigate(options.NavigationURL); err != nil {
		return result, err
	}
	select {
	case response := <-responses:
		if response.err != nil {
			return result, response.err
		}
		return app.UpstreamResponse{Status: response.status, ContentType: response.contentType, Body: response.body}, nil
	case <-time.After(options.Timeout):
		return result, fmt.Errorf("timed out waiting for operation response")
	}
}
