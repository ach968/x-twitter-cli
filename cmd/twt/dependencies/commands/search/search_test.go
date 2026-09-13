package search

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
	searchcommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/search"
	searchoperation "github.com/ach968/x-twitter-cli/internal/app/operations/search"
)

type controlledSearchClient struct {
	page    searchoperation.Page
	calls   int
	request searchoperation.Request
}

func (client *controlledSearchClient) Execute(_ context.Context, request searchoperation.Request) (searchoperation.Page, error) {
	client.calls++
	client.request = request
	return client.page, nil
}

func TestClientDefersAndReusesInitialization(t *testing.T) {
	client := &controlledSearchClient{}
	loads := 0
	dependency := newLazyClient(func(context.Context) (searchcommand.Requester, error) {
		loads++
		return client, nil
	})

	if loads != 0 {
		t.Fatalf("search client loaded during runtime construction")
	}
	for range 2 {
		_, err := dependency.Execute(context.Background(), searchoperation.Request{Query: "golang", Tab: searchoperation.TabLatest})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}
	if loads != 1 {
		t.Fatalf("search client loads = %d, want 1", loads)
	}
	if client.calls != 2 || client.request.Query != "golang" || client.request.Tab != searchoperation.TabLatest {
		t.Fatalf("search calls = %d, request = %#v", client.calls, client.request)
	}
}

func TestClientReusesInitializationFailure(t *testing.T) {
	want := errors.New("load search client")
	loads := 0
	dependency := newLazyClient(func(context.Context) (searchcommand.Requester, error) {
		loads++
		return nil, want
	})

	for range 2 {
		page, err := dependency.Execute(context.Background(), searchoperation.Request{Query: "golang", Tab: searchoperation.TabTop})
		if !errors.Is(err, want) || page.Query != "" || page.Results != nil {
			t.Fatalf("Execute() page = %#v, error = %v", page, err)
		}
	}
	if loads != 1 {
		t.Fatalf("search client loads = %d, want 1", loads)
	}
}

func TestLoadClientClassifiesInvalidContractState(t *testing.T) {
	t.Setenv("TWT_CONTRACT_FILE", filepath.Join(t.TempDir(), "missing-contracts.json"))
	_, err := loadClient(context.Background())
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != "CONTRACT_FAILED" || failure.RecoveryCommand != "twt contract refresh" {
		t.Fatalf("unexpected error: %v", err)
	}
}
