package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/providerauth"
	authproviders "github.com/pedronauck/agh/internal/providers"
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
	providerAuthModeKey   = cliAuthModeKey
	providerAuthModeValue = "Auth Mode"
	providerEnvPolicyKey  = "env_policy"
	providerHomePolicyKey = "home_policy"
	providerProviderKey   = "provider"
	providerStateKey      = "state"
	providerVaultKey      = "vault"
)

const (
	defaultProviderAuthCommandTimeout = authproviders.DefaultProviderAuthCommandTimeout
)

type providerAuthCommandRunner = authproviders.ProviderAuthCommandRunner

type providerAuthCommandSpec = authproviders.ProviderAuthCommandSpec

type providerAuthCommandResult = authproviders.ProviderAuthCommandResult

type providerAuthStatusRecord struct {
	Provider      string                         `json:"provider"`
	DisplayName   string                         `json:"display_name,omitempty"`
	AuthMode      string                         `json:"auth_mode"`
	EnvPolicy     string                         `json:"env_policy"`
	HomePolicy    string                         `json:"home_policy"`
	State         string                         `json:"state"`
	Code          string                         `json:"code,omitempty"`
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
		d.runProviderAuthCommand = authproviders.DefaultProviderAuthCommandRunner
	}
	if d.runProviderAuthLoginCommand == nil {
		d.runProviderAuthLoginCommand = authproviders.DefaultProviderAuthLoginRunner
	}
	return d
}

func newProviderCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   providerProviderKey,
		Short: "Inspect and manage provider authentication",
	}
	cmd.AddCommand(newProviderAuthCommand(deps))
	cmd.AddCommand(newProviderListCommand(deps))
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
	var remote bool
	cmd := &cobra.Command{
		Use:   "status <provider>",
		Short: "Show provider authentication status",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if remote {
				client, err := clientFromDeps(deps)
				if err != nil {
					return err
				}
				record, err := client.ProbeProviderAuth(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, providerAuthProbeBundle(record))
			}
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
	cmd.Flags().BoolVar(&remote, "remote", false, "Run the provider status probe through the daemon")
	return cmd
}

func newProviderAuthLoginCommand(deps commandDeps) *cobra.Command {
	var printCommand bool
	var noTTY bool
	var timeout time.Duration
	options := func() providerAuthLoginOptions {
		return providerAuthLoginOptions{printCommand: printCommand, noTTY: noTTY, timeout: timeout}
	}
	cmd := &cobra.Command{
		Use:   "login <provider>",
		Short: "Run the provider native login command locally",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProviderAuthLoginCommand(cmd, deps, args[0], options())
		},
	}
	cmd.Flags().
		BoolVar(&printCommand, "print-command", false, "Print only the resolved provider login command")
	cmd.Flags().BoolVar(&noTTY, "no-tty", false, "Disable TTY attachment for the login command")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Optional login command timeout")
	return cmd
}

type providerAuthLoginOptions struct {
	printCommand bool
	noTTY        bool
	timeout      time.Duration
}

func newProviderListCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   extensionListKey,
		Short: "List providers and declared auth state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			record, err := client.ListProviders(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, providerListBundle(record))
		},
	}
	return cmd
}

