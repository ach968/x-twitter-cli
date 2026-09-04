package httpclient

import (
	"context"
	"io"
	"net/http"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
)

type Transport struct {
	Client *http.Client
}

func NewTransport(client *http.Client) *Transport {
	if client == nil {
		client = http.DefaultClient
	}
	return &Transport{Client: client}
}

func (transport *Transport) Execute(ctx context.Context, _ app.OperationName, prepared app.PreparedRequest) (app.UpstreamResponse, error) {
	var body io.Reader
	if prepared.Body != "" {
		body = strings.NewReader(prepared.Body)
	}
	request, err := http.NewRequestWithContext(ctx, prepared.Method, prepared.URL, body)
	if err != nil {
		return app.UpstreamResponse{}, err
	}
	for key, value := range prepared.Headers {
		request.Header.Set(key, value)
	}
	response, err := transport.Client.Do(request)
	if err != nil {
		return app.UpstreamResponse{}, err
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return app.UpstreamResponse{}, err
	}
	var contentType *string
	if value := response.Header.Get("Content-Type"); value != "" {
		contentType = &value
	}
	return app.UpstreamResponse{Status: response.StatusCode, ContentType: contentType, Body: string(contents)}, nil
}
