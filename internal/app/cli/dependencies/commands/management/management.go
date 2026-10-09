package management

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	app "github.com/ach968/x-twitter-cli/internal/app"
	"github.com/ach968/x-twitter-cli/internal/app/browser"
	"github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands"
	managementservice "github.com/ach968/x-twitter-cli/internal/app/management"
	"golang.org/x/term"
)

func NewSetup(service managementservice.Service) commands.Command {
	return commands.Command{Run: runSetup(service), WriteHelp: WriteSetupHelp}
}

func NewAuth(service managementservice.Service) commands.Command {
	return commands.Command{Run: runAuth(service), WriteHelp: WriteAuthHelp}
}

func NewContract(service managementservice.Service) commands.Command {
	return commands.Command{Run: runContract(service), WriteHelp: WriteContractHelp}
}

func runSetup(service managementservice.Service) commands.Handler {
	return func(ctx context.Context, arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
		if helpRequested(arguments) {
			WriteSetupHelp(stdout)
			return commands.ExitSuccess
		}
		if len(arguments) != 0 {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "setup does not accept arguments")
		}
		if service == nil {
			return commands.WriteFailure(stderr, "SETUP_FAILED", "Unable to set up managed Chromium")
		}
		result, err := service.Setup(ctx, browserSetupConfirmation(terminalLoginPrompt(stdin, stderr)))
		if err != nil {
			return commands.WriteFailure(stderr, "SETUP_FAILED", "Unable to set up managed Chromium")
		}
		status := commands.ExitSuccess
		if result.Status != "ready" {
			status = commands.ExitFailure
		}
		return commands.WriteJSON(stdout, result, status)
	}
}

func runAuth(service managementservice.Service) commands.Handler {
	return func(ctx context.Context, arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
		if helpRequested(arguments) || subcommandHelpRequested(arguments, "login") {
			WriteAuthHelp(stdout)
			return commands.ExitSuccess
		}
		headless, valid := loginMode(arguments)
		if !valid {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "Usage: twt auth login [--headless|--headed]")
		}
		if service == nil {
			return commands.WriteFailure(stderr, "AUTHENTICATION_FAILED", "Unable to authenticate with X")
		}
		prompt := terminalLoginPrompt(stdin, stderr)
		options := managementservice.LoginOptions{Headless: headless}
		if headless {
			options.Prompt = prompt
		}
		result, err := service.Login(ctx, browserSetupConfirmation(prompt), options)
		if err != nil {
			return writeManagementFailure(stderr, err, "AUTHENTICATION_FAILED", "Unable to authenticate with X")
		}
		return commands.WriteJSON(stdout, result, commands.ExitSuccess)
	}
}

func loginMode(arguments []string) (headless, valid bool) {
	if len(arguments) == 0 || arguments[0] != "login" || len(arguments) > 2 {
		return false, false
	}
	if len(arguments) == 1 || arguments[1] == "--headed" {
		return false, true
	}
	return arguments[1] == "--headless", arguments[1] == "--headless"
}

func writeManagementFailure(output io.Writer, err error, code, message string) int {
	var failure *app.OperationFailure
	if errors.As(err, &failure) {
		switch failure.Code {
		case "AUTHENTICATION_FAILED", "AUTH_CHALLENGE_UNSUPPORTED", "AUTHENTICATION_REQUIRED", "AUTH_RATE_LIMITED", "AUTH_LOGIN_UNAVAILABLE":
			return commands.WriteFailure(output, failure.Code, failure.Message)
		}
	}
	return commands.WriteFailure(output, code, message)
}

