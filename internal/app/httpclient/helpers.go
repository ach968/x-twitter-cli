package httpclient

import (
	"errors"
	"maps"
	"net/url"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func addTransactionID(request *app.PreparedRequest, transactionIDs app.TransactionIDGenerator) error {
	if transactionIDs == nil {
		return errors.New("addTransactionID requires a transaction ID generator")
	}
	parsed, err := url.Parse(request.URL)
	if err != nil {
		return err
	}
	transactionID, err := transactionIDs.Generate(request.Method, parsed.Path)
	if err != nil {
		return err
	}
	request.Headers = maps.Clone(request.Headers)
	request.Headers["x-client-transaction-id"] = transactionID
	return nil
}