func providerLoginCommandEnv(loginEnv []string) []string {
	if len(loginEnv) == 0 {
		return os.Environ()
	}
	return append(os.Environ(), loginEnv...)
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

func runProviderAuthLoginCommand(
	cmd *cobra.Command,
	deps commandDeps,
	providerRef string,
	options providerAuthLoginOptions,
) error {
	runtime, err := providerAuthRuntime(deps)
	if err != nil {
		return err
	}
	target, err := providerAuthLoginTarget(runtime, deps, providerRef)
	if err != nil {
		return err
	}
	if options.printCommand {
		if err := rejectPrintCommandOutputFormat(cmd); err != nil {
			return err
		}
		return writeRawCommandOutput(cmd, target.OperatorCommand+"\n")
	}
	loginResult, err := deps.runProviderAuthLoginCommand(cmd.Context(), providerAuthCommandSpec{
		Command: target.LoginCommand,
		Env:     providerLoginCommandEnv(target.LoginEnv),
		Timeout: options.timeout,
		NoTTY:   options.noTTY,
	})
	if err != nil {
		return err
	}
	record, err := buildProviderAuthStatus(cmd.Context(), deps, runtime, target.ProviderName, true)
	if err != nil {
		return err
	}
	applyProviderAuthLoginResult(&record, target.Provider, loginResult)
	record.LoginCommand = firstNonEmptyString(record.LoginCommand, target.LoginCommand)
	record.LoginEnv = target.LoginEnv
	record.NativeCLI = target.NativeCLI
	return writeCommandOutput(cmd, providerAuthStatusBundle(record))
}

type providerAuthLoginTargetRecord struct {
	ProviderName    string
	Provider        aghconfig.ProviderConfig
	LoginCommand    string
	LoginEnv        []string
	NativeCLI       *providerNativeCLIStatusRecord
	OperatorCommand string
}

func providerAuthLoginTarget(
	runtime *runtimeContext,
	deps commandDeps,
	providerRef string,
) (providerAuthLoginTargetRecord, error) {
	providerName, provider, err := resolveProviderAuthTarget(&runtime.Config, providerRef)
	if err != nil {
		return providerAuthLoginTargetRecord{}, err
	}
	loginCommand := strings.TrimSpace(provider.AuthLoginCmd)
	if loginCommand == "" {
		return providerAuthLoginTargetRecord{}, providerMissingAuthLoginCommandError(providerName, provider)
	}
	if provider.EffectiveAuthMode() != aghconfig.ProviderAuthModeNativeCLI {
		return providerAuthLoginTargetRecord{}, fmt.Errorf(
			"cli: provider %q uses auth_mode %q; provider auth login only exposes native_cli login commands",
			providerName,
			provider.EffectiveAuthMode(),
		)
	}
	nativeCLI, err := providerNativeCLIStatusForCommand(
		loginCommand,
		providerauth.NativeCLISourceAuthLogin,
		deps.lookPath,
	)
	if err != nil {
		return providerAuthLoginTargetRecord{}, err
	}
	if nativeCLI != nil && !nativeCLI.Present {
		return providerAuthLoginTargetRecord{}, fmt.Errorf(
			"cli: provider %q native CLI %q was not found on PATH; install it before running %q",
			providerName,
			nativeCLI.Command,
			loginCommand,
		)
	}
	loginEnv, err := providerNativeCLILoginEnv(runtime.HomePaths, providerName, provider)
	if err != nil {
		return providerAuthLoginTargetRecord{}, err
	}
	operatorCommand, err := providerOperatorLoginCommand(loginCommand, loginEnv)
	if err != nil {
		return providerAuthLoginTargetRecord{}, err
	}
	return providerAuthLoginTargetRecord{
		ProviderName:    providerName,
		Provider:        provider,
		LoginCommand:    loginCommand,
		LoginEnv:        loginEnv,
		NativeCLI:       nativeCLI,
		OperatorCommand: operatorCommand,
	}, nil
}

func applyProviderAuthLoginResult(
	record *providerAuthStatusRecord,
	provider aghconfig.ProviderConfig,
	loginResult providerAuthCommandResult,
) {
	if record == nil || loginResult.ExitCode == 0 {
		return
	}
	classification := authproviders.ClassifyProbeResult(provider, authproviders.ProbeOutcome{
		ExitCode: loginResult.ExitCode,
		Stdout:   loginResult.Stdout,
		Stderr:   loginResult.Stderr,
	}, nil)
	record.State = string(classification.State)
	record.Code = classification.Code
	record.Message = classification.Message
	record.Probe = &loginResult
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
	probeEnv, err := providerAuthProbeEnv(runtime.HomePaths, providerName, provider, deps)
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	credentials, err := providerCredentialStatuses(ctx, provider, &probeEnv)
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	classification, err := authproviders.ClassifyDeclared(ctx, provider, &probeEnv)
	if err != nil {
		return providerAuthStatusRecord{}, err
	}
	record := providerAuthStatusRecord{
		Provider:      providerName,
		DisplayName:   strings.TrimSpace(provider.DisplayName),
		AuthMode:      string(provider.EffectiveAuthMode()),
		EnvPolicy:     string(provider.EffectiveEnvPolicy()),
		HomePolicy:    string(provider.EffectiveHomePolicy()),
		State:         string(classification.State),
		Code:          classification.Code,
		Message:       classification.Message,
		StatusCommand: strings.TrimSpace(provider.AuthStatusCmd),
		LoginCommand:  strings.TrimSpace(provider.AuthLoginCmd),
		Credentials:   credentials,
	}
	nativeReady, err := populateProviderNativeCLIStatus(&record, providerName, provider, deps, &probeEnv)
	if err != nil || !nativeReady {
		return record, err
	}
	if !probe || strings.TrimSpace(provider.AuthStatusCmd) == "" {
		return record, nil
	}
	if err := populateProviderAuthProbe(ctx, &record, provider, deps, &probeEnv); err != nil {
		return providerAuthStatusRecord{}, err
	}
	return record, nil
}

func populateProviderNativeCLIStatus(
	record *providerAuthStatusRecord,
	providerName string,
	provider aghconfig.ProviderConfig,
	deps commandDeps,
	probeEnv *authproviders.ProbeEnv,
) (bool, error) {
	if provider.EffectiveAuthMode() != aghconfig.ProviderAuthModeNativeCLI {
		return true, nil
	}
	nativeCLI, err := providerNativeCLIStatus(provider, deps.lookPath)
	if err != nil {
		return false, err
	}
	record.NativeCLI = nativeCLI
	if nativeCLI == nil || nativeCLI.Command == "" || nativeCLI.Present {
		return true, nil
	}
	missing := authproviders.ClassifyProbeResult(provider, authproviders.ProbeOutcome{
		ExitCode: -1,
		Stderr:   providerNativeCLIMissingMessage(providerName, provider, nativeCLI),
	}, probeEnv)
	record.State = string(missing.State)
	record.Code = missing.Code
	record.Message = missing.Message
	return false, nil
}

func populateProviderAuthProbe(
	ctx context.Context,
	record *providerAuthStatusRecord,
	provider aghconfig.ProviderConfig,
	deps commandDeps,
	probeEnv *authproviders.ProbeEnv,
) error {
	result, err := deps.runProviderAuthCommand(ctx, providerAuthCommandSpec{
		Command: strings.TrimSpace(provider.AuthStatusCmd),
		Env:     probeEnv.CommandEnv,
		Timeout: defaultProviderAuthCommandTimeout,
		NoTTY:   true,
	})
	if err != nil {
		return err
	}
	record.Probe = &result
	if provider.EffectiveAuthMode() == aghconfig.ProviderAuthModeNativeCLI {
		classification := authproviders.ClassifyProbeResult(provider, authproviders.ProbeOutcome{
			ExitCode: result.ExitCode,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
		}, probeEnv)
		record.State = string(classification.State)
		record.Code = classification.Code
		record.Message = classification.Message
	}
	return nil
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

func providerCredentialStatuses(
	ctx context.Context,
	provider aghconfig.ProviderConfig,
	env *authproviders.ProbeEnv,
) ([]providerCredentialStatusItem, error) {
	statuses, err := authproviders.CredentialStatuses(ctx, provider, env)
	if err != nil {
		return nil, err
	}
	if len(statuses) == 0 {
		return nil, nil
	}
	items := make([]providerCredentialStatusItem, 0, len(statuses))
	for _, status := range statuses {
		items = append(items, providerCredentialStatusItem{
			Name:      status.Name,
			TargetEnv: status.TargetEnv,
			SecretRef: status.SecretRef,
			Kind:      status.Kind,
			Required:  status.Required,
			Present:   status.Present,
			Source:    status.Source,
		})
	}
	return items, nil
}

func providerAuthProbeEnv(
	homePaths aghconfig.HomePaths,
	providerName string,
	provider aghconfig.ProviderConfig,
	deps commandDeps,
) (authproviders.ProbeEnv, error) {
	commandEnv, err := providerauth.CommandEnv(homePaths, providerName, provider, os.Environ())
	if err != nil {
		return authproviders.ProbeEnv{}, err
	}
	return authproviders.ProbeEnv{
		ProviderName: providerName,
		HomePaths:    homePaths,
		LookPath:     deps.lookPath,
		LookupEnv: func(key string) (string, bool) {
			value := deps.getenv(key)
			return value, strings.TrimSpace(value) != ""
		},
		Vault:      cliProviderVaultMetadataResolver{homePaths: homePaths, getenv: deps.getenv},
		CommandEnv: commandEnv,
		RunCommand: deps.runProviderAuthCommand,
	}, nil
}

type cliProviderVaultMetadataResolver struct {
	homePaths aghconfig.HomePaths
	getenv    func(string) string
}

func (r cliProviderVaultMetadataResolver) GetMetadata(ctx context.Context, ref string) (vault.Metadata, error) {
	db, err := globaldb.OpenGlobalDB(ctx, r.homePaths.DatabaseFile)
	if err != nil {
		return vault.Metadata{}, fmt.Errorf("cli: open global DB for provider auth status: %w", err)
	}
	lookupEnv := func(key string) (string, bool) {
		value := r.getenv(key)
		return value, strings.TrimSpace(value) != ""
	}
	service, err := vault.NewService(
		db,
		vault.NewFileKeyProvider(r.homePaths.HomeDir, lookupEnv),
		vault.WithLookupEnv(lookupEnv),
	)
	if err != nil {
		closeErr := db.Close(ctx)
		if closeErr != nil {
			return vault.Metadata{}, fmt.Errorf(
				"cli: initialize provider auth vault status: %w; close global DB: %v",
				err,
				closeErr,
			)
		}
		return vault.Metadata{}, fmt.Errorf("cli: initialize provider auth vault status: %w", err)
	}
	metadata, err := service.GetMetadata(ctx, ref)
	closeErr := db.Close(ctx)
	if err != nil {
		if errors.Is(err, vault.ErrSecretNotFound) {
			if closeErr != nil {
				return vault.Metadata{}, fmt.Errorf("cli: close provider auth vault status DB: %w", closeErr)
			}
			return vault.Metadata{}, err
		}
		if closeErr != nil {
			return vault.Metadata{}, fmt.Errorf(
				"cli: read provider credential metadata: %w; close global DB: %v",
				err,
				closeErr,
			)
		}
		return vault.Metadata{}, fmt.Errorf("cli: read provider credential metadata: %w", err)
	}
	if closeErr != nil {
		return vault.Metadata{}, fmt.Errorf("cli: close provider auth vault status DB: %w", closeErr)
	}
	return metadata, nil
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
				[]string{
					providerProviderKey,
					providerAuthModeKey,
					providerEnvPolicyKey,
					providerHomePolicyKey,
					providerStateKey,
				},
				[]string{record.Provider, record.AuthMode, record.EnvPolicy, record.HomePolicy, record.State},
			), nil
		},
	}
}

