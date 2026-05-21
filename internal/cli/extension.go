package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/spf13/cobra"
)

const (
	sessionTypeValue = "Type"
)

const (
	extensionTypeKey = "type"
)

const (
	extensionCapabilitiesValue = "Capabilities"
	extensionEnabledValue      = "Enabled"
	extensionHealthValue       = "Health"
	extensionCapabilitiesKey   = "capabilities"
	extensionEnabledKey        = "enabled"
	extensionExtensionKey      = "extension"
	extensionHealthKey         = "health"
	extensionListKey           = "list"
	extensionSearchQueryValue  = "search <query>"
	cliUseEnableName           = "enable <name>"
	cliUseDisableName          = "disable <name>"
)

type preparedExtensionInstall struct {
	Path     string
	Manifest *extensionpkg.Manifest
	Checksum string
}

type localExtensionRegistry interface {
	Install(manifest *extensionpkg.Manifest, path string, checksum string, opts ...extensionpkg.InstallOption) error
	List() ([]extensionpkg.ExtensionInfo, error)
	Get(name string) (*extensionpkg.ExtensionInfo, error)
	Enable(name string) error
	Disable(name string) error
	Uninstall(name string) error
}

func newExtensionCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   extensionExtensionKey,
		Short: "Manage AGH extensions",
	}

	cmd.AddCommand(newExtensionSearchCommand(deps))
	cmd.AddCommand(newExtensionListCommand(deps))
	cmd.AddCommand(newExtensionInstallCommand(deps))
	cmd.AddCommand(newExtensionRemoveCommand(deps))
	cmd.AddCommand(newExtensionUpdateCommand(deps))
	cmd.AddCommand(newExtensionEnableCommand(deps))
	cmd.AddCommand(newExtensionDisableCommand(deps))
	cmd.AddCommand(newExtensionStatusCommand(deps))
	cmd.AddCommand(newExtensionProvenanceCommand(deps))
	return cmd
}

func newExtensionSearchCommand(deps commandDeps) *cobra.Command {
	limit := defaultExtensionRegistrySearchLimit
	var sourceFilter string

	cmd := &cobra.Command{
		Use:   extensionSearchQueryValue,
		Short: "Search remote extension registries",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := searchExtensions(cmd.Context(), deps, args[0], sourceFilter, limit)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionSearchBundle(results))
		},
	}
	cmd.Flags().
		IntVar(&limit, "limit", defaultExtensionRegistrySearchLimit, "Maximum number of extension registry results to return")
	cmd.Flags().StringVar(&sourceFilter, "from", "", "Only query one configured extension registry source")
	return cmd
}

func newExtensionListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   extensionListKey,
		Short: "List installed extensions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			items, err := loadExtensionRecords(cmd.Context(), deps)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionListBundle(items))
		},
	}
}

func newExtensionInstallCommand(deps commandDeps) *cobra.Command {
	var sourceFilter string
	var version string
	var asset string
	var allowUnverified bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "install <path-or-slug>",
		Short: "Install a local extension or download one from a registry",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			prepared, isLocalPath, err := prepareLocalExtensionInstallIfPresent(args[0])
			if err != nil {
				return err
			}
			if err := confirmExtensionUnverifiedInstall(cmd, allowUnverified, yes); err != nil {
				return err
			}
			if isLocalPath {
				if strings.TrimSpace(sourceFilter) != "" || strings.TrimSpace(version) != "" ||
					strings.TrimSpace(asset) != "" {
					return errors.New("cli: --from, --version, and --asset are only supported for registry installs")
				}

				item, err := installExtension(cmd.Context(), deps, prepared, allowUnverified)
				if err != nil {
					return err
				}
				return writeCommandOutput(cmd, extensionBundle(item))
			}

			item, err := installMarketplaceExtension(
				cmd.Context(),
				deps,
				args[0],
				sourceFilter,
				version,
				asset,
				allowUnverified,
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionBundle(item))
		},
	}
	cmd.Flags().StringVar(&sourceFilter, "from", "", "Only use one configured extension registry source")
	cmd.Flags().StringVar(&version, daemonVersionKey, "", "Install a specific registry version")
	cmd.Flags().StringVar(&asset, "asset", "", "Select a specific registry asset when multiple archives exist")
	cmd.Flags().BoolVar(
		&allowUnverified,
		"allow-unverified",
		false,
		"Allow install when the extension checksum is not registry-verified",
	)
	cmd.Flags().BoolVar(&yes, yesFlagName, false, "Skip confirmation when using --allow-unverified")
	return cmd
}

func newExtensionRemoveCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed extension from disk and the registry",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := removeInstalledExtension(cmd.Context(), deps, args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionRemoveBundle(item))
		},
	}
}

func newExtensionUpdateCommand(deps commandDeps) *cobra.Command {
	var updateAll bool
	var checkOnly bool
	var version string
	var allowUnverified bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Check for or install updates for marketplace extensions",
		Args: func(_ *cobra.Command, args []string) error {
			if updateAll && len(args) > 0 {
				return errors.New("cli: update accepts either an extension name or --all, not both")
			}
			if !updateAll && len(args) != 1 {
				return errors.New("cli: update requires an extension name unless --all is set")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmExtensionUnverifiedInstall(cmd, allowUnverified && !checkOnly, yes); err != nil {
				return err
			}
			items, err := updateMarketplaceExtensions(
				cmd.Context(),
				deps,
				args,
				updateAll,
				checkOnly,
				version,
				allowUnverified,
			)
			if err != nil {
				return err
			}
			if err := writeCommandOutput(cmd, extensionUpdateBundle(items)); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&updateAll, "all", false, "Update every installed marketplace extension")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing them")
	cmd.Flags().StringVar(&version, daemonVersionKey, "", "Update to a specific registry version")
	cmd.Flags().BoolVar(
		&allowUnverified,
		"allow-unverified",
		false,
		"Allow update when the extension checksum is not registry-verified",
	)
	cmd.Flags().BoolVar(&yes, yesFlagName, false, "Skip confirmation when using --allow-unverified")
	return cmd
}

func newExtensionEnableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   cliUseEnableName,
		Short: "Enable an installed extension",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := mutateExtensionEnabled(cmd.Context(), deps, args[0], true)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionBundle(item))
		},
	}
}

func newExtensionDisableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   cliUseDisableName,
		Short: "Disable an installed extension",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := mutateExtensionEnabled(cmd.Context(), deps, args[0], false)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionBundle(item))
		},
	}
}

func newExtensionStatusCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show extension runtime status",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := extensionStatus(cmd.Context(), deps, args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionBundle(item))
		},
	}
}

func newExtensionProvenanceCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "provenance <name>",
		Short: "Show extension provenance and trust report",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := extensionProvenance(cmd.Context(), deps, args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionProvenanceBundle(item))
		},
	}
}

func loadExtensionRecords(ctx context.Context, deps commandDeps) ([]ExtensionRecord, error) {
	client, running, err := extensionClientIfRunning(deps)
	if err != nil {
		return nil, err
	}
	if running {
		return client.ListExtensions(ctx)
	}

	return withLocalExtensionRegistry(
		ctx,
		deps,
		func(_ *runtimeContext, registry localExtensionRegistry) ([]ExtensionRecord, error) {
			infos, err := registry.List()
			if err != nil {
				return nil, err
			}

			items := make([]ExtensionRecord, 0, len(infos))
			for _, info := range infos {
				items = append(items, localExtensionRecord(info, deps.now, deps.getenv))
			}
			return items, nil
		},
	)
}

