package search

import (
	"context"
	"os"
	"sync"

	transaction "github.com/ach968/x-client-transaction-id-go"
	searchcommand "github.com/ach968/x-twitter-cli3/internal/app/cli/dependencies/commands/search"
	"github.com/ach968/x-twitter-cli3/internal/app/contracts"
	"github.com/ach968/x-twitter-cli3/internal/app/httpclient"
	searchoperation "github.com/ach968/x-twitter-cli3/internal/app/operations/search"
	"github.com/ach968/x-twitter-cli3/internal/app/state"
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

func (client *lazyClient) Execute(ctx context.Context, request searchoperation.Request) (searchoperation.Page, error) {
	client.once.Do(func() {
		client.client, client.err = client.initialize(ctx)
	})
	if client.err != nil {
		return searchoperation.Page{}, client.err
	}
	return client.client.Execute(ctx, request)
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
		return nil, contracts.UnavailableFailure()
	}
	authentication, err := state.LoadAuthentication(paths.AuthenticationPath)
	if err != nil {
		return nil, err
	}
	generator, err := transaction.New(ctx, nil)
	if err != nil {
		return nil, err
	}
	requestClient := httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator)
	return searchoperation.New(searchoperation.NewDirectSource(requestClient)), nil
}
