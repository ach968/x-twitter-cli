// Package cli owns command routing, validation, streams, rendering, and exit
// status policy for the twt executable.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands"
	bookmarkscommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/bookmarks"
	managementcommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/management"
	searchcommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/search"
	viewcommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/view"
	"github.com/ach968/x-twitter-cli/internal/app/management"
)

type Dependencies struct {
	Search     searchcommand.Requester
	Bookmarks  bookmarkscommand.Requester
	View       viewcommand.Requester
	Management management.Service
	Version    string
}

type Application struct {
	commands map[string]commands.Command
	version  string
}

func New(dependencies Dependencies) *Application {
	version := dependencies.Version
	if version == "" {
		version = "dev"
	}
	application := &Application{commands: make(map[string]commands.Command), version: version}
	application.commands["search"] = searchcommand.New(dependencies.Search)
	application.commands["bookmarks"] = bookmarkscommand.New(dependencies.Bookmarks)
	application.commands["view"] = viewcommand.New(dependencies.View)
	application.commands["setup"] = managementcommand.NewSetup(dependencies.Management)
	application.commands["auth"] = managementcommand.NewAuth(dependencies.Management)
	application.commands["contract"] = managementcommand.NewContract(dependencies.Management)
	application.commands["version"] = newVersionCommand(version)
	return application
}

func (application *Application) Run(ctx context.Context, arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || (len(arguments) == 1 && (arguments[0] == "help" || arguments[0] == "--help" || arguments[0] == "-h")) {
		writeRootHelp(stdout)
		return commands.ExitSuccess
	}
	if len(arguments) == 1 && arguments[0] == "--version" {
		writeVersion(stdout, application.version)
		return commands.ExitSuccess
	}
	if len(arguments) == 2 && arguments[0] == "help" {
		if selected, ok := application.commands[arguments[1]]; ok {
			selected.WriteHelp(stdout)
			return commands.ExitSuccess
		}
	}
	selected, ok := application.commands[arguments[0]]
	if !ok {
		return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "Unknown command: "+arguments[0])
	}
	return selected.Run(ctx, arguments[1:], stdin, stdout, stderr)
}

func writeRootHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt <command> [options]\n\nCommands:\n  search      Search X\n  bookmarks   List or search bookmarked posts\n  view        View a post and its conversation\n  setup       Prepare managed Chromium\n  auth        Manage X authentication\n  contract    Inspect or refresh operation contracts\n  version     Print the installed version\n\nRun 'twt <command> --help' for command help.\n")
}

func newVersionCommand(version string) commands.Command {
	return commands.Command{
		Run: func(_ context.Context, arguments []string, _ io.Reader, stdout, stderr io.Writer) int {
			if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h" || arguments[0] == "help") {
				writeVersionHelp(stdout)
				return commands.ExitSuccess
			}
			if len(arguments) != 0 {
				return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "version does not accept arguments")
			}
			writeVersion(stdout, version)
			return commands.ExitSuccess
		},
		WriteHelp: writeVersionHelp,
	}
}

func writeVersion(output io.Writer, version string) {
	_, _ = fmt.Fprintf(output, "twt %s\n", version)
}

func writeVersionHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt version\n\nPrint the installed version. Tagged release binaries report their release tag; local development builds report dev. `twt --version` is equivalent.\n")
}
