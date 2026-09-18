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
	"github.com/ach968/x-twitter-cli/internal/app/operations/bookmarks"
	"github.com/ach968/x-twitter-cli/internal/app/state"
)

type observedTransport struct {
	generator app.TransactionIDGenerator
	base      app.XTransport
	status    int
	header    bool
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
	response, err := t.base.Execute(ctx, op, r)
	t.status = response.Status
	t.validJSON = json.Valid([]byte(response.Body))
	return response, err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	home, err := os.UserHomeDir()
	if err != nil {
		panic("home unavailable")
	}
	env := map[string]string{"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"), "XDG_STATE_HOME": os.Getenv("XDG_STATE_HOME"), "TWT_CONTRACT_FILE": os.Getenv("TWT_CONTRACT_FILE")}
	paths := state.ResolvePaths(home, env)
	properties, err := contracts.Load(state.ResolveContractSource("", env, paths.ActiveContractPath))
	if err != nil {
		panic("contracts unavailable")
	}
	auth, err := state.LoadAuthentication(paths.AuthenticationPath)
	if err != nil {
		panic("authentication unavailable")
	}
	generator, err := transaction.New(ctx, &http.Client{Timeout: 20 * time.Second})
	if err != nil {
		panic("generator initialization failed")
	}
	query := "x-twitter-cli-transaction-probe-improbable-6d1e2f"
	for _, search := range []bool{false, true} {
		for _, withID := range []bool{true, false, false} {
			var ids app.TransactionIDGenerator
			if withID {
				ids = generator
			}
			t := &observedTransport{generator: ids, base: httpclient.NewTransport(&http.Client{Timeout: 20 * time.Second})}
			client := bookmarks.New(bookmarks.NewDirectSource(httpclient.New(properties, auth, t, nil)))
			request := bookmarks.Request{}
			if search {
				request.Query = &query
			}
			_, err := client.Execute(ctx, request)
			fmt.Printf("search=%t transaction_header=%t status=%d valid_json=%t decoded_page=%t\n", search, t.header, t.status, t.validJSON, err == nil)
			if err != nil || t.status != 200 || !t.validJSON || t.header != withID {
				os.Exit(1)
			}
		}
	}
}
