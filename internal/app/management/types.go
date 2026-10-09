// Package management coordinates explicit browser setup, authentication, and
// operation-contract maintenance. Data commands do not call this package.
package management

import (
	"context"

	"github.com/ach968/x-twitter-cli/internal/app/browser"
)

type BrowserSetupPrompt struct {
	Action             string
	RequiredRevision   string
	InstalledRevisions []string
}

type ConfirmBrowserSetup func(BrowserSetupPrompt) (bool, error)

type LoginOptions struct {
	Headless bool
	Prompt   browser.LoginPromptFunc
}

type SetupResult struct {
	Status         string `json:"status"`
	ExecutablePath string `json:"executable_path,omitempty"`
	Action         string `json:"action,omitempty"`
	InstallCommand string `json:"install_command,omitempty"`
}

type StateChangeResult struct {
	Status             string `json:"status"`
	AuthenticationPath string `json:"authentication_path,omitempty"`
	ContractPath       string `json:"contract_path"`
}

type FileStatus struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type ContractStatus struct {
	LocallyConfigured bool       `json:"locally_configured"`
	Authentication    FileStatus `json:"authentication"`
	Contracts         FileStatus `json:"contracts"`
}

type Service interface {
	Setup(context.Context, ConfirmBrowserSetup) (SetupResult, error)
	Login(context.Context, ConfirmBrowserSetup, LoginOptions) (StateChangeResult, error)
	RefreshContracts(context.Context) (StateChangeResult, error)
	ContractStatus(context.Context) ContractStatus
}
