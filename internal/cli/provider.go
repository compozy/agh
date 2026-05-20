package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/providerauth"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/vault"
	"github.com/spf13/cobra"
)

const (
	providerMessageValue  = "Message"
	providerNameValue     = "Name"
	providerProviderValue = "Provider"
	providerStateValue    = "State"
	providerAuthKey       = "auth"
	providerProviderKey   = "provider"
	providerStateKey      = "state"
	providerVaultKey      = "vault"
)

const (
	defaultProviderAuthCommandTimeout = 30 * time.Second
	providerAuthStateAuthenticated    = "authenticated"
	providerAuthStateMissingRequired  = "missing_required"
	providerAuthStateMissingCLI       = "missing_cli"
	providerAuthStateNeedsLogin       = "needs_login"
	providerAuthStateNativeCLI        = "native_cli"
	statusStateNone                   = "none"
)

type providerAuthCommandRunner func(
	context.Context,
	providerAuthCommandSpec,
) (providerAuthCommandResult, error)

type providerAuthCommandSpec struct {
	Command string
	Env     []string
	Timeout time.Duration
}

type providerAuthCommandResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

type providerAuthStatusRecord struct {
	Provider      string                         `json:"provider"`
	DisplayName   string                         `json:"display_name,omitempty"`
	AuthMode      string                         `json:"auth_mode"`
	EnvPolicy     string                         `json:"env_policy"`
	HomePolicy    string                         `json:"home_policy"`
	State         string                         `json:"state"`
	Message       string                         `json:"message,omitempty"`
	StatusCommand string                         `json:"status_command,omitempty"`
	LoginCommand  string                         `json:"login_command,omitempty"`
	LoginEnv      []string                       `json:"login_env,omitempty"`
	NativeCLI     *providerNativeCLIStatusRecord `json:"native_cli,omitempty"`
	Credentials   []providerCredentialStatusItem `json:"credentials,omitempty"`
	Probe         *providerAuthCommandResult     `json:"probe,omitempty"`
}

type providerNativeCLIStatusRecord = providerauth.NativeCLIStatus

type providerCredentialStatusItem struct {
	Name      string `json:"name"`
	TargetEnv string `json:"target_env"`
	SecretRef string `json:"secret_ref"`
	Kind      string `json:"kind,omitempty"`
	Required  bool   `json:"required"`
	Present   bool   `json:"present"`
	Source    string `json:"source,omitempty"`
}

func (d commandDeps) withProviderAuthDefaults() commandDeps {
	if d.runProviderAuthCommand == nil {
		d.runProviderAuthCommand = defaultProviderAuthCommandRunner
	}
	return d
}

func newProviderCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   providerProviderKey,
		Short: "Inspect and manage provider authentication",
	}
	cmd.AddCommand(newProviderAuthCommand(deps))
	cmd.AddCommand(newProviderModelsCommand(deps))
	return cmd
}

func newProviderAuthCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   providerAuthKey,
		Short: "Inspect native CLI and bound-secret provider authentication",
	}
	cmd.AddCommand(newProviderAuthStatusCommand(deps))
	cmd.AddCommand(newProviderAuthLoginCommand(deps))
	return cmd
}

func newProviderAuthStatusCommand(deps commandDeps) *cobra.Command {
	var noProbe bool
	cmd := &cobra.Command{
		Use:   "status <provider>",
		Short: "Show provider authentication status",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := providerAuthRuntime(deps)
			if err != nil {
				return err
			}
			record, err := buildProviderAuthStatus(cmd.Context(), deps, runtime, args[0], !noProbe)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, providerAuthStatusBundle(record))
		},
	}
	cmd.Flags().BoolVar(&noProbe, "no-probe", false, "Do not run the provider status command")
	return cmd
}

