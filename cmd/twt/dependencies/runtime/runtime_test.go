package runtime

import (
	"context"
	"errors"
	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/httpclient"
	"github.com/ach968/x-twitter-cli/internal/app/state"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type echo struct{}

func (echo) Execute(_ context.Context, request string) (string, error) { return request, nil }

func TestDeferredInitializationIsSharedAcrossConcurrentCalls(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			calls := 0
			wantErr := errors.New("startup failed")
			client := &Client[string, string]{initialize: func(context.Context) (Requester[string, string], error) {
				calls++
				if fail {
					return nil, wantErr
				}
				return echo{}, nil
			}}
			if calls != 0 {
				t.Fatal("initialized eagerly")
			}
			var group sync.WaitGroup
			for range 8 {
				group.Go(func() {
					got, err := client.Execute(context.Background(), "request")
					if fail {
						if got != "" || !errors.Is(err, wantErr) {
							t.Errorf("got=%q err=%v", got, err)
						}
					} else if got != "request" || err != nil {
						t.Errorf("got=%q err=%v", got, err)
					}
				})
			}
			group.Wait()
			if calls != 1 {
				t.Fatalf("initializations=%d", calls)
			}
		})
	}
}

func savedState(t *testing.T) state.StatePaths {
	t.Helper()
	directory := t.TempDir()
	environment := map[string]string{"XDG_CONFIG_HOME": filepath.Join(directory, "config"), "XDG_STATE_HOME": filepath.Join(directory, "state")}
	paths := state.ResolvePaths(directory, environment)
	for key, value := range environment {
		t.Setenv(key, value)
	}
	t.Setenv("TWT_CONTRACT_FILE", paths.ActiveContractPath)
	properties := app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{}}
	for _, policy := range app.OperationPolicies() {
		properties.Operations[policy.Name] = app.OperationContract{Family: "graphql", Host: "x.com", Path: "/i/api/graphql/test/" + string(policy.Name), Method: "GET", Encoding: "query", Variables: map[string]any{}, Features: map[string]any{}, FieldToggles: map[string]any{}}
	}
	err := state.SaveCaptured(paths.ActiveContractPath, paths.AuthenticationPath, app.CapturedState{Contracts: properties, Authentication: app.AuthenticationState{Authorization: "synthetic", Cookies: []app.AuthenticationCookie{}}})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestOnlyRequiredOperationsInitializeGenerator(t *testing.T) {
	savedState(t)
	sentinel := errors.New("generator initialization")
	for _, test := range []struct {
		name       string
		operations []app.OperationName
		wantCalls  int
	}{
		{"search", []app.OperationName{app.SearchTimeline}, 1},
		{"bookmarks", []app.OperationName{app.Bookmarks, app.BookmarkSearchTimeline}, 0},
		{"view", []app.OperationName{app.TweetDetail}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client, err := load(context.Background(), test.operations, func(context.Context) (app.TransactionIDGenerator, error) { calls++; return nil, sentinel })
			if calls != test.wantCalls {
				t.Fatalf("generator calls=%d", calls)
			}
			if test.wantCalls == 1 {
				if !errors.Is(err, sentinel) {
					t.Fatalf("err=%v", err)
				}
			} else if err != nil || client == nil {
				t.Fatalf("client=%v err=%v", client, err)
			}
		})
	}
}

func TestNewDefersStateLoadingAndReusesBuiltClient(t *testing.T) {
	paths := savedState(t)
	builds := 0
	client := New([]app.OperationName{app.TweetDetail}, func(*httpclient.Client) Requester[string, string] { builds++; return echo{} })
	if builds != 0 {
		t.Fatal("built eagerly")
	}
	if _, err := client.Execute(context.Background(), "first"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(paths.AuthenticationPath); err != nil {
		t.Fatal(err)
	}
	if got, err := client.Execute(context.Background(), "second"); err != nil || got != "second" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if builds != 1 {
		t.Fatalf("builds=%d", builds)
	}
}

func TestStartupErrorsPrecedeGeneratorAndOperationConstruction(t *testing.T) {
	for _, stage := range []string{"contracts", "authentication"} {
		t.Run(stage, func(t *testing.T) {
			paths := savedState(t)
			missing := paths.AuthenticationPath
			if stage == "contracts" {
				missing = paths.ActiveContractPath
			}
			if err := os.Remove(missing); err != nil {
				t.Fatal(err)
			}
			_, err := load(context.Background(), []app.OperationName{app.SearchTimeline}, func(context.Context) (app.TransactionIDGenerator, error) {
				t.Fatal("generator initialized before valid state")
				return nil, nil
			})
			if err == nil {
				t.Fatal("missing state accepted")
			}
			client := New([]app.OperationName{app.SearchTimeline}, func(*httpclient.Client) Requester[string, string] { t.Fatal("built with invalid state"); return echo{} })
			if _, err := client.Execute(context.Background(), ""); err == nil {
				t.Fatal("startup error lost")
			}
		})
	}
}
