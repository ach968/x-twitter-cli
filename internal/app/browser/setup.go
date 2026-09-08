package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/go-rod/rod/lib/launcher"
)

const ChromiumSetupCommand = "twt setup"

type ChromiumSetupPrompt struct {
	Action             string
	RequiredRevision   string
	InstalledRevisions []string
}

type ChromiumSetupResult struct {
	Status         string
	ExecutablePath string
	Action         string
	InstallCommand string
}

type ChromiumSetupOptions struct {
	ExecutablePath  string
	ConfirmInstall  func(ChromiumSetupPrompt) (bool, error)
	ConfirmCleanup  func(ChromiumSetupPrompt) (bool, error)
	InstallChromium func() error
	RemoveRevisions func(string, []string) error
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0
}

var managedChromiumDirectory = regexp.MustCompile(`^chromium-(.+)$`)

func expectedManagedChromiumPath() string {
	return launcher.NewBrowser().BinPath()
}

func managedBrowserLocation(executablePath string) (cachePath, requiredRevision string) {
	directory := filepath.Dir(executablePath)
	for {
		match := managedChromiumDirectory.FindStringSubmatch(filepath.Base(directory))
		if len(match) == 2 {
			return filepath.Dir(directory), match[1]
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return filepath.Dir(filepath.Dir(executablePath)), "unknown"
		}
		directory = parent
	}
}

func installedManagedRevisions(cachePath string) []string {
	entries, err := os.ReadDir(cachePath)
	if err != nil {
		return nil
	}
	var revisions []string
	for _, entry := range entries {
		match := managedChromiumDirectory.FindStringSubmatch(entry.Name())
		if !entry.IsDir() || len(match) != 2 {
			continue
		}
		direct := filepath.Join(cachePath, entry.Name(), "chrome")
		nestedLayout := filepath.Join(cachePath, entry.Name(), "chrome-linux64", "chrome")
		if isExecutable(direct) || isExecutable(nestedLayout) {
			revisions = append(revisions, match[1])
		}
	}
	sort.Strings(revisions)
	return revisions
}

func installManagedChromium() error {
	_, err := launcher.NewBrowser().Get()
	return err
}

func removeManagedRevisions(cachePath string, revisions []string) error {
	for _, revision := range revisions {
		path := filepath.Join(cachePath, "chromium-"+revision)
		if filepath.Dir(path) != filepath.Clean(cachePath) {
			return fmt.Errorf("refusing to remove managed Chromium revision outside its cache")
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove managed Chromium revision %s: %w", revision, err)
		}
	}
	return nil
}

func EnsureChromiumAvailable(options ChromiumSetupOptions) (ChromiumSetupResult, error) {
	if options.ExecutablePath == "" {
		options.ExecutablePath = expectedManagedChromiumPath()
	}
	if options.ConfirmInstall == nil {
		return ChromiumSetupResult{}, fmt.Errorf("a Chromium installation confirmation callback is required")
	}
	if isExecutable(options.ExecutablePath) {
		return ChromiumSetupResult{Status: "ready", ExecutablePath: options.ExecutablePath}, nil
	}
	cachePath, requiredRevision := managedBrowserLocation(options.ExecutablePath)
	installed := installedManagedRevisions(cachePath)
	filtered := installed[:0]
	for _, revision := range installed {
		if revision != requiredRevision {
			filtered = append(filtered, revision)
		}
	}
	action := "install"
	if len(filtered) > 0 {
		action = "update"
	}
	prompt := ChromiumSetupPrompt{Action: action, RequiredRevision: requiredRevision, InstalledRevisions: filtered}
	confirmed, err := options.ConfirmInstall(prompt)
	if err != nil {
		return ChromiumSetupResult{}, err
	}
	if !confirmed {
		return ChromiumSetupResult{Status: "declined", Action: action, InstallCommand: ChromiumSetupCommand}, nil
	}
	installer := options.InstallChromium
	if installer == nil {
		installer = installManagedChromium
	}
	if err := installer(); err != nil {
		return ChromiumSetupResult{}, err
	}
	if !isExecutable(options.ExecutablePath) {
		return ChromiumSetupResult{}, fmt.Errorf("Chromium installation completed without producing the expected executable")
	}
	if action == "update" && options.ConfirmCleanup != nil {
		cleanup, err := options.ConfirmCleanup(ChromiumSetupPrompt{
			Action:             "cleanup",
			RequiredRevision:   requiredRevision,
			InstalledRevisions: filtered,
		})
		if err != nil {
			return ChromiumSetupResult{}, err
		}
		if cleanup {
			remove := options.RemoveRevisions
			if remove == nil {
				remove = removeManagedRevisions
			}
			if err := remove(cachePath, filtered); err != nil {
				return ChromiumSetupResult{}, err
			}
		}
	}
	return ChromiumSetupResult{Status: "ready", ExecutablePath: options.ExecutablePath}, nil
}