func newProviderAuthLoginCommand(deps commandDeps) *cobra.Command {
	var printCommand bool
	cmd := &cobra.Command{
		Use:   "login <provider>",
		Short: "Print the provider native login command for operator execution",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := providerAuthRuntime(deps)
			if err != nil {
				return err
			}
			providerName, provider, err := resolveProviderAuthTarget(&runtime.Config, args[0])
			if err != nil {
				return err
			}
			loginCommand := strings.TrimSpace(provider.AuthLoginCmd)
			if loginCommand == "" {
				return providerMissingAuthLoginCommandError(providerName, provider)
			}
			if provider.EffectiveAuthMode() != aghconfig.ProviderAuthModeNativeCLI {
				return fmt.Errorf(
					"cli: provider %q uses auth_mode %q; provider auth login only exposes native_cli login commands",
					providerName,
					provider.EffectiveAuthMode(),
				)
			}
			var nativeCLI *providerNativeCLIStatusRecord
			nativeCLI, err = providerNativeCLIStatusForCommand(
				loginCommand,
				providerauth.NativeCLISourceAuthLogin,
				deps.lookPath,
			)
			if err != nil {
				return err
			}
			if nativeCLI != nil && !nativeCLI.Present {
				return fmt.Errorf(
					"cli: provider %q native CLI %q was not found on PATH; install it before running %q",
					providerName,
					nativeCLI.Command,
					loginCommand,
				)
			}
			loginEnv, err := providerNativeCLILoginEnv(runtime.HomePaths, providerName, provider)
			if err != nil {
				return err
			}
			operatorCommand, err := providerOperatorLoginCommand(loginCommand, loginEnv)
			if err != nil {
				return err
			}
			if printCommand {
				if err := rejectPrintCommandOutputFormat(cmd); err != nil {
					return err
				}
				return writeRawCommandOutput(cmd, operatorCommand+"\n")
			}
			record := providerAuthStatusRecord{
				Provider:      providerName,
				DisplayName:   strings.TrimSpace(provider.DisplayName),
				AuthMode:      string(provider.EffectiveAuthMode()),
				EnvPolicy:     string(provider.EffectiveEnvPolicy()),
				HomePolicy:    string(provider.EffectiveHomePolicy()),
				State:         providerAuthStateNativeCLI,
				Message:       providerNativeCLILoginCommandMessage(providerName, operatorCommand),
				StatusCommand: strings.TrimSpace(provider.AuthStatusCmd),
				LoginCommand:  loginCommand,
				LoginEnv:      loginEnv,
				NativeCLI:     nativeCLI,
			}
			return writeCommandOutput(cmd, providerAuthStatusBundle(record))
		},
	}
	cmd.Flags().
		BoolVar(&printCommand, "print-command", false, "Print only the resolved provider login command")
	return cmd
}

func providerAuthRuntime(deps commandDeps) (*runtimeContext, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, err
	}
	if err := deps.ensureHome(runtime.HomePaths); err != nil {
		return nil, err
	}
	return runtime, nil
}

func buildProviderAuthStatus(
	ctx context.Context,
	deps commandDeps,
	runtime *runtimeContext,
	providerRef string,
	probe bool,
) (providerAuthStatusRecord, error) {
	if runtime == nil {
		return providerAuthStatusRecord{}, errors.New("cli: provider auth runtime is required")
	}
	providerName, provider, err := resolveProviderAuthTarget(&runtime.Config, providerRef)
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	credentials, err := providerCredentialStatuses(ctx, runtime.HomePaths, provider, deps.getenv)
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	record := providerAuthStatusRecord{
		Provider:      providerName,
		DisplayName:   strings.TrimSpace(provider.DisplayName),
		AuthMode:      string(provider.EffectiveAuthMode()),
		EnvPolicy:     string(provider.EffectiveEnvPolicy()),
		HomePolicy:    string(provider.EffectiveHomePolicy()),
		StatusCommand: strings.TrimSpace(provider.AuthStatusCmd),
		LoginCommand:  strings.TrimSpace(provider.AuthLoginCmd),
		Credentials:   credentials,
	}
	record.State, record.Message = providerAuthState(provider, credentials)
	if provider.EffectiveAuthMode() == aghconfig.ProviderAuthModeNativeCLI {
		nativeCLI, err := providerNativeCLIStatus(provider, deps.lookPath)
		if err != nil {
			return providerAuthStatusRecord{}, err
		}
		record.NativeCLI = nativeCLI
		if nativeCLI != nil && nativeCLI.Command != "" {
			if !nativeCLI.Present {
				record.State = providerAuthStateMissingCLI
				record.Message = providerNativeCLIMissingMessage(providerName, provider, nativeCLI)
				return record, nil
			}
			record.State = providerAuthStateNativeCLI
			record.Message = providerNativeCLIReadyMessage(providerName, provider, nativeCLI)
		}
	}
	if !probe || strings.TrimSpace(provider.AuthStatusCmd) == "" {
		return record, nil
	}
	env, err := providerAuthCommandEnv(runtime.HomePaths, providerName, provider)
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	result, err := deps.runProviderAuthCommand(ctx, providerAuthCommandSpec{
		Command: strings.TrimSpace(provider.AuthStatusCmd),
		Env:     env,
		Timeout: defaultProviderAuthCommandTimeout,
	})
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	record.Probe = &result
	if provider.EffectiveAuthMode() == aghconfig.ProviderAuthModeNativeCLI {
		if result.ExitCode == 0 {
			record.State = providerAuthStateAuthenticated
			record.Message = "Provider status command completed successfully."
		} else {
			record.State = providerAuthStateNeedsLogin
			record.Message = providerNativeCLIAuthProblemMessage(provider)
		}
	}
	return record, nil
}

