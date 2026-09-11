// Package cli owns command routing, validation, streams, rendering, and exit
// status policy for the twt executable.
package cli

import (
	"context"
	"io"

	"github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands"
	bookmarkscommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/bookmarks"
	managementcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/management"
	searchcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/search"
	"github.com/ach968/x-twt-cli/internal/app/management"
)

type Dependencies struct {
	Search     searchcommand.Requester
	Bookmarks  bookmarkscommand.Requester
	Management management.Service
}

type Application struct {
	commands map[string]commands.Command
}

func New(dependencies Dependencies) *Application {
	application := &Application{commands: make(map[string]commands.Command)}
	application.commands["search"] = searchcommand.New(dependencies.Search)
	application.commands["bookmarks"] = bookmarkscommand.New(dependencies.Bookmarks)
	application.commands["setup"] = managementcommand.NewSetup(dependencies.Management)
	application.commands["auth"] = managementcommand.NewAuth(dependencies.Management)
	application.commands["contract"] = managementcommand.NewContract(dependencies.Management)
	return application
}

func (application *Application) Run(ctx context.Context, arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || (len(arguments) == 1 && (arguments[0] == "help" || arguments[0] == "--help" || arguments[0] == "-h")) {
		writeRootHelp(stdout)
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
	_, _ = io.WriteString(output, "Usage: twt <command> [options]\n\nCommands:\n  search      Search X\n  bookmarks   List or search bookmarked posts\n  setup       Prepare managed Chromium\n  auth        Manage X authentication\n  contract    Inspect or refresh operation contracts\n\nRun 'twt <command> --help' for command help.\n")
}
