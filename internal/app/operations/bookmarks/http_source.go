package bookmarks

import (
	"context"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

type DirectExecutor interface {
	Execute(context.Context, app.OperationName, map[string]any) (app.OperationResult, error)
}

type directSource struct{ executor DirectExecutor }

func NewDirectSource(executor DirectExecutor) Source { return directSource{executor: executor} }

func (source directSource) Fetch(ctx context.Context, request Request) (any, error) {
	if source.executor == nil {
		return nil, failed()
	}
	operation := app.Bookmarks
	overrides := map[string]any{}
	if request.Query != nil {
		operation = app.BookmarkSearchTimeline
		overrides["rawQuery"] = *request.Query
	}
	if request.Cursor != nil {
		overrides["cursor"] = *request.Cursor
	}
	result, err := source.executor.Execute(ctx, operation, overrides)
	if err != nil {
		return nil, err
	}
	if !result.OK {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, failed()
	}
	return result.Payload, nil
}
