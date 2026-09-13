package search

import (
	"context"

	app "github.com/ach968/x-twitter-cli3/internal/app"
)

// DirectExecutor is the generic direct-operation seam used by the production
// Search source adapter.
type DirectExecutor interface {
	Execute(context.Context, app.OperationName, map[string]any) (app.OperationResult, error)
}

type directSource struct {
	executor DirectExecutor
}

func NewDirectSource(executor DirectExecutor) Source {
	return directSource{executor: executor}
}

func (source directSource) SearchTimeline(ctx context.Context, request Request) (any, error) {
	if source.executor == nil {
		return nil, &app.OperationFailure{Code: "SEARCH_FAILED", Message: "Unable to execute search"}
	}
	overrides := map[string]any{"rawQuery": request.Query, "product": request.Tab.Product()}
	if request.Cursor != nil {
		overrides["cursor"] = *request.Cursor
	}
	result, err := source.executor.Execute(ctx, app.SearchTimeline, overrides)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, &app.OperationFailure{Code: "SEARCH_FAILED", Message: "Unable to execute search"}
	}
	return result.Payload, nil
}
