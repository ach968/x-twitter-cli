package view

import (
	"context"
	"os"
	"sync"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twitter-cli/internal/app"
	viewcommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/view"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	viewoperation "github.com/ach968/x-twitter-cli/internal/app/operations/view"
	"github.com/ach968/x-twitter-cli/internal/app/state"
)

type clientLoader func(context.Context) (viewcommand.Requester, error)

type lazyClient struct {
	initialize clientLoader
	once       sync.Once
	client     viewcommand.Requester
	err        error
}

func New() viewcommand.Requester { return newLazyClient(loadClient) }

func newLazyClient(initialize clientLoader) *lazyClient { return &lazyClient{initialize: initialize} }

func (client *lazyClient) Execute(ctx context.Context, request viewoperation.Request) (viewoperation.Page, error) {
	client.once.Do(func() {
		client.client, client.err = client.initialize(ctx)
	})
	if client.err != nil {
		return viewoperation.Page{}, client.err
	}
	return client.client.Execute(ctx, request)
}

func loadClient(ctx context.Context) (viewcommand.Requester, error) {
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
	properties, err := contracts.Load(state.ResolveContractSource("", environment, paths.ActiveContractPath))
	if err != nil {
		return nil, contracts.UnavailableFailure()
	}
	if _, present := properties.Operations[app.TweetDetail]; !present {
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
	return viewoperation.New(viewoperation.NewDirectSource(httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator))), nil
}
