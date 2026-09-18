// Read-only comparison of View with and without x-client-transaction-id.
// Prints transport/normalization metadata only; never posts, cursors, or credentials.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	transaction "github.com/ach968/x-client-transaction-id-go"
	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/contracts"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	"github.com/ach968/x-twitter-cli/internal/app/operations/view"
	"github.com/ach968/x-twitter-cli/internal/app/state"
)

type observedTransport struct {
	generator app.TransactionIDGenerator
	inner     app.XTransport
	header    bool
	status    int
	validJSON bool
}

func (t *observedTransport) Execute(ctx context.Context, op app.OperationName, r app.PreparedRequest) (app.UpstreamResponse, error) {
	// The experiment controls this header independently of production policy.
	if t.generator != nil {
		parsed, err := url.Parse(r.URL)
		if err != nil {
			return app.UpstreamResponse{}, err
		}
		id, err := t.generator.Generate(r.Method, parsed.Path)
		if err != nil {
			return app.UpstreamResponse{}, err
		}
		r.Headers["x-client-transaction-id"] = id
	}
	t.header = r.Headers["x-client-transaction-id"] != ""
	result, err := t.inner.Execute(ctx, op, r)
	t.status = result.Status
	t.validJSON = json.Valid([]byte(result.Body))
	return result, err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home unavailable")
	}
	environment := map[string]string{"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"), "XDG_STATE_HOME": os.Getenv("XDG_STATE_HOME"), "TWT_CONTRACT_FILE": os.Getenv("TWT_CONTRACT_FILE")}
	paths := state.ResolvePaths(home, environment)
	props, err := contracts.Load(state.ResolveContractSource("", environment, paths.ActiveContractPath))
	if err != nil {
		return fmt.Errorf("contracts unavailable")
	}
	auth, err := state.LoadAuthentication(paths.AuthenticationPath)
	if err != nil {
		return fmt.Errorf("authentication unavailable")
	}
	generator, err := transaction.New(ctx, &http.Client{Timeout: 15 * time.Second})
	if err != nil {
		return fmt.Errorf("baseline generator initialization failed")
	}
	target := "2100261836654309833"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	check := func(label string, request view.Request, withID bool) (view.Page, error) {
		observed := &observedTransport{inner: httpclient.NewTransport(&http.Client{Timeout: 15 * time.Second})}
		var ids app.TransactionIDGenerator
		if withID {
			ids = generator
		}
		observed.generator = ids
		operation := view.New(view.NewDirectSource(httpclient.New(props, auth, observed, nil)))
		page, err := operation.Execute(ctx, request)
		fmt.Printf("scenario=%s transaction_header=%t http=%d valid_json=%t decoded_page=%t partial=%t\n", label, observed.header, observed.status, observed.validJSON, err == nil, page.Partial)
		if err != nil || observed.header != withID || observed.status != 200 || !observed.validJSON {
			return view.Page{}, fmt.Errorf("comparison failed in %s", label)
		}
		return page, nil
	}
	initial, err := check("initial_with", view.Request{PostID: target}, true)
	if err != nil {
		return err
	}
	if _, err = check("initial_without", view.Request{PostID: target}, false); err != nil {
		return err
	}
	if initial.NextCursor != nil {
		request := view.Request{PostID: target, Cursor: initial.NextCursor}
		if _, err = check("continuation_with", request, true); err != nil {
			return err
		}
		if _, err = check("continuation_without", request, false); err != nil {
			return err
		}
	}
	if len(initial.Replies) > 0 {
		request := view.Request{PostID: initial.Replies[0].ID}
		if _, err = check("reply_with", request, true); err != nil {
			return err
		}
		if _, err = check("reply_without", request, false); err != nil {
			return err
		}
	}
	if _, err = check("repeat_without", view.Request{PostID: target}, false); err != nil {
		return err
	}
	if capturedID, ok := props.Operations[app.TweetDetail].Variables["focalTweetId"].(string); ok && capturedID != "" && capturedID != target {
		if _, err = check("captured_target_without", view.Request{PostID: capturedID}, false); err != nil {
			return err
		}
	}
	return nil
}
