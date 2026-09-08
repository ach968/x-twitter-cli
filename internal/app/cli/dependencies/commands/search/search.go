package search

import (
	"context"
	"errors"
	"io"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands"
	searchresult "github.com/ach968/x-twt-cli/internal/app/search"
)

type Requester interface {
	SearchTimeline(context.Context, string, map[string]any) (app.OperationResult, error)
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
		overrides := map[string]any{"product": tab.Product()}
		if cursor != nil {
			overrides["cursor"] = *cursor
		}
		result, err := requester.SearchTimeline(ctx, query, overrides)
		if err != nil {
			return commands.WriteFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
		}
		if !result.OK {
			if result.Error == nil {
				return commands.WriteFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
			}
			return commands.WriteJSON(stderr, result.Error, commands.ExitFailure)
		}
		page, err := searchresult.DecodePage(result.Payload, query, tab)
		if err != nil {
			return commands.WriteFailure(stderr, "RESPONSE_SHAPE_CHANGED", "X returned an unrecognized SearchTimeline response")
		}
		return commands.WriteJSON(stdout, page, commands.ExitSuccess)
	}
}

func parseArguments(arguments []string) (string, searchresult.Tab, *string, error) {
	tab := searchresult.TabTop
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
			parsed, ok := searchresult.ParseTab(value)
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
