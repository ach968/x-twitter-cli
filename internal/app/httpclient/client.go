package httpclient

import (
	"context"
	"errors"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

type Client struct {
	contracts      app.ContractProperties
	authentication app.AuthenticationState
	transport      app.XTransport
	transactionIDs app.TransactionIDGenerator
}

func New(contracts app.ContractProperties, authentication app.AuthenticationState, transport app.XTransport, transactionIDs app.TransactionIDGenerator) *Client {
	return &Client{contracts: contracts, authentication: authentication, transport: transport, transactionIDs: transactionIDs}
}

// Execute prepares and executes a captured operation contract. Callers own the
// semantic variables they override; this client owns authenticated request
// mechanics and source-level failure classification.
func (client *Client) Execute(ctx context.Context, operation app.OperationName, overrides map[string]any) (app.OperationResult, error) {
	policy, supported := app.LookupOperation(operation)
	if !supported {
		return app.OperationResult{}, errors.New("operation is unsupported")
	}
	contract, ok := client.contracts.Operations[operation]
	if !ok {
		return app.OperationResult{}, errors.New("operation contract is unavailable")
	}
	request, err := prepareRequest(contract, client.authentication, overrides)
	if err != nil {
		return app.OperationResult{}, err
	}
	if policy.TransactionID == app.Required && client.transactionIDs != nil {
		if err := addTransactionID(&request, client.transactionIDs); err != nil {
			return app.OperationResult{}, err
		}
	}
	response, err := client.transport.Execute(ctx, operation, request)
	if err != nil {
		return app.OperationResult{}, err
	}
	return resultFromResponse(operation, response), nil
}