func installExtension(
	ctx context.Context,
	deps commandDeps,
	prepared preparedExtensionInstall,
	allowUnverified bool,
) (ExtensionRecord, error) {
	client, running, err := extensionClientIfRunning(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	if running {
		return client.InstallExtension(ctx, InstallExtensionRequest{
			Path:            prepared.Path,
			Checksum:        prepared.Checksum,
			AllowUnverified: allowUnverified,
		})
	}
	if !allowUnverified {
		return ExtensionRecord{}, extensionpkg.NewExtensionChecksumUnverifiedError(
			prepared.Manifest.Name,
			prepared.Path,
		)
	}

	return withLocalExtensionRegistry(
		ctx,
		deps,
		func(runtime *runtimeContext, registry localExtensionRegistry) (ExtensionRecord, error) {
			if err := installPreparedExtension(
				runtime.HomePaths,
				registry,
				prepared,
				deps.now(),
				allowUnverified,
			); err != nil {
				return ExtensionRecord{}, err
			}
			info, err := registry.Get(prepared.Manifest.Name)
			if err != nil {
				return ExtensionRecord{}, err
			}
			return localExtensionRecord(*info, deps.now, deps.getenv), nil
		},
	)
}

func mutateExtensionEnabled(ctx context.Context, deps commandDeps, name string, enabled bool) (ExtensionRecord, error) {
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	if enabled {
		return client.EnableExtension(ctx, name)
	}
	return client.DisableExtension(ctx, name)
}

func extensionStatus(ctx context.Context, deps commandDeps, name string) (ExtensionRecord, error) {
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	return client.ExtensionStatus(ctx, name)
}

func extensionProvenance(ctx context.Context, deps commandDeps, name string) (ExtensionProvenanceRecord, error) {
	client, err := requireExtensionDaemonClient(deps)
	if err != nil {
		return ExtensionProvenanceRecord{}, err
	}
	return client.ExtensionProvenance(ctx, name)
}

func extensionClientIfRunning(deps commandDeps) (DaemonClient, bool, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, false, err
	}

	info, running, err := daemonInfo(runtime.HomePaths, deps)
	if err != nil {
		return nil, false, err
	}
	if !running {
		return nil, false, nil
	}
	if info == (aghdaemon.Info{}) {
		return nil, false, nil
	}

	client, err := clientFromDeps(deps)
	if err != nil {
		return nil, false, err
	}
	return client, true, nil
}

func requireExtensionDaemonClient(deps commandDeps) (DaemonClient, error) {
	client, running, err := extensionClientIfRunning(deps)
	if err != nil {
		return nil, err
	}
	if !running {
		return nil, errors.New("cli: extension marketplace operations require a running daemon")
	}
	return client, nil
}

func confirmExtensionUnverifiedInstall(cmd *cobra.Command, allowUnverified bool, yes bool) error {
	if !allowUnverified || yes {
		return nil
	}
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	if mode != OutputHuman {
		return errors.New("cli: --allow-unverified requires --yes for structured output")
	}
	message := "This extension checksum is unverified. Continue with allow_unverified? [y/N] "
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), message); err != nil {
		return fmt.Errorf("cli: write extension trust prompt: %w", err)
	}
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("cli: read extension trust prompt: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != yesFlagName {
		return errors.New("cli: extension install declined")
	}
	return nil
}

func withLocalExtensionRegistry[T any](
	ctx context.Context,
	deps commandDeps,
	fn func(runtime *runtimeContext, registry localExtensionRegistry) (T, error),
) (result T, err error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return result, err
	}
	if err := deps.ensureHome(runtime.HomePaths); err != nil {
		return result, err
	}

	globalDB, err := globaldb.OpenGlobalDB(ctx, runtime.HomePaths.DatabaseFile)
	if err != nil {
		return result, fmt.Errorf("cli: open extension database %q: %w", runtime.HomePaths.DatabaseFile, err)
	}
	defer func() {
		closeErr := globalDB.Close(ctx)
		if closeErr == nil {
			return
		}
		closeErr = fmt.Errorf("cli: close extension database %q: %w", runtime.HomePaths.DatabaseFile, closeErr)
		if err == nil {
			err = closeErr
			return
		}
		err = errors.Join(err, closeErr)
	}()

	return fn(runtime, extensionpkg.NewRegistry(globalDB.DB()))
}

