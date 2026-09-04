package workflow

import (
	"context"
	"os"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/contracts"
	"github.com/ach968/x-twt-cli/internal/app/httpclient"
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
	home, err := requestClient.HomeTimeline(ctx, nil)
	if err != nil {
		return ContractActivationResult{}, err
	}
	search, err := requestClient.SearchTimeline(ctx, "x", nil)
	if err != nil {
		return ContractActivationResult{}, err
	}
	for _, result := range []app.OperationResult{home, search} {
		if !result.OK {
			return ContractActivationResult{Activated: false, Failure: result.Error, Upstream: result.Upstream}, nil
		}
	}
	if err := os.Rename(candidatePath, activePath); err != nil {
		return ContractActivationResult{}, err
	}
	return ContractActivationResult{Activated: true}, nil
}