func providerAuthProbeBundle(record contract.ProviderAuthProbeResponse) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: providerProviderValue, Value: stringOrDash(record.Provider)},
				{Label: providerAuthModeValue, Value: stringOrDash(record.AuthStatus.Mode)},
				{Label: "Env Policy", Value: stringOrDash(record.AuthStatus.EnvPolicy)},
				{Label: "Home Policy", Value: stringOrDash(record.AuthStatus.HomePolicy)},
				{Label: providerStateValue, Value: stringOrDash(record.AuthStatus.State)},
				{Label: cliCodeValue, Value: stringOrDash(record.AuthStatus.Code)},
				{Label: providerMessageValue, Value: stringOrDash(record.AuthStatus.Message)},
			}
			if record.Probe != nil {
				rows = append(rows,
					keyValue{Label: "Probe Exit Code", Value: fmt.Sprintf("%d", record.Probe.ExitCode)},
					keyValue{Label: "Probe Stdout", Value: stringOrDash(strings.TrimSpace(record.Probe.Stdout))},
					keyValue{Label: "Probe Stderr", Value: stringOrDash(strings.TrimSpace(record.Probe.Stderr))},
				)
			}
			return renderHumanSection("Provider Auth", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"provider_auth",
				[]string{providerProviderKey, providerAuthModeKey, providerStateKey},
				[]string{record.Provider, record.AuthStatus.Mode, record.AuthStatus.State},
			), nil
		},
	}
}

