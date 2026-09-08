package browser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func executableFor(cachePath, revision string) string {
	return filepath.Join(cachePath, "chromium-"+revision, "chrome-linux64", "chrome")
}

func TestEnsureChromiumAvailable(t *testing.T) {
	directory := t.TempDir()
	executable := executableFor(directory, "1234")
	if err := os.MkdirAll(filepath.Dir(executable), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	prompted := false
	result, err := EnsureChromiumAvailable(ChromiumSetupOptions{
		ExecutablePath: executable,
		ConfirmInstall: func(ChromiumSetupPrompt) (bool, error) { prompted = true; return false, nil },
	})
	if err != nil || result.Status != "ready" || prompted {
		t.Fatalf("available browser result: %#v %v", result, err)
	}

	missing := executableFor(t.TempDir(), "1234")
	var prompt ChromiumSetupPrompt
	result, err = EnsureChromiumAvailable(ChromiumSetupOptions{
		ExecutablePath: missing,
		ConfirmInstall: func(value ChromiumSetupPrompt) (bool, error) { prompt = value; return false, nil },
	})
	if err != nil || result.Status != "declined" || result.Action != "install" || prompt.RequiredRevision != "1234" {
		t.Fatalf("declined install result: %#v %#v %v", result, prompt, err)
	}

	updateDirectory := t.TempDir()
	missing = executableFor(updateDirectory, "1234")
	oldExecutable := executableFor(updateDirectory, "1217")
	if err := os.MkdirAll(filepath.Dir(oldExecutable), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldExecutable, []byte("browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err = EnsureChromiumAvailable(ChromiumSetupOptions{
		ExecutablePath: missing,
		ConfirmInstall: func(value ChromiumSetupPrompt) (bool, error) { prompt = value; return false, nil },
	})
	if err != nil || result.Action != "update" || !reflect.DeepEqual(prompt.InstalledRevisions, []string{"1217"}) {
		t.Fatalf("update result: %#v %#v %v", result, prompt, err)
	}

	installed := executableFor(t.TempDir(), "1234")
	result, err = EnsureChromiumAvailable(ChromiumSetupOptions{
		ExecutablePath: installed,
		ConfirmInstall: func(ChromiumSetupPrompt) (bool, error) { return true, nil },
		InstallChromium: func() error {
			if err := os.MkdirAll(filepath.Dir(installed), 0o700); err != nil {
				return err
			}
			return os.WriteFile(installed, []byte("browser"), 0o700)
		},
	})
	if err != nil || result.Status != "ready" {
		t.Fatalf("install result: %#v %v", result, err)
	}
}

func TestUpdateOffersCleanupAfterInstallingNewRevision(t *testing.T) {
	for _, test := range []struct {
		name       string
		cleanup    bool
		oldRemains bool
	}{
		{name: "accepted", cleanup: true, oldRemains: false},
		{name: "declined", cleanup: false, oldRemains: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cachePath := t.TempDir()
			newExecutable := executableFor(cachePath, "1234")
			oldExecutable := executableFor(cachePath, "1217")
			if err := os.MkdirAll(filepath.Dir(oldExecutable), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(oldExecutable, []byte("old browser"), 0o700); err != nil {
				t.Fatal(err)
			}

			cleanupPrompted := false
			result, err := EnsureChromiumAvailable(ChromiumSetupOptions{
				ExecutablePath: newExecutable,
				ConfirmInstall: func(prompt ChromiumSetupPrompt) (bool, error) {
					if prompt.Action != "update" {
						t.Fatalf("install prompt=%#v", prompt)
					}
					return true, nil
				},
				ConfirmCleanup: func(prompt ChromiumSetupPrompt) (bool, error) {
					cleanupPrompted = true
					if prompt.Action != "cleanup" || prompt.RequiredRevision != "1234" || !reflect.DeepEqual(prompt.InstalledRevisions, []string{"1217"}) {
						t.Fatalf("cleanup prompt=%#v", prompt)
					}
					return test.cleanup, nil
				},
				InstallChromium: func() error {
					if err := os.MkdirAll(filepath.Dir(newExecutable), 0o700); err != nil {
						return err
					}
					return os.WriteFile(newExecutable, []byte("new browser"), 0o700)
				},
			})
			if err != nil || result.Status != "ready" || !cleanupPrompted {
				t.Fatalf("result=%#v cleanupPrompted=%v err=%v", result, cleanupPrompted, err)
			}
			_, statErr := os.Stat(oldExecutable)
			if test.oldRemains && statErr != nil {
				t.Fatalf("old revision should remain: %v", statErr)
			}
			if !test.oldRemains && !os.IsNotExist(statErr) {
				t.Fatalf("old revision should be removed: %v", statErr)
			}
		})
	}
}

func TestDecliningUpdateDoesNotInstallOrRemoveAnything(t *testing.T) {
	cachePath := t.TempDir()
	newExecutable := executableFor(cachePath, "1234")
	oldExecutable := executableFor(cachePath, "1217")
	if err := os.MkdirAll(filepath.Dir(oldExecutable), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldExecutable, []byte("old browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	installCalled := false
	cleanupCalled := false
	result, err := EnsureChromiumAvailable(ChromiumSetupOptions{
		ExecutablePath: newExecutable,
		ConfirmInstall: func(ChromiumSetupPrompt) (bool, error) { return false, nil },
		ConfirmCleanup: func(ChromiumSetupPrompt) (bool, error) {
			cleanupCalled = true
			return true, nil
		},
		InstallChromium: func() error {
			installCalled = true
			return nil
		},
	})
	if err != nil || result.Status != "declined" || result.InstallCommand != ChromiumSetupCommand {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if installCalled || cleanupCalled {
		t.Fatalf("installCalled=%v cleanupCalled=%v", installCalled, cleanupCalled)
	}
	if _, err := os.Stat(oldExecutable); err != nil {
		t.Fatalf("old revision changed: %v", err)
	}
	if _, err := os.Stat(newExecutable); !os.IsNotExist(err) {
		t.Fatalf("new revision unexpectedly exists: %v", err)
	}
}
