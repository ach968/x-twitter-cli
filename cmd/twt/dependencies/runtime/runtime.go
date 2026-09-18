// Package runtime owns deferred startup for production data commands.
package runtime

import (
	"context"
	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	"github.com/ach968/x-twitter-cli/internal/app/state"
	"os"
	"sync"
)

type Requester[Request, Page any] interface {
	Execute(context.Context, Request) (Page, error)
}

type Client[Request, Page any] struct {
	initialize func(context.Context) (Requester[Request, Page], error)
	once       sync.Once
	client     Requester[Request, Page]
	err        error
}

// New defers all local state and network work until the first Execute call.
// Operations lists the contracts this command needs, independently of other
// commands' optional contracts or runtime dependencies.
func New[Request, Page any](operations []app.OperationName, build func(*httpclient.Client) Requester[Request, Page]) *Client[Request, Page] {
	operations = append([]app.OperationName(nil), operations...)
	return &Client[Request, Page]{initialize: func(ctx context.Context) (Requester[Request, Page], error) {
		client, err := load(ctx, operations, func(ctx context.Context) (app.TransactionIDGenerator, error) { return transaction.New(ctx, nil) })
		if err != nil {
			return nil, err
		}
		return build(client), nil
	}}
}

func (client *Client[Request, Page]) Execute(ctx context.Context, request Request) (Page, error) {
	client.once.Do(func() { client.client, client.err = client.initialize(ctx) })
	if client.err != nil {
		var empty Page
		return empty, client.err
	}
	return client.client.Execute(ctx, request)
}

func load(ctx context.Context, operations []app.OperationName, newGenerator func(context.Context) (app.TransactionIDGenerator, error)) (*httpclient.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	environment := map[string]string{"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"), "XDG_STATE_HOME": os.Getenv("XDG_STATE_HOME"), "TWT_CONTRACT_FILE": os.Getenv("TWT_CONTRACT_FILE")}
	paths := state.ResolvePaths(home, environment)
	properties, err := contracts.Load(state.ResolveContractSource("", environment, paths.ActiveContractPath))
	if err != nil {
		return nil, contracts.UnavailableFailure()
	}
	needsGenerator := false
	for _, operation := range operations {
		policy, supported := app.LookupOperation(operation)
		if _, present := properties.Operations[operation]; !supported || !present {
			return nil, contracts.UnavailableFailure()
		}
		needsGenerator = needsGenerator || policy.TransactionID == app.Required
	}
	authentication, err := state.LoadAuthentication(paths.AuthenticationPath)
	if err != nil {
		return nil, err
	}
	var generator app.TransactionIDGenerator
	if needsGenerator {
		generator, err = newGenerator(ctx)
		if err != nil {
			return nil, err
		}
	}
	return httpclient.New(properties, authentication, httpclient.NewTransport(nil), generator), nil
}
