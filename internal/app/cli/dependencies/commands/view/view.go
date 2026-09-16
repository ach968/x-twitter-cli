package view

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands"
	viewoperation "github.com/ach968/x-twitter-cli/internal/app/operations/view"
)

type Requester interface {
	Execute(context.Context, viewoperation.Request) (viewoperation.Page, error)
}

func New(requester Requester) commands.Command {
	return commands.Command{WriteHelp: WriteHelp, Run: func(ctx context.Context, arguments []string, _ io.Reader, stdout, stderr io.Writer) int {
		if len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h") {
			WriteHelp(stdout)
			return commands.ExitSuccess
		}
		request, err := parseArguments(arguments)
		if err != nil {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", err.Error())
		}
		if requester == nil {
			return commands.WriteFailure(stderr, "VIEW_FAILED", "Unable to view the requested post")
		}
		page, err := requester.Execute(ctx, request)
		if err != nil {
			var failure *app.OperationFailure
			if errors.As(err, &failure) {
				return commands.WriteJSON(stderr, failure, commands.ExitFailure)
			}
			return commands.WriteFailure(stderr, "VIEW_FAILED", "Unable to view the requested post")
		}
		return commands.WriteJSON(stdout, page, commands.ExitSuccess)
	}}
}
func parseArguments(arguments []string) (viewoperation.Request, error) {
	var request viewoperation.Request
	var input *string
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		name, value, equal := strings.Cut(argument, "=")
		if name == "--cursor" {
			if request.Cursor != nil {
				return request, errors.New("--cursor may be supplied once")
			}
			if !equal {
				index++
				if index >= len(arguments) {
					return request, errors.New("--cursor requires a value")
				}
				value = arguments[index]
			}
			if strings.TrimSpace(value) == "" {
				return request, errors.New("--cursor requires a non-empty value")
			}
			request.Cursor = &value
		} else {
			if strings.HasPrefix(argument, "-") {
				return request, errors.New("unknown view flag")
			}
			if input != nil {
				return request, errors.New("view requires one post URL or ID")
			}
			input = &argument
		}
	}
	if input == nil {
		return request, errors.New("view requires one post URL or ID")
	}
	var err error
	request.PostID, err = parsePostID(*input)
	return request, err
}

func validDecimal(value string) bool {
	if value == "" || value[0] == '0' {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	_, err := strconv.ParseUint(value, 10, 64)
	return err == nil
}
func parsePostID(value string) (string, error) {
	invalid := errors.New("view requires a positive post ID or a supported X/Twitter status URL")
	if validDecimal(value) {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", invalid
	}
	switch strings.ToLower(parsed.Host) {
	case "x.com", "www.x.com", "mobile.x.com", "twitter.com", "www.twitter.com", "mobile.twitter.com":
	default:
		return "", invalid
	}
	if strings.Contains(parsed.EscapedPath(), "%") {
		return "", invalid
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/"), "/"), "/")
	if len(parts) >= 2 && (parts[len(parts)-2] == "photo" || parts[len(parts)-2] == "video") {
		if !validDecimal(parts[len(parts)-1]) {
			return "", invalid
		}
		parts = parts[:len(parts)-2]
	}
	id := ""
	if len(parts) == 3 && parts[0] != "" && parts[1] == "status" {
		id = parts[2]
	}
	if len(parts) == 4 && parts[0] == "i" && parts[1] == "web" && parts[2] == "status" {
		id = parts[3]
	}
	if !validDecimal(id) {
		return "", invalid
	}
	return id, nil
}
func WriteHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt view <url-or-id> [--cursor value]\n\nView one conversation page using X's order and pagination. Repeat the same post with next_cursor to read more replies. Later pages may omit the requested post and ancestors. Partial and warnings report known page gaps. If a contract is missing or stale, run `twt contract refresh`.\n")
}
