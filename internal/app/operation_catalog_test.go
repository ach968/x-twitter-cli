package app_test

import (
	app "github.com/ach968/x-twitter-cli/internal/app"
	"testing"
)

func TestOperationCatalogHasExplicitConsistentPolicies(t *testing.T) {
	seen := map[app.OperationName]bool{}
	for _, policy := range app.OperationPolicies() {
		if policy.Name == "" || seen[policy.Name] {
			t.Fatalf("empty or duplicate operation: %q", policy.Name)
		}
		seen[policy.Name] = true
		for name, value := range map[string]app.Requirement{"contract file": policy.ContractFile, "refresh": policy.Refresh, "activation": policy.Activation, "transaction ID": policy.TransactionID} {
			if value != app.Required && value != app.NotRequired {
				t.Errorf("%s has no explicit %s policy", policy.Name, name)
			}
		}
		if policy.ContractFile == app.Required && policy.Refresh != app.Required {
			t.Errorf("%s could produce an unloadable refresh", policy.Name)
		}
		if policy.Refresh == app.Required && policy.Activation != app.Required {
			t.Errorf("%s could be refreshed without validation", policy.Name)
		}
		got, ok := app.LookupOperation(policy.Name)
		if !ok || got != policy {
			t.Errorf("lookup differs from catalog for %s", policy.Name)
		}
	}
	if _, ok := app.LookupOperation("UnknownOperation"); ok {
		t.Fatal("unknown operation is supported")
	}
}

func TestOperationCatalogCannotBeChangedByCallers(t *testing.T) {
	policies := app.OperationPolicies()
	original := policies[0]
	policies[0].Name = "changed"
	got, ok := app.LookupOperation(original.Name)
	if !ok || got != original {
		t.Fatal("caller changed shared catalog")
	}
}
