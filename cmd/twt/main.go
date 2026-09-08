package main

import (
	"context"
	"os"

	searchcommanddependency "github.com/ach968/x-twt-cli/cmd/twt/dependencies/commands/search"
	"github.com/ach968/x-twt-cli/internal/app/cli"
)

func main() {
	application := cli.New(cli.Dependencies{Search: searchcommanddependency.New()})
	status := application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(status)
}
