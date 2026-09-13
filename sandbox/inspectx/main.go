package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	app "github.com/ach968/x-twitter-cli3/internal/app"
	"github.com/ach968/x-twitter-cli3/internal/app/state"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
)

const defaultTimeout = 30 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		chromiumPath       = flag.String("chromium", defaultChromiumPath(), "Chromium executable")
		profilePath        = flag.String("profile", defaultProfilePath(), "x-twitter-cli3 Chromium profile")
		authenticationPath = flag.String("authentication", defaultAuthenticationPath(), "x-twitter-cli3 authentication state")
		headless           = flag.Bool("headless", false, "run Chromium without a visible window")
		timeout            = flag.Duration("timeout", defaultTimeout, "navigation timeout")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] https://x.com/...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		return errors.New("exactly one X URL is required")
	}

	targetURL, err := validateXURL(flag.Arg(0))
	if err != nil {
		return err
	}
	if info, err := os.Stat(*chromiumPath); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("Chromium executable is unavailable: %s", *chromiumPath)
	}
	if info, err := os.Stat(*profilePath); err != nil || !info.IsDir() {
		return fmt.Errorf("x-twitter-cli3 application profile does not exist: %s; authenticate it first with: make auth-browser", *profilePath)
	}
	authentication, err := state.LoadAuthentication(*authenticationPath)
	if err != nil {
		return fmt.Errorf("load x-twitter-cli3 authentication state: %w", err)
	}
	authenticationCookies, err := requiredAuthenticationCookies(authentication.Cookies)
	if err != nil {
		return err
	}

	browserLauncher := launcher.New().
		Bin(*chromiumPath).
		UserDataDir(*profilePath).
		Headless(*headless).
		Set(flags.Flag("disable-blink-features"), "AutomationControlled")
	controlURL, err := browserLauncher.Launch()
	if err != nil {
		return fmt.Errorf("launch Chromium: %w", err)
	}
	defer browserLauncher.Kill()

	browser := rod.New().ControlURL(controlURL).NoDefaultDevice()
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("connect to Chromium: %w", err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}
	page = page.Timeout(*timeout)
	if err := page.SetCookies(authenticationCookies); err != nil {
		return fmt.Errorf("apply x-twitter-cli3 authentication state: %w", err)
	}
	if err := page.Navigate(targetURL.String()); err != nil {
		return fmt.Errorf("navigate to X URL: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("wait for X page: %w", err)
	}

	cookies, err := page.Cookies([]string{"https://x.com"})
	if err != nil {
		return fmt.Errorf("inspect authentication cookies: %w", err)
	}
	authenticated := hasAuthenticationCookies(cookies)
	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("inspect page: %w", err)
	}

	fmt.Printf("URL: %s\n", info.URL)
	fmt.Printf("Title: %s\n", info.Title)
	fmt.Printf("Authenticated: %t\n", authenticated)
	fmt.Printf("DevTools: %s\n", controlURL)
	if !authenticated {
		fmt.Fprintln(os.Stderr, "warning: the application profile has no auth_token and ct0 cookies for x.com")
	}
	fmt.Println("Browser is ready for inspection; press Ctrl-C to close it.")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	<-signals
	return nil
}

func validateXURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	host := strings.ToLower(parsed.Hostname())
	if parsed.Scheme != "https" || parsed.User != nil || (host != "x.com" && !strings.HasSuffix(host, ".x.com")) {
		return nil, fmt.Errorf("URL must use HTTPS on x.com or one of its subdomains: %s", raw)
	}
	return parsed, nil
}

func hasAuthenticationCookies(cookies []*proto.NetworkCookie) bool {
	found := map[string]bool{"auth_token": false, "ct0": false}
	for _, cookie := range cookies {
		if _, ok := found[cookie.Name]; ok && cookie.Value != "" {
			found[cookie.Name] = true
		}
	}
	return found["auth_token"] && found["ct0"]
}

func requiredAuthenticationCookies(cookies []app.AuthenticationCookie) ([]*proto.NetworkCookieParam, error) {
	required := map[string]string{"auth_token": "", "ct0": ""}
	for _, cookie := range cookies {
		if _, ok := required[cookie.Name]; ok && cookie.Value != "" {
			required[cookie.Name] = cookie.Value
		}
	}
	if required["auth_token"] == "" || required["ct0"] == "" {
		return nil, errors.New("x-twitter-cli3 authentication state must contain auth_token and ct0 cookies; run make test-live to refresh it")
	}
	return []*proto.NetworkCookieParam{
		{Name: "auth_token", Value: required["auth_token"], URL: "https://x.com/", Secure: true},
		{Name: "ct0", Value: required["ct0"], URL: "https://x.com/", Secure: true},
	}, nil
}

func defaultChromiumPath() string {
	if configured := os.Getenv("TWT_CHROMIUM_EXECUTABLE"); configured != "" {
		return configured
	}
	return "/usr/bin/chromium"
}

func defaultProfilePath() string {
	stateRoot := os.Getenv("XDG_STATE_HOME")
	if stateRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		stateRoot = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateRoot, "x-twitter-cli3", "chromium-profile")
}

func defaultAuthenticationPath() string {
	configRoot := os.Getenv("XDG_CONFIG_HOME")
	if configRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configRoot = filepath.Join(home, ".config")
	}
	return filepath.Join(configRoot, "x-twitter-cli3", "authentication.json")
}
