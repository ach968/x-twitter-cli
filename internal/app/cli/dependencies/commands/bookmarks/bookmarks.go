package bookmarks

import (
	"context"
	"errors"
	"io"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands"
	bookmarkoperation "github.com/ach968/x-twt-cli/internal/app/operations/bookmarks"
)

type Requester interface {
	Execute(context.Context, bookmarkoperation.Request) (bookmarkoperation.Page, error)
}

func New(requester Requester) commands.Command {
	return commands.Command{Run: run(requester), WriteHelp: WriteHelp}
}

func run(requester Requester) commands.Handler {
	return func(ctx context.Context, arguments []string, _ io.Reader, stdout, stderr io.Writer) int {
		if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
			WriteHelp(stdout)
			return commands.ExitSuccess
		}
		query, cursor, err := parseArguments(arguments)
		if err != nil {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", err.Error())
		}
		if requester == nil {
			return commands.WriteFailure(stderr, "BOOKMARKS_FAILED", "Unable to execute bookmarks")
		}
		page, err := requester.Execute(ctx, bookmarkoperation.Request{Query: query, Cursor: cursor})
		if err != nil {
			var failure *app.OperationFailure
			if errors.As(err, &failure) {
				return commands.WriteJSON(stderr, failure, commands.ExitFailure)
			}
			return commands.WriteFailure(stderr, "BOOKMARKS_FAILED", "Unable to execute bookmarks")
		}
		return commands.WriteJSON(stdout, page, commands.ExitSuccess)
	}
}

func parseArguments(arguments []string) (*string, *string, error) {
	var query, cursor *string
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		name, value, equal := strings.Cut(argument, "=")
		switch name {
		case "--search", "--cursor":
			if !equal {
				index++
				if index >= len(arguments) {
					return nil, nil, errors.New(name + " requires a value")
				}
				value = arguments[index]
			}
			if name == "--search" {
				if query != nil {
					return nil, nil, errors.New("--search may be supplied once")
				}
				if strings.TrimSpace(value) == "" {
					return nil, nil, errors.New("--search requires a non-empty query")
				}
				query = &value
			} else {
				if cursor != nil {
					return nil, nil, errors.New("--cursor may be supplied once")
				}
				cursor = &value
			}
		default:
			return nil, nil, errors.New("unknown bookmarks flag: " + argument)
		}
	}
	return query, cursor, nil
}

func WriteHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt bookmarks [--search query] [--cursor value]\n\nLists private bookmarked posts, or searches them with X when --search is supplied. The cursor is opaque; repeat the same query with a returned cursor for the next page. Warnings preserve usable partial output. If a contract becomes stale, run `twt contract refresh`.\n")
}
