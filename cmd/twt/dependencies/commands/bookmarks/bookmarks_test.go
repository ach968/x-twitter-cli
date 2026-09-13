package bookmarks

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
	bookmarkcommand "github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands/bookmarks"
	"github.com/ach968/x-twitter-cli/internal/app/models"
	bookmarkoperation "github.com/ach968/x-twitter-cli/internal/app/operations/bookmarks"
)

type testRequester struct{}

func (testRequester) Execute(_ context.Context, _ bookmarkoperation.Request) (bookmarkoperation.Page, error) {
	return bookmarkoperation.Page{Bookmarks: []models.Post{}, Warnings: []bookmarkoperation.Warning{}}, nil
}

func TestLazyClientInitializesOnce(t *testing.T) {
	calls := 0
	client := newLazyClient(func(context.Context) (bookmarkcommand.Requester, error) {
		calls++
		return testRequester{}, nil
	})
	if _, err := client.Execute(context.Background(), bookmarkoperation.Request{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Execute(context.Background(), bookmarkoperation.Request{}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("loader called %d times", calls)
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
