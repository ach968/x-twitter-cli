package bookmarks

import (
	"github.com/ach968/x-twitter-cli/cmd/twt/dependencies/runtime"
	app "github.com/ach968/x-twitter-cli/internal/app"
	command "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/bookmarks"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	operation "github.com/ach968/x-twitter-cli/internal/app/operations/bookmarks"
)

func New() command.Requester {
	return runtime.New([]app.OperationName{app.Bookmarks, app.BookmarkSearchTimeline}, func(client *httpclient.Client) runtime.Requester[operation.Request, operation.Page] {
		return operation.New(operation.NewDirectSource(client))
	})
}