func prepareExtensionInstall(path string) (preparedExtensionInstall, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return preparedExtensionInstall{}, errors.New("extension: install path is required")
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return preparedExtensionInstall{}, fmt.Errorf("extension: resolve install path %q: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return preparedExtensionInstall{}, fmt.Errorf("extension: stat install path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return preparedExtensionInstall{}, fmt.Errorf("extension: install path %q must be a directory", absPath)
	}

	manifest, err := extensionpkg.LoadManifest(absPath)
	if err != nil {
		return preparedExtensionInstall{}, err
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(absPath)
	if err != nil {
		return preparedExtensionInstall{}, err
	}

	return preparedExtensionInstall{
		Path:     absPath,
		Manifest: manifest,
		Checksum: checksum,
	}, nil
}

func prepareLocalExtensionInstallIfPresent(path string) (preparedExtensionInstall, bool, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return preparedExtensionInstall{}, false, errors.New("extension: install path or registry slug is required")
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return preparedExtensionInstall{}, false, fmt.Errorf("extension: resolve install path %q: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if errors.Is(err, os.ErrNotExist) {
		return preparedExtensionInstall{}, false, nil
	}
	if err != nil {
		return preparedExtensionInstall{}, false, fmt.Errorf("extension: stat install path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return preparedExtensionInstall{}, false, fmt.Errorf("extension: install path %q must be a directory", absPath)
	}

	prepared, err := prepareExtensionInstall(absPath)
	if err != nil {
		return preparedExtensionInstall{}, false, err
	}
	return prepared, true, nil
}

func installPreparedExtension(
	homePaths aghconfig.HomePaths,
	registry localExtensionRegistry,
	prepared preparedExtensionInstall,
	installedAt time.Time,
	allowUnverified bool,
) error {
	if registry == nil {
		return errors.New("extension: registry is required")
	}
	if prepared.Manifest == nil {
		return errors.New("extension: manifest is required")
	}
	if !allowUnverified {
		return extensionpkg.NewExtensionChecksumUnverifiedError(prepared.Manifest.Name, prepared.Path)
	}
	return extensionpkg.InstallLocalManaged(
		homePaths,
		registry,
		prepared.Manifest,
		prepared.Path,
		prepared.Checksum,
		extensionpkg.WithInstallProvenance(extensionpkg.LocalPathProvenance(
			prepared.Manifest,
			prepared.Path,
			prepared.Checksum,
			installedAt,
			allowUnverified,
		)),
	)
}

func localExtensionRecord(
	info extensionpkg.ExtensionInfo,
	now func() time.Time,
	getenv func(string) string,
) ExtensionRecord {
	ext := &extensionpkg.Extension{
		Info: info,
		Status: extensionpkg.ExtensionStatus{
			Name:    info.Name,
			Version: info.Version,
			Source:  info.Source,
			Enabled: info.Enabled,
		},
	}
	if manifest, err := extensionpkg.LoadManifest(filepath.Dir(info.ManifestPath)); err == nil {
		ext.Manifest = manifest
		ext.Status.MissingEnv = manifest.MissingEnv(getenv)
		ext.Status.MissingEnvChecked = len(manifest.RequiresEnv) > 0
	}
	return extensionpkg.DescribeExtension(ext, false, now())
}

func extensionListBundle(items []ExtensionRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Extensions",
		[]string{
			automationNameValue,
			daemonVersionValue,
			sessionTypeValue,
			authoredContextStateValue,
			authoredContextSourceValue,
			"Missing Env",
			extensionCapabilitiesValue,
		},
		"extensions",
		[]string{
			automationNameKey,
			daemonVersionKey,
			extensionTypeKey,
			"state",
			automationSourceKey,
			"missing_env",
			extensionCapabilitiesKey,
		},
		func(item ExtensionRecord) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Version),
				stringOrDash(item.Type),
				stringOrDash(item.State),
				stringOrDash(item.Source),
				stringOrDash(strings.Join(item.MissingEnv, ", ")),
				stringOrDash(strings.Join(item.Capabilities, ", ")),
			}
		},
		func(item ExtensionRecord) []string {
			return []string{
				item.Name,
				item.Version,
				item.Type,
				item.State,
				item.Source,
				strings.Join(item.MissingEnv, "|"),
				strings.Join(item.Capabilities, "|"),
			}
		},
	)
}

