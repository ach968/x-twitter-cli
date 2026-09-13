package main

import (
	"context"
	"os"

	bookmarkscommanddependency "github.com/ach968/x-twitter-cli/cmd/twt/dependencies/commands/bookmarks"
	searchcommanddependency "github.com/ach968/x-twitter-cli/cmd/twt/dependencies/commands/search"
	"github.com/ach968/x-twitter-cli/internal/app/cli"
	"github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands"
	"github.com/ach968/x-twitter-cli/internal/app/management"
)

func main() {
	managementService, err := management.NewLocal()
	if err != nil {
		os.Exit(commands.WriteFailure(os.Stderr, "LOCAL_STATE_UNAVAILABLE", "Unable to resolve local application state"))
	}
	application := cli.New(cli.Dependencies{Search: searchcommanddependency.New(), Bookmarks: bookmarkscommanddependency.New(), Management: managementService})
	status := application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(status)
}
