package main

import (
	"context"
	"os"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/cli"
	"github.com/ach968/x-twt-cli/internal/app/contracts"
	"github.com/ach968/x-twt-cli/internal/app/httpclient"
	"github.com/ach968/x-twt-cli/internal/app/state"
)

type searchRequester func(context.Context, string, map[string]any) (app.OperationResult, error)

func (request searchRequester) SearchTimeline(ctx context.Context, query string, overrides map[string]any) (app.OperationResult, error) {
	return request(ctx, query, overrides)
}

func main() {
	requester := searchRequester(func(ctx context.Context, query string, overrides map[string]any) (app.OperationResult, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return app.OperationResult{}, err
		}
		environment := map[string]string{
			"XDG_CONFIG_HOME":   os.Getenv("XDG_CONFIG_HOME"),
			"XDG_STATE_HOME":    os.Getenv("XDG_STATE_HOME"),
			"TWT_CONTRACT_FILE": os.Getenv("TWT_CONTRACT_FILE"),
		}
		paths := state.ResolvePaths(home, environment)
		contractPath := state.ResolveContractSource("", environment, paths.ActiveContractPath)
		properties, err := contracts.Load(contractPath)
		if err != nil {
			return app.OperationResult{}, err
		}
		authentication, err := state.LoadAuthentication(paths.AuthenticationPath)
		if err != nil {
			return app.OperationResult{}, err
		}
		generator, err := transaction.New(ctx, nil)
		if err != nil {
			return app.OperationResult{}, err
		}
		client := httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator)
		return client.SearchTimeline(ctx, query, overrides)
	})

	application := cli.New(cli.Dependencies{Search: requester})
	status := application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(status)
}
