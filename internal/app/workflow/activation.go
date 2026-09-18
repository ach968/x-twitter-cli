package workflow

import (
	"context"
	"errors"
	"fmt"
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
	for _, policy := range app.OperationPolicies() {
		if policy.Activation != app.Required {
			continue
		}
		contract, present := properties.Operations[policy.Name]
		if !present {
			// Required file entries were checked by Load.
			continue
		}
		overrides, err := validationOverrides(policy.Name)
		if err != nil {
			return ContractActivationResult{}, err
		}
		var postID string
		if policy.Name == app.TweetDetail {
			postID, _ = contract.Variables["focalTweetId"].(string)
			if postID == "" {
				return ContractActivationResult{Failure: contracts.UnavailableFailure()}, nil
			}
		}
		result, err := requestClient.Execute(ctx, policy.Name, overrides)
		if err != nil {
			return ContractActivationResult{}, err
		}
		if !result.OK {
			return ContractActivationResult{Activated: false, Failure: result.Error, Upstream: result.Upstream}, nil
		}
		if policy.Name == app.TweetDetail {
			if _, err := view.DecodePage(result.Payload, view.Request{PostID: postID}); err != nil {
				var failure *app.OperationFailure
				if errors.As(err, &failure) {
					return ContractActivationResult{Failure: failure, Upstream: result.Upstream}, nil
				}
				return ContractActivationResult{}, err
			}
		}
	}
	if err := os.Rename(candidatePath, activePath); err != nil {
		return ContractActivationResult{}, err
	}
	return ContractActivationResult{Activated: true}, nil
}

// Validation requests belong to this workflow; the catalog decides which run.
func validationOverrides(operation app.OperationName) (map[string]any, error) {
	switch operation {
	case app.SearchTimeline:
		return map[string]any{"rawQuery": "x", "product": "Top"}, nil
	case app.Bookmarks, app.TweetDetail:
		return nil, nil
	case app.BookmarkSearchTimeline:
		return map[string]any{"rawQuery": "x-twitter-cli-contract-validation-improbable-6d1e2f"}, nil
	default:
		return nil, fmt.Errorf("operation %s requires an activation check but has no validation request", operation)
	}
}
