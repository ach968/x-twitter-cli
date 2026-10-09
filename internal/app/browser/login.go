package browser

import (
	"net/url"
	"strings"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const loginURL = "https://x.com/i/jf/onboarding/web?mode=login"

type LoginPrompt struct {
	Label  string
	Secret bool
}
type LoginPromptFunc func(LoginPrompt) (string, error)

func loginFailure(message string) error {
	return &app.OperationFailure{Code: "AUTHENTICATION_FAILED", Message: message}
}
func unsupportedLogin() error {
	return &app.OperationFailure{Code: "AUTH_CHALLENGE_UNSUPPORTED", Message: "The CLI could not handle this X login step in headless mode; try twt auth login --headed"}
}

// authenticateBrowser uses the same page and profile later used for capture.
// A nil prompt leaves input to the person interacting with the visible window.
func authenticateBrowser(client *browserClient, prompt LoginPromptFunc, startURL string) error {
	page := client.page.Timeout(10 * time.Minute)
	defer page.CancelTimeout()
	origin, err := url.Parse(startURL)
	if err != nil {
		return loginFailure("Unable to open X login")
	}
	origin.Path, origin.RawQuery, origin.Fragment = "/", "", ""
	if err := page.Navigate(origin.String() + "home"); err != nil {
		return loginFailure("Unable to open X; check connectivity and try again")
	}
	if err := page.WaitLoad(); err != nil {
		return loginFailure("Unable to load X login")
	}
	if complete, _ := browserAuthenticated(page, origin.String()); complete {
		return nil
	}
	if err := page.Navigate(startURL); err != nil {
		return loginFailure("Unable to open X login")
	}
	return completeBrowserLogin(page, prompt, origin.String())
}

func browserAuthenticated(page *rod.Page, origin string) (bool, error) {
	info, err := page.Info()
	if err != nil {
		return false, err
	}
	parsed, err := url.Parse(info.URL)
	expected, _ := url.Parse(origin)
	if err != nil || parsed.Scheme != expected.Scheme || parsed.Host != expected.Host || authenticationRequiredURL(info.URL) {
		return false, nil
	}
	cookies, err := (proto.NetworkGetCookies{Urls: []string{origin}}).Call(page)
	if err != nil {
		return false, err
	}
	for _, cookie := range cookies.Cookies {
		if cookie.Name == "auth_token" && cookie.Value != "" {
			// An expired cookie can remain while X renders its logged-out page.
			// Wait for account navigation before treating the session as valid.
			result, err := page.Eval(`() => {` + loginElementVisibility + `
 return [...document.querySelectorAll('[data-testid="SideNav_AccountSwitcher_Button"],[data-testid="AppTabBar_Profile_Link"]')].some(visible);
}`)
			if err != nil {
				// The execution context may be replaced during a redirect.
				return false, nil
			}
			return result.Value.Bool(), nil
		}
	}
	return false, nil
}

type loginScreen struct {
	Origin      string `json:"origin"`
	Signature   string `json:"signature"`
	Kind        string `json:"kind"`
	Identifier  string `json:"identifier"`
	Fields      int    `json:"fields"`
	Submit      bool   `json:"submit"`
	UsePassword bool   `json:"usePassword"`
	Failure     string `json:"failure"`
	Unsupported bool   `json:"unsupported"`
}

const loginElementVisibility = `
 const visible = e => {
   if (!e || e.closest('[inert]')) return false;
   const rect = e.getBoundingClientRect();
   if (!rect.width || !rect.height) return false;
   for (let parent=e; parent; parent=parent.parentElement) {
     const style=getComputedStyle(parent);
     if (style.visibility==='hidden' || style.display==='none' || style.opacity==='0') return false;
   }
   return true;
 };
`

// Inspect only the form shape, never return input values or raw page text.
const inspectLoginForm = `() => {` + loginElementVisibility + `
 const root = [...document.querySelectorAll('[role="dialog"],dialog')].find(visible) || document.body;
 if (!root) return {origin:location.origin,signature:location.href,fields:0};
 const textNodes = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
 const visibleText = [];
 while (textNodes.nextNode()) {
   if (visible(textNodes.currentNode.parentElement)) visibleText.push(textNodes.currentNode.textContent);
 }
 const text = visibleText.join(' ').replace(/\s+/g,' ').toLowerCase();
 const inputs = [...root.querySelectorAll('input')].filter(e => visible(e) && !e.disabled && !e.readOnly && ['text','email','tel','password','number'].includes(e.type));
 let fields = inputs.filter(e => e.type === 'password');
 let kind = fields.length ? 'password' : '';
 let identifier = '';
 if (!fields.length) {
   fields = inputs;
   const description = inputs.map(e => [e.name,e.autocomplete,e.placeholder,e.getAttribute('aria-label')].join(' ')).join(' ').toLowerCase();
   if (/one-time-code|confirmation code|verification code|authentication code|enter (the |a |your )?code|backup code/.test(description+' '+text)) kind='code';
   else if (/username|email|phone|identifier/.test(description+' '+text)) kind='identifier';
   if (kind==='identifier' && fields.length===1 && fields[0].name==='username_or_email') identifier='username_or_email';
 }
 document.querySelectorAll('[data-twt-login-field],[data-twt-login-submit],[data-twt-login-password]').forEach(e => {
   e.removeAttribute('data-twt-login-field');e.removeAttribute('data-twt-login-submit');e.removeAttribute('data-twt-login-password');
 });
 fields.forEach((e,i) => e.setAttribute('data-twt-login-field',String(i)));
 const buttons = [...root.querySelectorAll('button,[role="button"],input[type="submit"]')].filter(e => visible(e) && !e.disabled && e.getAttribute('aria-disabled') !== 'true');
 const label = e => (e.innerText || e.value || e.getAttribute('aria-label') || '').trim();
 const submit = buttons.find(e => /^(continue|next|log in|sign in|verify|confirm|submit)$/i.test(label(e)));
 const usePassword = buttons.find(e => /^(use (a |your )?password|log in with password|sign in with password)$/i.test(label(e)));
 if(submit) submit.setAttribute('data-twt-login-submit','');
 if(usePassword) usePassword.setAttribute('data-twt-login-password','');
 let failure = '';
 if (/temporarily limited|too many attempts/.test(text)) failure='rate_limited';
 else if (/incorrect password|wrong password|incorrect code|invalid code/.test(text)) failure='rejected';
 else if (/could not log you in|couldn.t log you in|something went wrong/.test(text)) failure='unavailable';
 const unsupported = !fields.length && (/captcha|security key|passkey|verify you are human|finish signing up|create your account/.test(text) || [...root.querySelectorAll('iframe')].some(e => visible(e) && /captcha|arkose/.test(e.src)));
 const headings = [...root.querySelectorAll('h1,h2,h3,[role="heading"]')].map(e => e.innerText).join('|');
 return {origin:location.origin,signature:location.href+'|'+headings+'|'+fields.map(e=>[e.name,e.type,e.autocomplete,e.placeholder].join(':')).join('|'),kind,identifier,fields:fields.length,submit:!!submit,usePassword:!!usePassword,failure,unsupported};
}`

func readLoginScreen(page *rod.Page) (loginScreen, error) {
	var screen loginScreen
	result, err := page.Eval(inspectLoginForm)
	if err != nil {
		return screen, err
	}
	err = result.Value.Unmarshal(&screen)
	return screen, err
}

func completeBrowserLogin(page *rod.Page, prompt LoginPromptFunc, origin string) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	lastSubmission := ""
	lastProgress := time.Now()
	passwordSent := false
	prompts := 0
	for {
		complete, err := browserAuthenticated(page, origin)
		if err != nil {
			return loginFailure("X login window was closed or interrupted")
		}
		if complete {
			return nil
		}
		if err := page.GetContext().Err(); err != nil {
			return loginFailure("X login was interrupted or timed out")
		}
		if prompt != nil {
			screen, err := readLoginScreen(page)
			if err != nil {
				return loginFailure("Unable to read the X login page")
			}
			if screen.Origin != strings.TrimSuffix(origin, "/") {
				return unsupportedLogin()
			}
			switch screen.Failure {
			case "rate_limited":
				return &app.OperationFailure{Code: "AUTH_RATE_LIMITED", Message: "X has temporarily limited login attempts; try again later"}
			case "unavailable":
				return &app.OperationFailure{Code: "AUTH_LOGIN_UNAVAILABLE", Message: "X displayed a login error; try again later"}
			case "rejected":
				return loginFailure("X rejected the password or verification code; check the entered value and try again")
			}
			if screen.Unsupported {
				return unsupportedLogin()
			}
			if screen.Signature != lastSubmission && screen.UsePassword {
				button, err := page.Element("[data-twt-login-password]")
				if err != nil {
					return unsupportedLogin()
				}
				if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
					return loginFailure("Unable to select password login")
				}
				lastSubmission, lastProgress = screen.Signature, time.Now()
			} else if screen.Signature != lastSubmission && screen.Fields > 0 {
				if screen.Kind == "" || (screen.Fields > 1 && screen.Kind != "code") {
					return unsupportedLogin()
				}
				if prompts >= 10 {
					return loginFailure("X login exceeded its step limit")
				}
				var question LoginPrompt
				switch screen.Kind {
				case "password":
					if passwordSent {
						return loginFailure("X requested the password again; login stopped")
					}
					question = LoginPrompt{Label: "X password: ", Secret: true}
					passwordSent = true
				case "code":
					question = LoginPrompt{Label: "X verification code: ", Secret: true}
				case "identifier":
					question = LoginPrompt{Label: "X username, email, or phone: "}
					if screen.Identifier == "username_or_email" {
						question.Label = "X username or email: "
					}
				}
				answer, err := prompt(question)
				if err != nil || answer == "" {
					return loginFailure("X login input was cancelled or empty")
				}
				if err := page.GetContext().Err(); err != nil {
					return loginFailure("X login was interrupted or timed out")
				}
				prompts++
				if err := submitLoginAnswer(page, screen, answer); err != nil {
					return err
				}
				lastSubmission, lastProgress = screen.Signature, time.Now()
			} else if time.Since(lastProgress) > 30*time.Second {
				return unsupportedLogin()
			}
		}
		select {
		case <-page.GetContext().Done():
			return loginFailure("X login was interrupted or timed out")
		case <-ticker.C:
		}
	}
}

