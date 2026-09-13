package commands

import (
	"context"
	"encoding/json"
	"io"

	app "github.com/ach968/x-twitter-cli3/internal/app"
)

const (
	ExitSuccess = 0
	ExitFailure = 1
)

type Handler func(context.Context, []string, io.Reader, io.Writer, io.Writer) int

type Command struct {
	Run       Handler
	WriteHelp func(io.Writer)
}

func WriteFailure(output io.Writer, code, message string) int {
	return WriteJSON(output, app.OperationFailure{Code: code, Message: message}, ExitFailure)
}

func WriteJSON(output io.Writer, value any, status int) int {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return ExitFailure
	}
	return status
}
