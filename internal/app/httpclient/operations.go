package httpclient

import (
	"context"
	"maps"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func (client *Client) HomeTimeline(ctx context.Context, overrides map[string]any) (app.OperationResult, error) {
	if overrides == nil {
		overrides = map[string]any{}
	}
	request, err := prepareRequest(client.contracts.Operations[app.HomeTimeline], client.authentication, overrides)
	if err != nil {
		return app.OperationResult{}, err
	}
	response, err := client.transport.Execute(ctx, app.HomeTimeline, request)
	if err != nil {
		return app.OperationResult{}, err
	}
	return resultFromResponse(app.HomeTimeline, response), nil
}

func (client *Client) SearchTimeline(ctx context.Context, query string, overrides map[string]any) (app.OperationResult, error) {
	if overrides == nil {
		overrides = map[string]any{}
	}
	variables := map[string]any{"rawQuery": query}
	maps.Copy(variables, overrides)
	request, err := prepareRequest(client.contracts.Operations[app.SearchTimeline], client.authentication, variables)
	if err != nil {
		return app.OperationResult{}, err
	}
	if err := addTransactionID(&request, client.transactionIDs); err != nil {
		return app.OperationResult{}, err
	}
	response, err := client.transport.Execute(ctx, app.SearchTimeline, request)
	if err != nil {
		return app.OperationResult{}, err
	}
	return resultFromResponse(app.SearchTimeline, response), nil
}