func resolveProviderAuthTarget(
	cfg *aghconfig.Config,
	providerRef string,
) (string, aghconfig.ProviderConfig, error) {
	providerName := aghconfig.CanonicalProviderName(providerRef)
	if providerName == "" {
		return "", aghconfig.ProviderConfig{}, errors.New("cli: provider is required")
	}
	var effective aghconfig.Config
	if cfg != nil {
		effective = *cfg
	}
	provider, err := effective.ResolveProvider(providerName)
	if err != nil {
		return "", aghconfig.ProviderConfig{}, fmt.Errorf("cli: resolve provider %q: %w", providerName, err)
	}
	return providerName, provider, nil
}

func providerNativeCLIStatus(
	provider aghconfig.ProviderConfig,
	lookPath func(string) (string, error),
) (*providerNativeCLIStatusRecord, error) {
	return providerauth.NativeCLIStatusForProvider(provider, lookPath)
}

func providerNativeCLIStatusForCommand(
	command string,
	source string,
	lookPath func(string) (string, error),
) (*providerNativeCLIStatusRecord, error) {
	return providerauth.NativeCLIStatusForCommand(command, source, lookPath)
}

func providerMissingAuthLoginCommandError(providerName string, provider aghconfig.ProviderConfig) error {
	if provider.EffectiveAuthMode() != aghconfig.ProviderAuthModeNativeCLI {
		return fmt.Errorf("cli: provider %q does not define auth_login_command", providerName)
	}
	return fmt.Errorf(
		"cli: provider %q does not define auth_login_command; "+
			"run the provider's own login command outside AGH or set providers.%s.auth_login_command",
		providerName,
		providerName,
	)
}

func providerNativeCLIMissingMessage(
	providerName string,
	provider aghconfig.ProviderConfig,
	nativeCLI *providerNativeCLIStatusRecord,
) string {
	return providerauth.NativeCLIMissingMessage(providerName, provider, nativeCLI)
}

func providerNativeCLIReadyMessage(
	providerName string,
	provider aghconfig.ProviderConfig,
	nativeCLI *providerNativeCLIStatusRecord,
) string {
	return providerauth.NativeCLIReadyMessage(providerName, provider, nativeCLI)
}

func providerNativeCLIAuthProblemMessage(provider aghconfig.ProviderConfig) string {
	return providerauth.NativeCLIAuthProblemMessage(provider)
}

func providerNativeCLILoginCommandMessage(
	providerName string,
	operatorCommand string,
) string {
	return providerauth.NativeCLILoginCommandMessage(providerName, operatorCommand)
}

func rejectPrintCommandOutputFormat(cmd *cobra.Command) error {
	outputFlag := cmd.Flag(outputFlagName)
	if outputFlag != nil && outputFlag.Changed {
		return errors.New("cli: --print-command emits raw shell text and cannot be combined with --output")
	}
	jsonFlag := cmd.Flag(jsonFlagName)
	if jsonFlag != nil && jsonFlag.Changed {
		return errors.New("cli: --print-command emits raw shell text and cannot be combined with --json")
	}
	return nil
}

func providerNativeCLILoginEnv(
	homePaths aghconfig.HomePaths,
	providerName string,
	provider aghconfig.ProviderConfig,
) ([]string, error) {
	return providerauth.NativeCLILoginEnv(homePaths, providerName, provider, os.Environ())
}

func providerOperatorLoginCommand(command string, loginEnv []string) (string, error) {
	return providerauth.OperatorLoginCommand(command, loginEnv)
}

