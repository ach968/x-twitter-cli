package bookmarks

import (
	"context"
	"os"
	"sync"

	transaction "github.com/ach968/x-client-transaction-id-go"
	bookmarkcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/bookmarks"
	"github.com/ach968/x-twt-cli/internal/app/contracts"
	"github.com/ach968/x-twt-cli/internal/app/httpclient"
	bookmarkoperation "github.com/ach968/x-twt-cli/internal/app/operations/bookmarks"
	"github.com/ach968/x-twt-cli/internal/app/state"
)

type clientLoader func(context.Context) (bookmarkcommand.Requester, error)

type lazyClient struct {
	initialize clientLoader
	once       sync.Once
	client     bookmarkcommand.Requester
	err        error
}

func New() bookmarkcommand.Requester { return newLazyClient(loadClient) }

func newLazyClient(initialize clientLoader) *lazyClient { return &lazyClient{initialize: initialize} }

func (client *lazyClient) Execute(ctx context.Context, request bookmarkoperation.Request) (bookmarkoperation.Page, error) {
	client.once.Do(func() {
		client.client, client.err = client.initialize(ctx)
	})
	if client.err != nil {
		return bookmarkoperation.Page{}, client.err
	}
	return client.client.Execute(ctx, request)
}

func loadClient(ctx context.Context) (bookmarkcommand.Requester, error) {
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
	return bookmarkoperation.New(bookmarkoperation.NewDirectSource(httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator))), nil
}
