package browser

import (
	"errors"
	"os"
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

func acquireProfileLock(profilePath string) (func() error, error) {
	lockPath := profilePath + ".lock"
	handle, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(handle.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = handle.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, &BrowserProfileLockedError{Code: "BROWSER_PROFILE_LOCKED", ProfilePath: profilePath}
		}
		return nil, err
	}
	// Keep the lock file in place: removing it would let another process lock a
	// new inode while an existing owner still holds this one.
	if err := handle.Truncate(0); err != nil {
		_ = syscall.Flock(int(handle.Fd()), syscall.LOCK_UN)
		_ = handle.Close()
		return nil, err
	}
	return func() error {
		unlockErr := syscall.Flock(int(handle.Fd()), syscall.LOCK_UN)
		if closeErr := handle.Close(); unlockErr == nil {
			return closeErr
		}
		return unlockErr
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
