package management

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ach968/x-twitter-cli/internal/app/cli/dependencies/commands"
	managementservice "github.com/ach968/x-twitter-cli/internal/app/management"
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
		result, err := service.Setup(ctx, browserSetupConfirmation(stdin, stderr))
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
		if len(arguments) != 1 || arguments[0] != "login" {
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "Usage: twt auth login")
		}
		if service == nil {
			return commands.WriteFailure(stderr, "AUTHENTICATION_FAILED", "Unable to authenticate with X")
		}
		result, err := service.Login(ctx, browserSetupConfirmation(stdin, stderr))
		if err != nil {
			return commands.WriteFailure(stderr, "AUTHENTICATION_FAILED", "Unable to authenticate with X")
		}
		return commands.WriteJSON(stdout, result, commands.ExitSuccess)
	}
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
				return commands.WriteFailure(stderr, "CONTRACT_REFRESH_FAILED", "Unable to refresh operation contracts")
			}
			return commands.WriteJSON(stdout, result, commands.ExitSuccess)
		case "status":
			return commands.WriteJSON(stdout, service.ContractStatus(ctx), commands.ExitSuccess)
		default:
			return commands.WriteFailure(stderr, "INVALID_ARGUMENT", "Usage: twt contract refresh|status")
		}
	}
}

func browserSetupConfirmation(input io.Reader, output io.Writer) managementservice.ConfirmBrowserSetup {
	reader := bufio.NewReader(input)
	return func(prompt managementservice.BrowserSetupPrompt) (bool, error) {
		if prompt.Action == "cleanup" {
			_, _ = fmt.Fprintf(output, "New managed Chromium revision %s is ready. Remove older managed revisions %s? Warning: older twt builds may still require them. [y/N] ", prompt.RequiredRevision, strings.Join(prompt.InstalledRevisions, ", "))
			return readConfirmation(reader)
		}
		action := prompt.Action
		if action == "" {
			action = "install"
		}
		if len(prompt.InstalledRevisions) > 0 {
			_, _ = fmt.Fprintf(output, "Managed Chromium revisions %s are installed; revision %s is required. %s now? [y/N] ", strings.Join(prompt.InstalledRevisions, ", "), prompt.RequiredRevision, displayAction(action))
		} else {
			_, _ = fmt.Fprintf(output, "Managed Chromium revision %s is required. %s now? [y/N] ", prompt.RequiredRevision, displayAction(action))
		}
		return readConfirmation(reader)
	}
}

func readConfirmation(reader *bufio.Reader) (bool, error) {
	answer, err := reader.ReadString('\n')
	if err != nil && len(answer) == 0 && err != io.EOF {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(answer), "y") || strings.EqualFold(strings.TrimSpace(answer), "yes"), nil
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
	_, _ = io.WriteString(output, "Usage: twt auth login\n\nCheck the application profile headlessly, open it headed only when interactive X login is required, then capture and activate operation contracts.\n")
}

func WriteContractHelp(output io.Writer) {
	_, _ = io.WriteString(output, "Usage: twt contract refresh|status\n\nRefresh operation contracts explicitly, or inspect local authentication and contract state without launching Chromium.\n")
}
