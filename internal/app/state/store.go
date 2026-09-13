package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	app "github.com/ach968/x-twitter-cli3/internal/app"
)

func SaveCaptured(candidatePath, authenticationPath string, capture app.CapturedState) error {
	directories := map[string]struct{}{
		filepath.Dir(candidatePath):      {},
		filepath.Dir(authenticationPath): {},
	}
	for directory := range directories {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return err
		}
	}
	contracts, err := json.MarshalIndent(capture.Contracts, "", "  ")
	if err != nil {
		return err
	}
	authentication, err := json.MarshalIndent(capture.Authentication, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(candidatePath, append(contracts, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(candidatePath, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(authenticationPath, append(authentication, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(authenticationPath, 0o600)
}

func LoadAuthentication(path string) (app.AuthenticationState, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return app.AuthenticationState{}, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(contents, &raw); err != nil || raw == nil {
		return app.AuthenticationState{}, errors.New("Authentication state is invalid")
	}
	var authentication app.AuthenticationState
	if _, ok := raw["authorization"]; !ok {
		return app.AuthenticationState{}, errors.New("Authentication state is invalid")
	}
	if _, ok := raw["cookies"]; !ok {
		return app.AuthenticationState{}, errors.New("Authentication state is invalid")
	}
	if err := json.Unmarshal(contents, &authentication); err != nil || authentication.Cookies == nil {
		return app.AuthenticationState{}, errors.New("Authentication state is invalid")
	}
	return authentication, nil
}