func providerListBundle(record contract.ProviderListResponse) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			rows := make([][]string, 0, len(record.Providers))
			for _, provider := range record.Providers {
				rows = append(rows, []string{
					stringOrDash(provider.Name),
					stringOrDash(provider.DisplayName),
					stringOrDash(provider.AuthStatus.State),
					stringOrDash(provider.AuthStatus.Mode),
				})
			}
			return renderHumanTable(
				"Providers",
				[]string{providerNameValue, "Display Name", providerStateValue, providerAuthModeValue},
				rows,
			), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"providers",
				[]string{"count"},
				[]string{fmt.Sprintf("%d", len(record.Providers))},
			), nil
		},
	}
}

func providerAuthStatusRows(record providerAuthStatusRecord) []keyValue {
	rows := []keyValue{
		{Label: providerProviderValue, Value: stringOrDash(record.Provider)},
		{Label: "Display Name", Value: stringOrDash(record.DisplayName)},
		{Label: providerAuthModeValue, Value: stringOrDash(record.AuthMode)},
		{Label: "Env Policy", Value: stringOrDash(record.EnvPolicy)},
		{Label: "Home Policy", Value: stringOrDash(record.HomePolicy)},
		{Label: providerStateValue, Value: stringOrDash(record.State)},
		{Label: cliCodeValue, Value: stringOrDash(record.Code)},
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
