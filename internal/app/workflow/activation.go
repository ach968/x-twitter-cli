package workflow

import (
	"context"
	"errors"
	"os"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	"github.com/ach968/x-twitter-cli/internal/app/operations/view"
)

type ContractActivationResult struct {
	Activated bool
	Failure   *app.OperationFailure
	Upstream  *app.UpstreamResponse
}

func ValidateAndActivateCandidate(ctx context.Context, candidatePath, activePath string, authentication app.AuthenticationState, transport app.XTransport, transactionIDs app.TransactionIDGenerator) (ContractActivationResult, error) {
	properties, err := contracts.Load(candidatePath)
	if err != nil {
		return ContractActivationResult{}, err
	}
	requestClient := httpclient.New(properties, authentication, transport, transactionIDs)
	requests := []struct {
		operation app.OperationName
		overrides map[string]any
	}{
		{operation: app.SearchTimeline, overrides: map[string]any{"rawQuery": "x", "product": "Top"}},
		{operation: app.Bookmarks},
		{operation: app.BookmarkSearchTimeline, overrides: map[string]any{"rawQuery": "x-twitter-cli-contract-validation-improbable-6d1e2f"}},
	}
	for _, request := range requests {
		result, err := requestClient.Execute(ctx, request.operation, request.overrides)
		if err != nil {
			return ContractActivationResult{}, err
		}
		if !result.OK {
			return ContractActivationResult{Activated: false, Failure: result.Error, Upstream: result.Upstream}, nil
		}
	}
	if contract, present := properties.Operations[app.TweetDetail]; present {
		postID, ok := contract.Variables["focalTweetId"].(string)
		if !ok || postID == "" {
			return ContractActivationResult{Failure: contracts.UnavailableFailure()}, nil
		}
		_, err := view.New(view.NewDirectSource(requestClient)).Execute(ctx, view.Request{PostID: postID})
		if err != nil {
			var failure *app.OperationFailure
			if errors.As(err, &failure) {
				return ContractActivationResult{Failure: failure}, nil
			}
			return ContractActivationResult{}, err
		}
	}
	if err := os.Rename(candidatePath, activePath); err != nil {
		return ContractActivationResult{}, err
	}
	return ContractActivationResult{Activated: true}, nil
}
