package browser

import "testing"

func TestAuthenticationRequiredURLIncludesOnboarding(t *testing.T) {
	if !authenticationRequiredURL("https://x.com/i/jf/onboarding/web?redirect_after_login=%2Fhome&mode=login") {
		t.Fatal("X onboarding login was not recognized")
	}
}