func extensionBundle(item ExtensionRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Extension", []keyValue{
				{Label: automationNameValue, Value: stringOrDash(item.Name)},
				{Label: daemonVersionValue, Value: stringOrDash(item.Version)},
				{Label: sessionTypeValue, Value: stringOrDash(item.Type)},
				{Label: authoredContextSourceValue, Value: stringOrDash(item.Source)},
				{Label: extensionEnabledValue, Value: fmt.Sprintf("%t", item.Enabled)},
				{Label: authoredContextStateValue, Value: stringOrDash(item.State)},
				{Label: "Daemon", Value: boolLabel(item.DaemonRunning, "running", "offline")},
				{Label: cliPIDValue, Value: intOrDash(item.PID)},
				{Label: cliUptimeValue, Value: stringOrDash(formatExtensionUptime(item.UptimeSeconds))},
				{
					Label: extensionHealthValue,
					Value: stringOrDash(joinExtensionHealth(item.Health, item.HealthMessage)),
				},
				{Label: extensionCapabilitiesValue, Value: stringOrDash(strings.Join(item.Capabilities, ", "))},
				{Label: "Actions", Value: stringOrDash(strings.Join(item.Actions, ", "))},
				{Label: "Requires Env", Value: stringOrDash(strings.Join(item.RequiresEnv, ", "))},
				{Label: "Missing Env", Value: stringOrDash(strings.Join(item.MissingEnv, ", "))},
				{Label: "Last Error", Value: stringOrDash(item.LastError)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(extensionExtensionKey, []string{
				automationNameKey,
				daemonVersionKey,
				extensionTypeKey,
				automationSourceKey,
				extensionEnabledKey,
				"state",
				"daemon_running",
				cliPIDKey,
				"uptime_seconds",
				extensionHealthKey,
				"last_error",
				extensionCapabilitiesKey,
				"actions",
				"requires_env",
				"missing_env",
			}, []string{
				item.Name,
				item.Version,
				item.Type,
				item.Source,
				fmt.Sprintf("%t", item.Enabled),
				item.State,
				fmt.Sprintf("%t", item.DaemonRunning),
				fmt.Sprintf("%d", item.PID),
				fmt.Sprintf("%d", item.UptimeSeconds),
				joinExtensionHealth(item.Health, item.HealthMessage),
				item.LastError,
				strings.Join(item.Capabilities, "|"),
				strings.Join(item.Actions, "|"),
				strings.Join(item.RequiresEnv, "|"),
				strings.Join(item.MissingEnv, "|"),
			}), nil
		},
	}
}

func formatExtensionUptime(seconds int64) string {
	if seconds <= 0 {
		return ""
	}

	duration := time.Duration(seconds) * time.Second
	if duration < time.Minute {
		return fmt.Sprintf("%ds", seconds)
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func joinExtensionHealth(health string, message string) string {
	if strings.TrimSpace(health) == "" {
		return ""
	}
	if strings.TrimSpace(message) == "" {
		return health
	}
	return health + " (" + message + ")"
}

func extensionProvenanceBundle(item ExtensionProvenanceRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Extension Provenance", []keyValue{
				{Label: extensionMarketplaceSlugValue, Value: stringOrDash(item.Slug)},
				{Label: "Installed From", Value: stringOrDash(item.InstalledFrom)},
				{Label: "Source URL", Value: stringOrDash(item.SourceURL)},
				{Label: "Checksum", Value: stringOrDash(item.ChecksumSHA256)},
				{Label: "Checksum Verified", Value: fmt.Sprintf("%t", item.ChecksumVerified)},
				{Label: "Registry Tier", Value: stringOrDash(item.RegistryTier)},
				{Label: "Allow Unverified", Value: fmt.Sprintf("%t", item.AllowUnverified)},
				{Label: "Installed By", Value: stringOrDash(item.InstalledBy)},
				{Label: "Trust", Value: stringOrDash(extensionTrustDecisionLabel(item.Trust))},
				{Label: "Permissions", Value: stringOrDash(strings.Join(item.Permissions, ", "))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("extension_provenance", []string{
				extensionMarketplaceSlugKey,
				"installed_from",
				"source_url",
				"checksum_sha256",
				"checksum_verified",
				"registry_tier",
				"allow_unverified",
				"installed_by",
				"trust",
				"permissions",
			}, []string{
				item.Slug,
				item.InstalledFrom,
				item.SourceURL,
				item.ChecksumSHA256,
				fmt.Sprintf("%t", item.ChecksumVerified),
				item.RegistryTier,
				fmt.Sprintf("%t", item.AllowUnverified),
				item.InstalledBy,
				extensionTrustDecisionLabel(item.Trust),
				strings.Join(item.Permissions, "|"),
			}), nil
		},
	}
}

func extensionTrustDecisionLabel(trust *contract.ExtensionTrustReportPayload) string {
	if trust == nil {
		return ""
	}
	return strings.TrimSpace(trust.Decision)
}

func boolLabel(value bool, whenTrue string, whenFalse string) string {
	if value {
		return whenTrue
	}
	return whenFalse
}
