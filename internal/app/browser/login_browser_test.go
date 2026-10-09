//go:build browser

package browser

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

const loginTestHome = `<h1>Home</h1><button data-testid="SideNav_AccountSwitcher_Button">Account</button>`

func loginTestClient(t *testing.T, profile string) *browserClient {
	t.Helper()
	client, err := newBrowserClient(profile, true, 15*time.Second)
	if err != nil {
		t.Fatalf("launch browser: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func loginTestForm(heading, input, button string) string {
	return fmt.Sprintf(`<!doctype html><html><body><h1>%s</h1><form method="post" action="/login">%s<button type="submit">%s</button></form></body></html>`, heading, input, button)
}

func loginTestCookies(w http.ResponseWriter) {
	for _, name := range []string{"auth_token", "ct0"} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "synthetic-" + name, Path: "/", MaxAge: 3600})
	}
}

func assertLoginFailure(t *testing.T, err error, code string, secrets ...string) {
	t.Helper()
	var failure *app.OperationFailure
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(err.Error(), secret) {
			t.Fatal("login error exposed a credential or page detail")
		}
	}
}

func TestBrowserLoginUsesOneBrowserForCredentialsAndSession(t *testing.T) {
	const username = "synthetic-user"
	const password = "  synthetic password\t "
	const verification = "012345"
	var mu sync.Mutex
	var submitted []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/home" {
			fmt.Fprint(w, loginTestHome)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, loginTestForm("Sign in to X", `<input name="username" autocomplete="username" aria-label="Username">`, "Continue"))
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Form.Has("username"):
			submitted = append(submitted, r.Form.Get("username"))
			fmt.Fprint(w, loginTestForm("Enter your password", `<input name="password" type="password" autocomplete="current-password">`, "Log in"))
		case r.Form.Has("password"):
			submitted = append(submitted, r.Form.Get("password"))
			fmt.Fprint(w, loginTestForm("Enter your verification code", `<input name="verification" autocomplete="one-time-code" inputmode="numeric">`, "Continue"))
		case r.Form.Has("verification"):
			submitted = append(submitted, r.Form.Get("verification"))
			loginTestCookies(w)
			http.Redirect(w, r, "/home", http.StatusSeeOther)
		}
	}))
	t.Cleanup(server.Close)
	profile := t.TempDir()
	t.Run("new session", func(t *testing.T) {
		client := loginTestClient(t, profile)
		target, pid := client.page.TargetID, client.launched.launcher.PID()
		answers := []string{username, password, verification}
		var questions []LoginPrompt
		err := authenticateBrowser(client, func(question LoginPrompt) (string, error) {
			questions = append(questions, question)
			if len(questions) > len(answers) {
				return "", errors.New("unexpected extra prompt")
			}
			return answers[len(questions)-1], nil
		}, server.URL+"/login")
		if err != nil {
			t.Fatalf("authenticate: %v", err)
		}
		if len(questions) != 3 || questions[0].Secret || !questions[1].Secret || !questions[2].Secret {
			t.Fatalf("wrong credential prompt sequence: %+v", questions)
		}
		mu.Lock()
		got := append([]string(nil), submitted...)
		mu.Unlock()
		if len(got) != len(answers) {
			t.Fatalf("expected %d submitted steps, got %d", len(answers), len(got))
		}
		for i := range answers {
			if got[i] != answers[i] {
				t.Errorf("step %d did not preserve its supplied credential", i)
			}
		}
		if client.page.TargetID != target || client.launched.launcher.PID() != pid {
			t.Fatal("login replaced the page or browser process")
		}
		info, err := client.page.Info()
		if err != nil || info.URL != server.URL+"/home" {
			t.Fatalf("expected authenticated home page, got %+v (%v)", info, err)
		}
	})
	reused := loginTestClient(t, profile)
	err := authenticateBrowser(reused, func(LoginPrompt) (string, error) {
		t.Error("persisted authenticated profile prompted for credentials")
		return "", errors.New("unexpected prompt")
	}, server.URL+"/login")
	if err != nil {
		t.Fatalf("reuse authenticated profile: %v", err)
	}
}

func TestBrowserLoginManualModeUsesSameSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/home" {
			fmt.Fprint(w, loginTestHome)
			return
		}
		if r.Method == http.MethodPost {
			loginTestCookies(w)
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
		fmt.Fprint(w, loginTestForm("Sign in", `<input name="username" autocomplete="username">`, "Continue"))
		fmt.Fprint(w, `<script>setTimeout(() => {document.querySelector('input').value='manual-user';document.querySelector('form').requestSubmit()}, 400)</script>`)
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	target, pid := client.page.TargetID, client.launched.launcher.PID()
	if err := authenticateBrowser(client, nil, server.URL+"/login"); err != nil {
		t.Fatalf("manual authentication: %v", err)
	}
	if client.page.TargetID != target || client.launched.launcher.PID() != pid {
		t.Fatal("manual login replaced the page or browser process")
	}
}

func TestBrowserLoginRejectedPasswordDoesNotReprompt(t *testing.T) {
	const password = "synthetic-rejected-password"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		heading := "Enter your password"
		if r.Method == http.MethodPost {
			heading = "Wrong password: " + password
		}
		fmt.Fprint(w, loginTestForm(heading, `<input name="password" type="password">`, "Log in"))
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	prompts := 0
	err := authenticateBrowser(client, func(LoginPrompt) (string, error) {
		prompts++
		return password, nil
	}, server.URL+"/login")
	assertLoginFailure(t, err, "AUTHENTICATION_FAILED", password)
	if prompts != 1 {
		t.Fatalf("expected one password prompt, got %d", prompts)
	}
}

func TestBrowserLoginReportsUpstreamFailuresWithoutReprompt(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		page    string
		code    string
		message string
	}{
		{
			name:    "login temporarily limited",
			page:    "We’ve temporarily limited your login. Please try again later.",
			code:    "AUTH_RATE_LIMITED",
			message: "X has temporarily limited login attempts; try again later",
		},
		{
			name:    "generic login page error",
			page:    "Something went wrong",
			code:    "AUTH_LOGIN_UNAVAILABLE",
			message: "X displayed a login error; try again later",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			const identifier = "synthetic-login-identifier"
			submitted := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if r.Method == http.MethodPost {
					submitted <- r.FormValue("username_or_email")
					fmt.Fprintf(w, "<h1>%s</h1><p>%s</p>", scenario.page, identifier)
					return
				}
				fmt.Fprint(w, loginTestForm("Sign in to X", `<input name="username_or_email" autocomplete="username">`, "Continue"))
			}))
			t.Cleanup(server.Close)
			client := loginTestClient(t, t.TempDir())
			prompts := 0
			err := authenticateBrowser(client, func(question LoginPrompt) (string, error) {
				prompts++
				if question.Secret {
					t.Error("login failure prompted for a password or code")
				}
				return identifier, nil
			}, server.URL+"/login")
			assertLoginFailure(t, err, scenario.code, identifier)
			var failure *app.OperationFailure
			if !errors.As(err, &failure) || failure.Message != scenario.message {
				t.Fatalf("expected precise static failure message, got %v", err)
			}
			if prompts != 1 || len(submitted) != 1 {
				t.Fatalf("expected one prompt and submission, got %d prompts and %d submissions", prompts, len(submitted))
			}
		})
	}
}

