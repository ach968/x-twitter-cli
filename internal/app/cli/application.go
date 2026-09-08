// Package cli owns command routing, validation, streams, rendering, and exit
// status policy for the twt executable.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/search"
)

const (
	exitSuccess = 0
	exitFailure = 1
)

type SearchRequester interface {
	SearchTimeline(context.Context, string, map[string]any) (app.OperationResult, error)
}

type Dependencies struct {
	Search SearchRequester
}

type commandHandler func(context.Context, []string, io.Reader, io.Writer, io.Writer) int

type command struct {
	run       commandHandler
	writeHelp func(io.Writer)
}

type Application struct {
	commands map[string]command
}

func New(dependencies Dependencies) *Application {
	application := &Application{commands: make(map[string]command)}
	application.commands["search"] = command{run: application.searchCommand(dependencies.Search), writeHelp: writeSearchHelp}
	return application
}

func (application *Application) Run(ctx context.Context, arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || (len(arguments) == 1 && (arguments[0] == "help" || arguments[0] == "--help" || arguments[0] == "-h")) {
		writeRootHelp(stdout)
		return exitSuccess
	}
	if len(arguments) == 2 && arguments[0] == "help" {
		if selected, ok := application.commands[arguments[1]]; ok {
			selected.writeHelp(stdout)
			return exitSuccess
		}
	}
	selected, ok := application.commands[arguments[0]]
	if !ok {
		return writeFailure(stderr, "INVALID_ARGUMENT", "Unknown command: "+arguments[0])
	}
	return selected.run(ctx, arguments[1:], stdin, stdout, stderr)
}

func (application *Application) searchCommand(requester SearchRequester) commandHandler {
	return func(ctx context.Context, arguments []string, _ io.Reader, stdout, stderr io.Writer) int {
		if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
			writeSearchHelp(stdout)
			return exitSuccess
		}
		query, tab, cursor, err := parseSearchArguments(arguments)
		if err != nil {
			return writeFailure(stderr, "INVALID_ARGUMENT", err.Error())
		}
		if requester == nil {
			return writeFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
		}
		overrides := map[string]any{"product": tab.Product()}
		if cursor != nil {
			overrides["cursor"] = *cursor
		}
		result, err := requester.SearchTimeline(ctx, query, overrides)
		if err != nil {
			return writeFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
		}
		if !result.OK {
			if result.Error == nil {
				return writeFailure(stderr, "SEARCH_FAILED", "Unable to execute search")
			}
			return writeJSON(stderr, result.Error, exitFailure)
		}
		page, err := search.DecodePage(result.Payload, query, tab)
		if err != nil {
			return writeFailure(stderr, "RESPONSE_SHAPE_CHANGED", "X returned an unrecognized SearchTimeline response")
		}
		return writeJSON(stdout, page, exitSuccess)
	}
}

func parseSearchArguments(arguments []string) (string, search.Tab, *string, error) {
	tab := search.TabTop
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
			parsed, ok := search.ParseTab(value)
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
	if len(positionals) != 1 || strings.TrimSpace(positionals[0]) == "" {
		return "", "", nil, errors.New("search requires exactly one non-empty query")
	}
	return positionals[0], tab, cursor, nil
}

func writeRootHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt <command> [options]\n\nCommands:\n  search    Search X\n\nRun 'twt search --help' for command help.\n")
}

func writeSearchHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt search <query> [--tab top|latest|people|media|lists] [--cursor value]\n")
}

func writeFailure(output io.Writer, code, message string) int {
	return writeJSON(output, app.OperationFailure{Code: code, Message: message}, exitFailure)
}

func writeJSON(output io.Writer, value any, status int) int {
	if err := json.NewEncoder(output).Encode(value); err != nil {
		return exitFailure
	}
	return status
}
