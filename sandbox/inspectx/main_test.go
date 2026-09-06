package main

import (
	"testing"

	app "github.com/ach968/x-twt-cli/internal/app"
	"github.com/go-rod/rod/lib/proto"
)

func TestValidateXURL(t *testing.T) {
	valid := []string{
		"https://x.com",
		"https://x.com/example/status/123",
		"https://mobile.x.com/example",
	}
	for _, raw := range valid {
		if _, err := validateXURL(raw); err != nil {
			t.Errorf("validateXURL(%q): %v", raw, err)
		}
	}

	invalid := []string{
		"http://x.com/example",
		"https://x.com.example/",
		"https://example.com/",
		"https://user@x.com/",
	}
	for _, raw := range invalid {
		if _, err := validateXURL(raw); err == nil {
			t.Errorf("validateXURL(%q) succeeded", raw)
		}
	}
}

func TestHasAuthenticationCookies(t *testing.T) {
	if hasAuthenticationCookies([]*proto.NetworkCookie{{Name: "auth_token", Value: "secret"}}) {
		t.Fatal("auth_token alone should not be authenticated")
	}
	if !hasAuthenticationCookies([]*proto.NetworkCookie{
		{Name: "auth_token", Value: "secret"},
		{Name: "ct0", Value: "secret"},
	}) {
		t.Fatal("auth_token and ct0 should be authenticated")
	}
}

func TestRequiredAuthenticationCookies(t *testing.T) {
	cookies, err := requiredAuthenticationCookies([]app.AuthenticationCookie{
		{Name: "guest_id", Value: "not-forwarded"},
		{Name: "auth_token", Value: "auth-secret"},
		{Name: "ct0", Value: "csrf-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 2 || cookies[0].Name != "auth_token" || cookies[1].Name != "ct0" {
		t.Fatalf("unexpected authentication cookies: %#v", cookies)
	}

	if _, err := requiredAuthenticationCookies([]app.AuthenticationCookie{{Name: "auth_token", Value: "auth-secret"}}); err == nil {
		t.Fatal("missing ct0 should fail")
	}
}
