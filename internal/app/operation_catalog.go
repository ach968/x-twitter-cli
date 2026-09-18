package app

// Requirement has no implicit default: every catalog entry must choose a policy.
type Requirement string

const (
	Required    Requirement = "required"
	NotRequired Requirement = "not-required"
)

// OperationPolicy contains code-owned lifecycle rules, not captured request data.
// Catalog membership means the operation is supported for loading and capture.
type OperationPolicy struct {
	Name         OperationName
	ContractFile Requirement
	Refresh      Requirement
	// Activation requires a validation request when this contract is present.
	Activation    Requirement
	TransactionID Requirement
}

// OperationPolicies returns an independent copy in activation order.
func OperationPolicies() []OperationPolicy {
	return []OperationPolicy{
		{Name: HomeTimeline, ContractFile: NotRequired, Refresh: NotRequired, Activation: NotRequired, TransactionID: NotRequired},
		{Name: SearchTimeline, ContractFile: Required, Refresh: Required, Activation: Required, TransactionID: Required},
		{Name: Bookmarks, ContractFile: Required, Refresh: Required, Activation: Required, TransactionID: NotRequired},
		{Name: BookmarkSearchTimeline, ContractFile: Required, Refresh: Required, Activation: Required, TransactionID: NotRequired},
		{Name: TweetDetail, ContractFile: NotRequired, Refresh: Required, Activation: Required, TransactionID: NotRequired},
	}
}

// LookupOperation returns false for names not explicitly supported by this build.
func LookupOperation(name OperationName) (OperationPolicy, bool) {
	for _, policy := range OperationPolicies() {
		if policy.Name == name {
			return policy, true
		}
	}
	return OperationPolicy{}, false
}