func providerAuthState(
	provider aghconfig.ProviderConfig,
	credentials []providerCredentialStatusItem,
) (string, string) {
	switch provider.EffectiveAuthMode() {
	case aghconfig.ProviderAuthModeBoundSecret:
		for _, credential := range credentials {
			if credential.Required && !credential.Present {
				return providerAuthStateMissingRequired, "Missing required AGH-managed provider credential."
			}
		}
		return providerAuthStatePresent, "Required AGH-managed provider credentials are present."
	case aghconfig.ProviderAuthModeNone:
		return statusStateNone, "Provider starts without AGH-managed authentication."
	default:
		return providerAuthStateNativeCLI, "Provider owns authentication through its native CLI login state."
	}
}

func providerCredentialStatuses(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	provider aghconfig.ProviderConfig,
	getenv func(string) string,
) ([]providerCredentialStatusItem, error) {
	slots := provider.EffectiveCredentialSlots()
	if len(slots) == 0 {
		return nil, nil
	}
	statuses := make([]providerCredentialStatusItem, 0, len(slots))
	for _, slot := range slots {
		status, err := providerCredentialStatus(ctx, homePaths, slot, getenv)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func providerCredentialStatus(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	slot aghconfig.ProviderCredentialSlot,
	getenv func(string) string,
) (providerCredentialStatusItem, error) {
	secretRef := vault.NormalizeRef(slot.SecretRef)
	status := providerCredentialStatusItem{
		Name:      strings.TrimSpace(slot.Name),
		TargetEnv: strings.TrimSpace(slot.TargetEnv),
		SecretRef: secretRef,
		Kind:      strings.TrimSpace(slot.Kind),
		Required:  slot.Required,
	}
	switch {
	case vault.IsEnvRef(secretRef):
		status.Source = configEnvKey
		envName, err := vault.EnvNameFromRef(secretRef)
		if err != nil {
			return providerCredentialStatusItem{}, err
		}
		status.Present = strings.TrimSpace(getenv(envName)) != ""
		return status, nil
	case vault.IsSecretRef(secretRef):
		status.Source = providerVaultKey
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			return providerCredentialStatusItem{}, fmt.Errorf("cli: open global DB for provider auth status: %w", err)
		}
		lookupEnv := func(key string) (string, bool) {
			value := getenv(key)
			return value, strings.TrimSpace(value) != ""
		}
		service, err := vault.NewService(
			db,
			vault.NewFileKeyProvider(homePaths.HomeDir, lookupEnv),
			vault.WithLookupEnv(lookupEnv),
		)
		if err != nil {
			closeErr := db.Close(ctx)
			if closeErr != nil {
				return providerCredentialStatusItem{}, fmt.Errorf(
					"cli: initialize provider auth vault status: %w; close global DB: %v",
					err,
					closeErr,
				)
			}
			return providerCredentialStatusItem{}, fmt.Errorf("cli: initialize provider auth vault status: %w", err)
		}
		metadata, err := service.GetMetadata(ctx, secretRef)
		closeErr := db.Close(ctx)
		if err != nil {
			if errors.Is(err, vault.ErrSecretNotFound) {
				if closeErr != nil {
					return providerCredentialStatusItem{}, fmt.Errorf(
						"cli: close provider auth vault status DB: %w",
						closeErr,
					)
				}
				return status, nil
			}
			if closeErr != nil {
				return providerCredentialStatusItem{}, fmt.Errorf(
					"cli: read provider credential metadata: %w; close global DB: %v",
					err,
					closeErr,
				)
			}
			return providerCredentialStatusItem{}, fmt.Errorf("cli: read provider credential metadata: %w", err)
		}
		if closeErr != nil {
			return providerCredentialStatusItem{}, fmt.Errorf("cli: close provider auth vault status DB: %w", closeErr)
		}
		status.Present = metadata.Present
		return status, nil
	default:
		status.Source = "unsupported"
		return status, nil
	}
}

func providerAuthCommandEnv(
	homePaths aghconfig.HomePaths,
	providerName string,
	provider aghconfig.ProviderConfig,
) ([]string, error) {
	return providerauth.CommandEnv(homePaths, providerName, provider, os.Environ())
}

func defaultProviderAuthCommandRunner(
	ctx context.Context,
	spec providerAuthCommandSpec,
) (providerAuthCommandResult, error) {
	command := strings.TrimSpace(spec.Command)
	if command == "" {
		return providerAuthCommandResult{}, errors.New("cli: provider auth command is required")
	}
	argv, err := shellquote.Split(command)
	if err != nil {
		return providerAuthCommandResult{}, fmt.Errorf("cli: parse provider auth command: %w", err)
	}
	if len(argv) == 0 {
		return providerAuthCommandResult{}, errors.New("cli: provider auth command is empty")
	}
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = defaultProviderAuthCommandTimeout
	}
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 -- Provider auth commands are operator-configured commands intentionally
	// executed by this CLI verb after shell-style parsing.
	execCmd := exec.CommandContext(commandCtx, argv[0], argv[1:]...)
	execCmd.Env = append([]string(nil), spec.Env...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr
	err = execCmd.Run()
	result := providerAuthCommandResult{
		ExitCode: exitCodeFromError(err),
		Stdout:   diagnostics.RedactAndBound(stdout.String(), 4096),
		Stderr:   diagnostics.RedactAndBound(stderr.String(), 4096),
	}
	if commandCtx.Err() != nil {
		return result, commandCtx.Err()
	}
	if err == nil {
		return result, nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr != nil {
		return result, nil
	}
	return result, fmt.Errorf("cli: run provider auth command: %w", err)
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func providerAuthStatusBundle(record providerAuthStatusRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			sections := []string{renderHumanSection("Provider Auth", providerAuthStatusRows(record))}
			if len(record.Credentials) > 0 {
				sections = append(sections, renderHumanTable(
					"Credentials",
					[]string{
						providerNameValue,
						"Target",
						authoredContextSourceValue,
						"Required",
						authoredContextPresentValue,
					},
					providerCredentialStatusRows(record.Credentials),
				))
			}
			return strings.Join(sections, "\n"), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"provider_auth",
				[]string{providerProviderKey, "auth_mode", "env_policy", "home_policy", providerStateKey},
				[]string{record.Provider, record.AuthMode, record.EnvPolicy, record.HomePolicy, record.State},
			), nil
		},
	}
}

