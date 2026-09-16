package main

import (
	"context"
	"os"
	"runtime/debug"

	bookmarkscommand "github.com/ach968/x-twitter-cli/cmd/twt/dependencies/commands/bookmarks"
	searchcommand "github.com/ach968/x-twitter-cli/cmd/twt/dependencies/commands/search"
	"github.com/ach968/x-twitter-cli/internal/app/cli"
	"github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands"
	"github.com/ach968/x-twitter-cli/internal/app/management"
)

var version = "dev"

func installedVersion() string {
	if version != "dev" {
		return version
	}
	if build, ok := debug.ReadBuildInfo(); ok && build.Main.Version != "" && build.Main.Version != "(devel)" {
		return build.Main.Version
	}
	return version
}

func main() {
	managementService, err := management.NewLocal()
	if err != nil {
		os.Exit(commands.WriteFailure(os.Stderr, "LOCAL_STATE_UNAVAILABLE", "Unable to resolve local application state"))
	}
	application := cli.New(cli.Dependencies{Search: searchcommand.New(), Bookmarks: bookmarkscommand.New(), Management: managementService, Version: installedVersion()})
	status := application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(status)
}
