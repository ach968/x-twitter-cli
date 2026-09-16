package app

import "context"

type OperationName string

const (
	HomeTimeline           OperationName = "HomeTimeline"
	SearchTimeline         OperationName = "SearchTimeline"
	Bookmarks              OperationName = "Bookmarks"
	BookmarkSearchTimeline OperationName = "BookmarkSearchTimeline"
	TweetDetail            OperationName = "TweetDetail"
)

type OperationContract struct {
	Family       string         `json:"family"`
	Host         string         `json:"host"`
	Path         string         `json:"path"`
	Method       string         `json:"method"`
	Encoding     string         `json:"encoding"`
	Body         map[string]any `json:"body,omitempty"`
	Variables    map[string]any `json:"variables"`
	Features     map[string]any `json:"features"`
	FieldToggles map[string]any `json:"fieldToggles"`
}

type ContractProperties struct {
	Version    int                                 `json:"version"`
	Operations map[OperationName]OperationContract `json:"operations"`
}

type AuthenticationCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AuthenticationState struct {
	Cookies       []AuthenticationCookie `json:"cookies"`
	Authorization string                 `json:"authorization"`
}

type CapturedState struct {
	Contracts      ContractProperties  `json:"contracts"`
	Authentication AuthenticationState `json:"authentication"`
}

type PreparedRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

type UpstreamResponse struct {
	Status      int     `json:"status"`
	ContentType *string `json:"contentType"`
	Body        string  `json:"body"`
}

type OperationFailure struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	RecoveryCommand string `json:"recoveryCommand,omitempty"`
}

func (failure *OperationFailure) Error() string {
	if failure == nil {
		return ""
	}
	return failure.Code + ": " + failure.Message
}

type OperationResult struct {
	OK       bool              `json:"ok"`
	Payload  any               `json:"payload,omitempty"`
	Error    *OperationFailure `json:"error,omitempty"`
	Upstream *UpstreamResponse `json:"upstream,omitempty"`
}

type TransactionIDGenerator interface {
	Generate(method, path string) (string, error)
}

// XTransport is the seam between request preparation and an operation's
// execution strategy. Implementations may use direct HTTP or a browser page.
type XTransport interface {
	Execute(context.Context, OperationName, PreparedRequest) (UpstreamResponse, error)
}