func providerAuthStatusRows(record providerAuthStatusRecord) []keyValue {
	rows := []keyValue{
		{Label: providerProviderValue, Value: stringOrDash(record.Provider)},
		{Label: "Display Name", Value: stringOrDash(record.DisplayName)},
		{Label: "Auth Mode", Value: stringOrDash(record.AuthMode)},
		{Label: "Env Policy", Value: stringOrDash(record.EnvPolicy)},
		{Label: "Home Policy", Value: stringOrDash(record.HomePolicy)},
		{Label: providerStateValue, Value: stringOrDash(record.State)},
		{Label: providerMessageValue, Value: stringOrDash(record.Message)},
		{Label: "Status Command", Value: stringOrDash(record.StatusCommand)},
		{Label: "Login Command", Value: stringOrDash(record.LoginCommand)},
	}
	if len(record.LoginEnv) > 0 {
		rows = append(rows, keyValue{Label: "Login Env", Value: strings.Join(record.LoginEnv, " ")})
	}
	if record.NativeCLI != nil {
		rows = append(rows,
			keyValue{Label: "Native CLI Command", Value: stringOrDash(record.NativeCLI.Command)},
			keyValue{Label: "Native CLI Present", Value: boolString(record.NativeCLI.Present)},
			keyValue{Label: "Native CLI Path", Value: stringOrDash(record.NativeCLI.Path)},
			keyValue{Label: "Native CLI Source", Value: stringOrDash(record.NativeCLI.Source)},
		)
		if record.NativeCLI.Error != "" {
			rows = append(rows, keyValue{Label: "Native CLI Error", Value: record.NativeCLI.Error})
		}
	}
	if record.Probe != nil {
		rows = append(rows,
			keyValue{Label: "Probe Exit Code", Value: fmt.Sprintf("%d", record.Probe.ExitCode)},
			keyValue{Label: "Probe Stdout", Value: stringOrDash(strings.TrimSpace(record.Probe.Stdout))},
			keyValue{Label: "Probe Stderr", Value: stringOrDash(strings.TrimSpace(record.Probe.Stderr))},
		)
	}
	return rows
}

func providerCredentialStatusRows(statuses []providerCredentialStatusItem) [][]string {
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		rows = append(rows, []string{
			stringOrDash(status.Name),
			stringOrDash(status.TargetEnv),
			stringOrDash(status.Source),
			boolString(status.Required),
			boolString(status.Present),
		})
	}
	return rows
}
