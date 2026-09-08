package main

import (
	"context"
	"os"

	searchcommanddependency "github.com/ach968/x-twt-cli/cmd/twt/dependencies/commands/search"
	"github.com/ach968/x-twt-cli/internal/app/cli"
	"github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands"
	"github.com/ach968/x-twt-cli/internal/app/management"
)

func main() {
	managementService, err := management.NewLocal()
	if err != nil {
		os.Exit(commands.WriteFailure(os.Stderr, "LOCAL_STATE_UNAVAILABLE", "Unable to resolve local application state"))
	}
	application := cli.New(cli.Dependencies{Search: searchcommanddependency.New(), Management: managementService})
	status := application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(status)
}
