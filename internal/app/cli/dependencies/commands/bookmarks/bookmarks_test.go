package bookmarks

import (
	"bytes"
	"context"
	"errors"
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/ach968/x-twt-cli/internal/app/models"
	bookmarkoperation "github.com/ach968/x-twt-cli/internal/app/operations/bookmarks"
)

type requester struct {
	request bookmarkoperation.Request
	page    bookmarkoperation.Page
	err     error
	calls   int
}

func (r *requester) Execute(_ context.Context, request bookmarkoperation.Request) (bookmarkoperation.Page, error) {
	r.calls++
	r.request = request
	if r.err != nil {
		return bookmarkoperation.Page{}, r.err
	}
	if r.page.Bookmarks != nil {
		return r.page, nil
	}
	return bookmarkoperation.Page{Bookmarks: []models.Post{}, Warnings: []bookmarkoperation.Warning{}}, nil
}

func TestCommandWritesOneListPageToStdout(t *testing.T) {
	r := &requester{page: bookmarkoperation.Page{
		Bookmarks: []models.Post{},
		Warnings:  []bookmarkoperation.Warning{},
	}}
	var stdout, stderr bytes.Buffer

	code := New(r).Run(context.Background(), nil, nil, &stdout, &stderr)

	if code != 0 || stderr.Len() != 0 || r.calls != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout.String(), stderr.String(), r.calls)
	}
	want := "{\"query\":null,\"bookmarks\":[],\"next_cursor\":null,\"warnings\":[]}\n"
	if stdout.String() != want {
		t.Fatalf("stdout=%q, want %q", stdout.String(), want)
	}
}

func TestCommandForwardsOpaqueSearchAndCursor(t *testing.T) {
	for _, arguments := range [][]string{
		{"--search=exact query", "--cursor", "opaque"},
		{"--search", "exact query", "--cursor=opaque"},
	} {
		r := &requester{}
		var stdout, stderr bytes.Buffer
		if code := New(r).Run(context.Background(), arguments, nil, &stdout, &stderr); code != 0 {
			t.Fatalf("arguments=%q code=%d stderr=%s", arguments, code, stderr.String())
		}
		if r.request.Query == nil || *r.request.Query != "exact query" || r.request.Cursor == nil || *r.request.Cursor != "opaque" {
			t.Fatalf("arguments=%q request=%#v", arguments, r.request)
		}
	}
}

func TestCommandForwardsCursorForAllBookmarks(t *testing.T) {
	r := &requester{}
	var stdout, stderr bytes.Buffer

	code := New(r).Run(context.Background(), []string{"--cursor=opaque-list-cursor"}, nil, &stdout, &stderr)

	if code != 0 || r.request.Query != nil || r.request.Cursor == nil || *r.request.Cursor != "opaque-list-cursor" {
		t.Fatalf("code=%d request=%#v stderr=%q", code, r.request, stderr.String())
	}
}

func TestCommandRejectsMalformedFlags(t *testing.T) {
	for _, arguments := range [][]string{{"--search", " "}, {"--search"}, {"--cursor"}, {"--unknown"}, {"--search=x", "--search=y"}} {
		r := &requester{}
		var stdout, stderr bytes.Buffer
		if code := New(r).Run(context.Background(), arguments, nil, &stdout, &stderr); code == 0 || stdout.Len() != 0 || r.calls != 0 {
			t.Fatalf("arguments %q unexpectedly succeeded or invoked requester", arguments)
		}
	}
}

func TestCommandWritesOperationFailureToStderr(t *testing.T) {
	command := New(&requester{err: &app.OperationFailure{Code: "RESPONSE_SHAPE_CHANGED", Message: "safe"}})
	var stdout, stderr bytes.Buffer
	if code := command.Run(context.Background(), nil, nil, &stdout, &stderr); code == 0 || stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("RESPONSE_SHAPE_CHANGED")) {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCommandHidesUnexpectedFailureDetails(t *testing.T) {
	command := New(&requester{err: errors.New("private transport detail")})
	var stdout, stderr bytes.Buffer

	code := command.Run(context.Background(), nil, nil, &stdout, &stderr)

	if code == 0 || stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("BOOKMARKS_FAILED")) || bytes.Contains(stderr.Bytes(), []byte("private transport detail")) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestHelpDoesNotInvokeRequesterAndIncludesRecovery(t *testing.T) {
	r := &requester{}
	command := New(r)
	var stdout, stderr bytes.Buffer
	if code := command.Run(context.Background(), []string{"--help"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d", code)
	}
	if r.calls != 0 || !bytes.Contains(stdout.Bytes(), []byte("twt contract refresh")) {
		t.Fatalf("unexpected help: %q", stdout.String())
	}
}
