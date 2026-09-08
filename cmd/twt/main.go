package main

import (
	"context"
	"os"

	searchdependency "github.com/ach968/x-twt-cli/cmd/twt/dependencies/commands/search"
	"github.com/ach968/x-twt-cli/internal/app/cli"
)

func main() {
	application := cli.New(cli.Dependencies{Search: searchdependency.New()})
	status := application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(status)
}
