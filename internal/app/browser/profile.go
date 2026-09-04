package browser

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type ApplicationProfileVisit struct {
	ProfilePath string
	Headless    bool
	URL         string
}

type ApplicationProfileVisitResult struct {
	Title    string
	FinalURL string
}

type BrowserProfileLockedError struct {
	Code        string
	ProfilePath string
}

func (e *BrowserProfileLockedError) Error() string {
	return "The application profile is already in use: " + e.ProfilePath
}

func processIsRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || !(errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH))
}

func acquireProfileLock(profilePath string, mayRecover ...bool) (func() error, error) {
	recoverStale := true
	if len(mayRecover) > 0 {
		recoverStale = mayRecover[0]
	}
	lockPath := profilePath + ".lock"
	handle, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			if recoverStale {
				contents, readErr := os.ReadFile(lockPath)
				ownerPID, parseErr := strconv.Atoi(strings.TrimSpace(string(contents)))
				if readErr != nil || parseErr != nil || ownerPID <= 0 || !processIsRunning(ownerPID) {
					if removeErr := os.Remove(lockPath); removeErr != nil {
						return nil, removeErr
					}
					return acquireProfileLock(profilePath, false)
				}
			}
			return nil, &BrowserProfileLockedError{Code: "BROWSER_PROFILE_LOCKED", ProfilePath: profilePath}
		}
		return nil, err
	}
	if _, err := fmt.Fprintf(handle, "%d\n", os.Getpid()); err != nil {
		handle.Close()
		os.Remove(lockPath)
		return nil, err
	}
	return func() error {
		closeErr := handle.Close()
		removeErr := os.Remove(lockPath)
		if closeErr != nil {
			return closeErr
		}
		return removeErr
	}, nil
}

func VisitWithApplicationProfile(options ApplicationProfileVisit) (result ApplicationProfileVisitResult, err error) {
	client, err := newBrowserClient(options.ProfilePath, options.Headless, 0)
	if err != nil {
		return result, err
	}
	defer func() {
		if closeErr := client.Close(); err == nil {
			err = closeErr
		}
	}()
	if err = client.page.Navigate(options.URL); err != nil {
		return result, err
	}
	if err = client.page.WaitLoad(); err != nil {
		return result, err
	}
	info, err := client.page.Info()
	if err != nil {
		return result, err
	}
	return ApplicationProfileVisitResult{Title: info.Title, FinalURL: info.URL}, nil
}
