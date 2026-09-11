package search

import (
	"context"
	"errors"
	"io"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands"
	searchoperation "github.com/ach968/x-twt-cli/internal/app/operations/search"
)

type Requester interface {
	Execute(context.Context, searchoperation.Request) (searchoperation.Page, error)
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
		query, tab, cursor, err := parseArguments(arguments)
		if err != nil {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", err.Error())
		}
		if requester == nil {
			return commands.WriteFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
		}
		page, err := requester.Execute(ctx, searchoperation.Request{Query: query, Tab: tab, Cursor: cursor})
		if err != nil {
			var failure *app.OperationFailure
			if errors.As(err, &failure) {
				return commands.WriteJSON(stderr, failure, commands.ExitFailure)
			}
			return commands.WriteFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
		}
		return commands.WriteJSON(stdout, page, commands.ExitSuccess)
	}
}

func parseArguments(arguments []string) (string, searchoperation.Tab, *string, error) {
	tab := searchoperation.TabTop
	var cursor *string
	positionals := make([]string, 0, 1)
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		name, value, hasValue := strings.Cut(argument, "=")
		switch name {
		case "--tab":
			if !hasValue {
				index++
				if index >= len(arguments) {
					return "", "", nil, errors.New("--tab requires a value")
				}
				value = arguments[index]
			}
			parsed, ok := searchoperation.ParseTab(value)
			if !ok {
				return "", "", nil, errors.New("tab must be top, latest, people, media, or lists")
			}
			tab = parsed
		case "--cursor":
			if !hasValue {
				index++
				if index >= len(arguments) {
					return "", "", nil, errors.New("--cursor requires a value")
				}
				value = arguments[index]
			}
			cursor = &value
		default:
			if strings.HasPrefix(argument, "-") {
				return "", "", nil, errors.New("unknown search flag: " + argument)
			}
			positionals = append(positionals, argument)
		}
	}
	if len(positionals) == 1 && strings.TrimSpace(positionals[0]) == "" {
		return "", "", nil, errors.New("search query is empty; shells expand $NAME inside double quotes, so use single quotes (for example: twt search '$NVDA') or escape the dollar sign")
	}
	if len(positionals) != 1 {
		return "", "", nil, errors.New("search requires exactly one non-empty query")
	}
	return positionals[0], tab, cursor, nil
}

func WriteHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt search <query> [--tab top|latest|people|media|lists] [--cursor value]\n")
}