func TestBrowserLoginUsernameOrEmailPromptIgnoresHiddenErrors(t *testing.T) {
	const identifier = "1234567890"
	submitted := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/home" {
			fmt.Fprint(w, loginTestHome)
			return
		}
		if r.Method == http.MethodPost {
			submitted <- r.FormValue("username_or_email")
			loginTestCookies(w)
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
		fmt.Fprint(w, loginTestForm("Sign in to X", `<input name="username_or_email" autocomplete="username">`, "Continue"))
		fmt.Fprint(w, `<div style="opacity:0">Something went wrong</div><div inert>Too many attempts</div>`)
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	prompts := 0
	err := authenticateBrowser(client, func(question LoginPrompt) (string, error) {
		prompts++
		if question.Label != "X username or email: " || question.Secret {
			t.Fatalf("expected username/email prompt, got %+v", question)
		}
		return identifier, nil
	}, server.URL+"/login")
	if err != nil {
		t.Fatalf("authenticate despite hidden errors: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("expected one username/email prompt, got %d", prompts)
	}
	select {
	case got := <-submitted:
		if got != identifier {
			t.Fatal("numeric identifier was not preserved")
		}
	default:
		t.Fatal("numeric identifier was not submitted")
	}
}

func TestBrowserLoginUnsupportedChallengeStopsBeforePrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<h1>Verify you are human</h1><p>CAPTCHA required</p>")
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	err := authenticateBrowser(client, func(LoginPrompt) (string, error) {
		t.Error("unsupported challenge prompted for credentials")
		return "", nil
	}, server.URL+"/login")
	assertLoginFailure(t, err, "AUTH_CHALLENGE_UNSUPPORTED")
}

func TestBrowserLoginDoesNotExposePromptErrors(t *testing.T) {
	const secret = "synthetic-secret-from-terminal"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, loginTestForm("Enter your password", `<input name="password" type="password">`, "Log in"))
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	err := authenticateBrowser(client, func(LoginPrompt) (string, error) {
		return secret, errors.New("terminal failed with " + secret)
	}, server.URL+"/login")
	assertLoginFailure(t, err, "AUTHENTICATION_FAILED", secret)
}

func TestBrowserLoginRejectsOriginChangeWhilePrompting(t *testing.T) {
	const password = "synthetic-secret-for-trusted-origin"
	form := loginTestForm("Enter your password", `<input name="password" type="password">`, "Log in")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, form)
	}))
	t.Cleanup(server.Close)
	untrusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, form)
	}))
	t.Cleanup(untrusted.Close)
	client := loginTestClient(t, t.TempDir())
	err := authenticateBrowser(client, func(LoginPrompt) (string, error) {
		if err := client.page.Navigate(untrusted.URL + "/login"); err != nil {
			return "", err
		}
		if err := client.page.WaitLoad(); err != nil {
			return "", err
		}
		return password, nil
	}, server.URL+"/login")
	assertLoginFailure(t, err, "AUTHENTICATION_FAILED", password)
	value, err := client.page.Eval(`() => document.querySelector('input').value`)
	if err != nil {
		t.Fatalf("inspect untrusted form: %v", err)
	}
	if value.Value.Str() != "" {
		t.Fatal("credential was entered into the untrusted origin")
	}
}