func runContract(service managementservice.Service) commands.Handler {
	return func(ctx context.Context, arguments []string, _ io.Reader, stdout, stderr io.Writer) int {
		if helpRequested(arguments) || subcommandHelpRequested(arguments, "refresh") || subcommandHelpRequested(arguments, "status") {
			WriteContractHelp(stdout)
			return commands.ExitSuccess
		}
		if len(arguments) != 1 {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "Usage: twt contract refresh|status")
		}
		if service == nil {
			return commands.WriteFailure(stderr, "CONTRACT_COMMAND_FAILED", "Unable to inspect or refresh operation contracts")
		}
		switch arguments[0] {
		case "refresh":
			result, err := service.RefreshContracts(ctx)
			if err != nil {
				return writeManagementFailure(stderr, err, "CONTRACT_REFRESH_FAILED", "Unable to refresh operation contracts")
			}
			return commands.WriteJSON(stdout, result, commands.ExitSuccess)
		case "status":
			return commands.WriteJSON(stdout, service.ContractStatus(ctx), commands.ExitSuccess)
		default:
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "Usage: twt contract refresh|status")
		}
	}
}

func browserSetupConfirmation(prompt browser.LoginPromptFunc) managementservice.ConfirmBrowserSetup {
	return func(setup managementservice.BrowserSetupPrompt) (bool, error) {
		var label string
		if setup.Action == "cleanup" {
			label = fmt.Sprintf("New managed Chromium revision %s is ready. Remove older managed revisions %s? Warning: older twt builds may still require them. [y/N] ", setup.RequiredRevision, strings.Join(setup.InstalledRevisions, ", "))
		} else if len(setup.InstalledRevisions) > 0 {
			label = fmt.Sprintf("Managed Chromium revisions %s are installed; revision %s is required. %s now? [y/N] ", strings.Join(setup.InstalledRevisions, ", "), setup.RequiredRevision, displayAction(setup.Action))
		} else {
			label = fmt.Sprintf("Managed Chromium revision %s is required. %s now? [y/N] ", setup.RequiredRevision, displayAction(setup.Action))
		}
		answer, err := prompt(browser.LoginPrompt{Label: label})
		if err != nil && err != io.EOF {
			return false, err
		}
		return strings.EqualFold(strings.TrimSpace(answer), "y") || strings.EqualFold(strings.TrimSpace(answer), "yes"), nil
	}
}

func displayAction(action string) string {
	if action == "" {
		return "Install"
	}
	return strings.ToUpper(action[:1]) + action[1:]
}

func helpRequested(arguments []string) bool {
	return len(arguments) == 1 && (arguments[0] == "--help" || arguments[0] == "-h" || arguments[0] == "help")
}

func subcommandHelpRequested(arguments []string, subcommand string) bool {
	return len(arguments) == 2 && arguments[0] == subcommand && (arguments[1] == "--help" || arguments[1] == "-h" || arguments[1] == "help")
}

func WriteSetupHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt setup\n\nCheck for the required managed Chromium revision, offer to install or update it, and optionally remove older revisions after a successful update.\n")
}

func WriteAuthHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt auth login [--headless|--headed]\n\nLog in with the application Chromium profile, then capture and activate operation contracts. Headed login is the default and opens a visible browser window. Use --headless to enter credentials and verification codes through terminal prompts.\n")
}

func WriteContractHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt contract refresh|status\n\nRefresh operation contracts explicitly, or inspect local authentication and contract state without launching Chromium.\n")
}

func terminalLoginPrompt(input io.Reader, output io.Writer) browser.LoginPromptFunc {
	reader := bufio.NewReader(input)
	file, isFile := input.(*os.File)
	isTerminal := isFile && term.IsTerminal(int(file.Fd()))
	terminal := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{input, output}, "")
	return func(prompt browser.LoginPrompt) (string, error) {
		if isTerminal {
			state, err := term.MakeRaw(int(file.Fd()))
			if err != nil {
				return "", err
			}
			defer term.Restore(int(file.Fd()), state)
			if prompt.Secret {
				return terminal.ReadPassword(prompt.Label)
			}
			terminal.SetPrompt(prompt.Label)
			return terminal.ReadLine()
		}
		if _, err := io.WriteString(output, prompt.Label); err != nil {
			return "", err
		}
		value, err := reader.ReadString('\n')
		if err != nil && !(err == io.EOF && len(value) > 0) {
			return "", err
		}
		return strings.TrimSuffix(strings.TrimSuffix(value, "\n"), "\r"), nil
	}
}
