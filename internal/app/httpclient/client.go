package httpclient

import (
	app "github.com/ach968/x-twt-cli/internal/app"
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