func TestBrowserLoginStaleCookieDoesNotSkipPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/home" {
			if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value == "synthetic-auth_token" {
				fmt.Fprint(w, loginTestHome)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "stale-session", Path: "/"})
			fmt.Fprint(w, "<h1>Welcome to X</h1><a href='/login'>Sign in</a>")
			return
		}
		if r.Method == http.MethodPost {
			loginTestCookies(w)
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
		fmt.Fprint(w, loginTestForm("Sign in to X", `<input name="username" autocomplete="username">`, "Continue"))
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	prompts := 0
	err := authenticateBrowser(client, func(LoginPrompt) (string, error) {
		prompts++
		return "synthetic-user", nil
	}, server.URL+"/login")
	if err != nil {
		t.Fatalf("authenticate with a stale session: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("stale session should prompt once, got %d prompts", prompts)
	}
}

func TestBrowserLoginIgnoresInvisiblePasswordFields(t *testing.T) {
	for _, hidden := range []struct {
		name  string
		field string
	}{
		{name: "offscreen inert", field: `<input name="password" type="password" inert style="position:absolute;left:-10000px;top:0">`},
		{name: "transparent", field: `<input name="password" type="password" style="opacity:0">`},
		{name: "transparent ancestor", field: `<div style="opacity:0"><input name="password" type="password"></div>`},
		{name: "inert ancestor", field: `<div inert><input name="password" type="password"></div>`},
	} {
		t.Run(hidden.name, func(t *testing.T) {
			submitted := make(chan [2]string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if r.URL.Path == "/home" {
					fmt.Fprint(w, loginTestHome)
					return
				}
				if r.Method == http.MethodPost {
					submitted <- [2]string{r.FormValue("username"), r.FormValue("password")}
					loginTestCookies(w)
					http.Redirect(w, r, "/home", http.StatusSeeOther)
					return
				}
				fields := `<input name="username" autocomplete="username">` + hidden.field
				fmt.Fprint(w, loginTestForm("Sign in to X", fields, "Continue"))
			}))
			t.Cleanup(server.Close)
			client := loginTestClient(t, t.TempDir())
			prompts := 0
			err := authenticateBrowser(client, func(question LoginPrompt) (string, error) {
				prompts++
				if question.Secret {
					t.Error("invisible password input caused a secret prompt before username")
					return "", errors.New("unexpected password prompt")
				}
				return "synthetic-user", nil
			}, server.URL+"/login")
			if err != nil {
				t.Fatalf("authenticate: %v", err)
			}
			if prompts != 1 {
				t.Fatalf("expected one username prompt, got %d", prompts)
			}
			select {
			case got := <-submitted:
				if got[0] != "synthetic-user" || got[1] != "" {
					t.Fatal("username was not entered exclusively into the visible input")
				}
			default:
				t.Fatal("username form was not submitted")
			}
		})
	}
}

func TestBrowserLoginSupportsCodeAutoSubmission(t *testing.T) {
	const verification = "012345"
	submitted := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/home":
			fmt.Fprint(w, loginTestHome)
		case "/verify":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "bad code", http.StatusBadRequest)
				return
			}
			submitted <- string(body)
			loginTestCookies(w)
			fmt.Fprint(w, "verified")
		default:
			fmt.Fprint(w, `<!doctype html><h1>Enter your verification code</h1><input name="code" autocomplete="one-time-code" oninput="if(this.value.length===6){const code=this.value;document.body.innerHTML='<h1>Checking code</h1>';fetch('/verify',{method:'POST',body:code}).then(()=>setTimeout(()=>location.assign('/home'),200))}">`)
		}
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	prompts := 0
	err := authenticateBrowser(client, func(question LoginPrompt) (string, error) {
		prompts++
		if !question.Secret {
			t.Error("verification code prompt was not secret")
		}
		return verification, nil
	}, server.URL+"/login")
	if err != nil {
		t.Fatalf("authenticate with auto-submitted verification: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("expected one verification prompt, got %d", prompts)
	}
	select {
	case got := <-submitted:
		if got != verification {
			t.Fatal("auto-submitted code did not preserve the input")
		}
	default:
		t.Fatal("verification code was not automatically submitted")
	}
}

func TestBrowserLoginManualModeStopsWhenPageCloses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, loginTestForm("Sign in", `<input name="username" autocomplete="username">`, "Continue"))
	}))
	t.Cleanup(server.Close)
	client := loginTestClient(t, t.TempDir())
	page := client.page.Timeout(3 * time.Second)
	defer page.CancelTimeout()
	if err := page.Navigate(server.URL + "/login"); err != nil {
		t.Fatalf("open login page: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("load login page: %v", err)
	}
	closed := make(chan error, 1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		closed <- page.Close()
	}()
	started := time.Now()
	err := completeBrowserLogin(page, nil, server.URL+"/")
	assertLoginFailure(t, err, "AUTHENTICATION_FAILED")
	if time.Since(started) > 2500*time.Millisecond {
		t.Error("manual login waited for its timeout after the page closed")
	}
	if err := <-closed; err != nil {
		t.Fatalf("close login page: %v", err)
	}
}
