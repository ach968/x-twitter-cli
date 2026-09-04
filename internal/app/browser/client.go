package browser

import (
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
)

type launchedBrowser struct {
	launcher *launcher.Launcher
	browser  *rod.Browser
}

func launchApplicationBrowser(profilePath string, headless bool) (*launchedBrowser, error) {
	executablePath := os.Getenv("TWT_CHROMIUM_EXECUTABLE")
	if executablePath == "" {
		executablePath = expectedManagedChromiumPath()
	}
	if !isExecutable(executablePath) {
		return nil, fmt.Errorf("managed Chromium is unavailable; run %s", ChromiumSetupCommand)
	}
	l := launcher.New().
		Bin(executablePath).
		UserDataDir(profilePath).
		Headless(headless).
		Set(flags.Flag("disable-blink-features"), "AutomationControlled")
	controlURL, err := l.Launch()
	if err != nil {
		return nil, err
	}
	browser := rod.New().ControlURL(controlURL).NoDefaultDevice()
	if err := browser.Connect(); err != nil {
		l.Kill()
		return nil, err
	}
	return &launchedBrowser{launcher: l, browser: browser}, nil
}

func (launched *launchedBrowser) Close() error {
	err := launched.browser.Close()
	deadline := time.Now().Add(5 * time.Second)
	for processIsRunning(launched.launcher.PID()) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processIsRunning(launched.launcher.PID()) {
		launched.launcher.Kill()
	}
	return err
}

type browserClient struct {
	page     *rod.Page
	launched *launchedBrowser
	release  func() error
}

func newBrowserClient(profilePath string, headless bool, timeout time.Duration) (*browserClient, error) {
	if err := os.MkdirAll(profilePath, 0o700); err != nil {
		return nil, err
	}
	release, err := acquireProfileLock(profilePath)
	if err != nil {
		return nil, err
	}
	launched, err := launchApplicationBrowser(profilePath, headless)
	if err != nil {
		_ = release()
		return nil, err
	}
	page, err := launched.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		_ = launched.Close()
		_ = release()
		return nil, err
	}
	if timeout > 0 {
		page = page.Timeout(timeout)
	}
	return &browserClient{page: page, launched: launched, release: release}, nil
}

func (client *browserClient) Close() error {
	closeErr := client.launched.Close()
	releaseErr := client.release()
	if closeErr != nil {
		return closeErr
	}
	return releaseErr
}