func submitLoginAnswer(page *rod.Page, screen loginScreen, answer string) error {
	current, err := readLoginScreen(page)
	if err != nil || current.Origin != screen.Origin || current.Signature != screen.Signature {
		return loginFailure("X login form changed while waiting for input; try again")
	}
	fields, err := page.Elements("[data-twt-login-field]")
	if err != nil || len(fields) != screen.Fields {
		return loginFailure("X login form changed while waiting for input; try again")
	}
	values := []string{answer}
	if screen.Fields > 1 {
		values = nil
		for _, char := range answer {
			values = append(values, string(char))
		}
		if len(values) != len(fields) {
			return loginFailure("Verification code length does not match the X form")
		}
	}
	for i, field := range fields {
		if err := field.SelectAllText(); err != nil {
			return loginFailure("Unable to fill X login form")
		}
		if err := field.Input(values[i]); err != nil {
			return loginFailure("Unable to fill X login form")
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		updated, err := readLoginScreen(page)
		if err != nil {
			return loginFailure("Unable to read X login form")
		}
		if updated.Origin != screen.Origin {
			return unsupportedLogin()
		}
		if updated.Signature != screen.Signature {
			// Some code forms submit automatically on the final digit.
			return nil
		}
		if updated.Submit {
			button, err := page.Element("[data-twt-login-submit]")
			if err != nil {
				return loginFailure("X login form changed before submission")
			}
			if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return loginFailure("Unable to submit X login form")
			}
			return nil
		}
		select {
		case <-page.GetContext().Done():
			return loginFailure("X login was interrupted or timed out")
		case <-time.After(100 * time.Millisecond):
		}
	}
	return unsupportedLogin()
}
