package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

func testContracts() app.ContractProperties {
	return app.ContractProperties{Version: 1, Operations: map[app.OperationName]app.OperationContract{
		app.HomeTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/home-id/HomeTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.SearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/search-id/SearchTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.Bookmarks: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmarks-id/Bookmarks", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
		app.BookmarkSearchTimeline: {
			Family: "graphql", Host: "x.com", Path: "/i/api/graphql/bookmark-search-id/BookmarkSearchTimeline", Method: "GET", Encoding: "query",
			Variables: map[string]any{"count": float64(20)}, Features: map[string]any{}, FieldToggles: map[string]any{},
		},
	}}
}

func writeContracts(t *testing.T, path string, value any) {
	t.Helper()
	contents, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contracts.json")
	writeContracts(t, path, testContracts())
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 1 || loaded.Operations[app.HomeTimeline].Path != "/i/api/graphql/home-id/HomeTimeline" {
		t.Fatalf("unexpected contracts: %#v", loaded)
	}
}

func TestLoadAcceptsRequiredOperationsWithoutOptionalHomeTimeline(t *testing.T) {
	value := testContracts()
	delete(value.Operations, app.HomeTimeline)
	path := filepath.Join(t.TempDir(), "contracts.json")
	writeContracts(t, path, value)
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Operations) != 3 {
		t.Fatalf("unexpected contracts: %#v", loaded)
	}
}

func TestLoadSupportsViewWithoutMakingItRequiredForExistingCommands(t *testing.T) {
	props := testContracts()
	props.Operations[app.TweetDetail] = app.OperationContract{Family: "graphql", Host: "x.com", Path: "/i/api/graphql/view-id/TweetDetail", Method: "GET", Encoding: "query", Variables: map[string]any{"focalTweetId": "100"}, Features: map[string]any{}, FieldToggles: map[string]any{}}
	path := filepath.Join(t.TempDir(), "contracts.json")
	writeContracts(t, path, props)
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	delete(props.Operations, app.TweetDetail)
	writeContracts(t, path, props)
	if _, err := Load(path); err != nil {
		t.Fatalf("old contracts invalid: %v", err)
	}
}

func TestLoadRejectsInvalidProperties(t *testing.T) {
	cases := []struct {
		name string
		code string
		edit func(map[string]any)
	}{
		{"unsupported version", "UNSUPPORTED_CONTRACT_VERSION", func(value map[string]any) { value["version"] = 2 }},
		{"non-X host", "INVALID_CONTRACT_PROPERTIES", func(value map[string]any) {
			value["operations"].(map[string]any)["HomeTimeline"].(map[string]any)["host"] = "attacker.example"
		}},
		{"unsupported method", "INVALID_CONTRACT_PROPERTIES", func(value map[string]any) {
			value["operations"].(map[string]any)["SearchTimeline"].(map[string]any)["method"] = "DELETE"
		}},
		{"JSON POST without body", "INVALID_CONTRACT_PROPERTIES", func(value map[string]any) {
			home := value["operations"].(map[string]any)["HomeTimeline"].(map[string]any)
			home["method"] = "POST"
			home["encoding"] = "json"
			delete(home, "body")
		}},
		{"unexpected operation", "INVALID_CONTRACT_PROPERTIES", func(value map[string]any) {
			value["operations"].(map[string]any)["UnknownOperation"] = value["operations"].(map[string]any)["HomeTimeline"]
		}},
		{"incomplete operations", "INVALID_CONTRACT_PROPERTIES", func(value map[string]any) {
			delete(value["operations"].(map[string]any), "BookmarkSearchTimeline")
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			encoded, _ := json.Marshal(testContracts())
			var value map[string]any
			_ = json.Unmarshal(encoded, &value)
			test.edit(value)
			path := filepath.Join(t.TempDir(), "contracts.json")
			writeContracts(t, path, value)
			_, err := Load(path)
			if ErrorCode(err) != test.code {
				t.Fatalf("got %q from %v, want %q", ErrorCode(err), err, test.code)
			}
		})
	}
}

func TestLoadReportsUnreadableAndMalformedFiles(t *testing.T) {
	malformed := filepath.Join(t.TempDir(), "contracts.json")
	if err := os.WriteFile(malformed, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(malformed); ErrorCode(err) != "INVALID_CONTRACT_PROPERTIES" {
		t.Fatalf("unexpected malformed JSON error: %v", err)
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); ErrorCode(err) != "CONTRACT_PROPERTIES_UNREADABLE" {
		t.Fatalf("unexpected unreadable error: %v", err)
	}
}

func TestDecodeOperationObjectRejectsDuplicateNames(t *testing.T) {
	if _, err := decodeOperationObject([]byte(`{"Bookmarks":{},"Bookmarks":{}}`)); err == nil {
		t.Fatal("expected duplicate operation name to be rejected")
	}
}

func TestUnavailableFailureIsSafeAndActionable(t *testing.T) {
	failure := UnavailableFailure()
	if failure.Code != "CONTRACT_FAILED" || failure.Message != "Operation contracts are unavailable or invalid" || failure.RecoveryCommand != "twt contract refresh" {
		t.Fatalf("unexpected failure: %#v", failure)
	}
}
