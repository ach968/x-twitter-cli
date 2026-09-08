package search

import (
	"context"
	"os"
	"sync"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twt-cli/internal/app"
	searchcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/search"
	"github.com/ach968/x-twt-cli/internal/app/contracts"
	"github.com/ach968/x-twt-cli/internal/app/httpclient"
	"github.com/ach968/x-twt-cli/internal/app/state"
)

type clientLoader func(context.Context) (searchcommand.Requester, error)

type lazyClient struct {
	initialize clientLoader
	once       sync.Once
	client     searchcommand.Requester
	err        error
}

func New() searchcommand.Requester {
	return newLazyClient(loadClient)
}

func newLazyClient(initialize clientLoader) *lazyClient {
	return &lazyClient{initialize: initialize}
}

func (client *lazyClient) SearchTimeline(ctx context.Context, query string, overrides map[string]any) (app.OperationResult, error) {
	client.once.Do(func() {
		client.client, client.err = client.initialize(ctx)
	})
	if client.err != nil {
		return app.OperationResult{}, client.err
	}
	return client.client.SearchTimeline(ctx, query, overrides)
}

func loadClient(ctx context.Context) (searchcommand.Requester, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	authentication, err := state.LoadAuthentication(paths.AuthenticationPath)
	if err != nil {
		return nil, err
	}
	generator, err := transaction.New(ctx, nil)
	if err != nil {
		return nil, err
	}
	return httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator), nil
}
