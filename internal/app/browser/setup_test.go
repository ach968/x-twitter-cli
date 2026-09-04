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
