package browser

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestProfileLockIgnoresStalePIDAndBlocksActiveOwner(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	lockPath := profile + ".lock"
	// A leftover lock can name a PID that has since been reused by another process.
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	release, err := acquireProfileLock(profile)
	if err != nil {
		t.Fatalf("stale PID blocked profile: %v", err)
	}
	if _, err := acquireProfileLock(profile); err == nil {
		t.Fatal("concurrent profile owner was accepted")
	} else {
		var locked *BrowserProfileLockedError
		if !errors.As(err, &locked) {
			t.Fatalf("wrong concurrent-owner error: %v", err)
		}
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	release, err = acquireProfileLock(profile)
	if err != nil {
		t.Fatalf("released profile stayed locked: %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
}
