package bookmarks

import (
	"context"
	"testing"

	bookmarkcommand "github.com/ach968/x-twt-cli/internal/app/cli/dependencies/commands/bookmarks"
	"github.com/ach968/x-twt-cli/internal/app/models"
	bookmarkoperation "github.com/ach968/x-twt-cli/internal/app/operations/bookmarks"
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
