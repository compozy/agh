package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/spf13/cobra"
)

type preparedExtensionInstall struct {
	Path     string
	Manifest *extensionpkg.Manifest
	Checksum string
}

type localExtensionRegistry interface {
	Install(manifest *extensionpkg.Manifest, path string, checksum string) error
	List() ([]extensionpkg.ExtensionInfo, error)
	Get(name string) (*extensionpkg.ExtensionInfo, error)
	Enable(name string) error
	Disable(name string) error
}

func newExtensionCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: "Manage AGH extensions",
	}

	cmd.AddCommand(newExtensionListCommand(deps))
	cmd.AddCommand(newExtensionInstallCommand(deps))
	cmd.AddCommand(newExtensionEnableCommand(deps))
	cmd.AddCommand(newExtensionDisableCommand(deps))
	cmd.AddCommand(newExtensionStatusCommand(deps))
	return cmd
}

func newExtensionListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
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
	return &cobra.Command{
		Use:   "install <path>",
		Short: "Install a local extension directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prepared, err := prepareExtensionInstall(args[0])
			if err != nil {
				return err
			}

			item, err := installExtension(cmd.Context(), deps, prepared)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionBundle(item))
		},
	}
}

func newExtensionEnableCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable an installed extension",
		Args:  cobra.ExactArgs(1),
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
		Use:   "disable <name>",
		Short: "Disable an installed extension",
		Args:  cobra.ExactArgs(1),
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := extensionStatus(cmd.Context(), deps, args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, extensionBundle(item))
		},
	}
}

func loadExtensionRecords(ctx context.Context, deps commandDeps) ([]ExtensionRecord, error) {
	client, running, _, err := extensionClientIfRunning(deps)
	if err != nil {
		return nil, err
	}
	if running {
		return client.ListExtensions(ctx)
	}

	return withLocalExtensionRegistry(ctx, deps, func(runtime runtimeContext, registry localExtensionRegistry) ([]ExtensionRecord, error) {
		infos, err := registry.List()
		if err != nil {
			return nil, err
		}

		items := make([]ExtensionRecord, 0, len(infos))
		for _, info := range infos {
			items = append(items, localExtensionRecord(info, deps.now))
		}
		return items, nil
	})
}

func installExtension(ctx context.Context, deps commandDeps, prepared preparedExtensionInstall) (ExtensionRecord, error) {
	client, running, _, err := extensionClientIfRunning(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	if running {
		return client.InstallExtension(ctx, InstallExtensionRequest{
			Path:     prepared.Path,
			Checksum: prepared.Checksum,
		})
	}

	return withLocalExtensionRegistry(ctx, deps, func(runtime runtimeContext, registry localExtensionRegistry) (ExtensionRecord, error) {
		if err := installPreparedExtension(registry, prepared); err != nil {
			return ExtensionRecord{}, err
		}
		info, err := registry.Get(prepared.Manifest.Name)
		if err != nil {
			return ExtensionRecord{}, err
		}
		return localExtensionRecord(*info, deps.now), nil
	})
}

func mutateExtensionEnabled(ctx context.Context, deps commandDeps, name string, enabled bool) (ExtensionRecord, error) {
	client, running, _, err := extensionClientIfRunning(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	if running {
		if enabled {
			return client.EnableExtension(ctx, name)
		}
		return client.DisableExtension(ctx, name)
	}

	return withLocalExtensionRegistry(ctx, deps, func(runtime runtimeContext, registry localExtensionRegistry) (ExtensionRecord, error) {
		if enabled {
			if err := registry.Enable(name); err != nil {
				return ExtensionRecord{}, err
			}
		} else {
			if err := registry.Disable(name); err != nil {
				return ExtensionRecord{}, err
			}
		}

		info, err := registry.Get(name)
		if err != nil {
			return ExtensionRecord{}, err
		}
		return localExtensionRecord(*info, deps.now), nil
	})
}

func extensionStatus(ctx context.Context, deps commandDeps, name string) (ExtensionRecord, error) {
	client, running, _, err := extensionClientIfRunning(deps)
	if err != nil {
		return ExtensionRecord{}, err
	}
	if running {
		return client.ExtensionStatus(ctx, name)
	}

	return withLocalExtensionRegistry(ctx, deps, func(runtime runtimeContext, registry localExtensionRegistry) (ExtensionRecord, error) {
		info, err := registry.Get(name)
		if err != nil {
			return ExtensionRecord{}, err
		}
		return localExtensionRecord(*info, deps.now), nil
	})
}

func extensionClientIfRunning(deps commandDeps) (DaemonClient, bool, runtimeContext, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, false, runtimeContext{}, err
	}

	info, running, err := daemonInfo(runtime.HomePaths, deps)
	if err != nil {
		return nil, false, runtimeContext{}, err
	}
	if !running {
		return nil, false, runtime, nil
	}
	if info == (aghdaemon.Info{}) {
		return nil, false, runtime, nil
	}

	client, _, err := clientFromDeps(deps)
	if err != nil {
		return nil, false, runtimeContext{}, err
	}
	return client, true, runtime, nil
}

func withLocalExtensionRegistry[T any](ctx context.Context, deps commandDeps, fn func(runtime runtimeContext, registry localExtensionRegistry) (T, error)) (result T, err error) {
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

func installPreparedExtension(registry localExtensionRegistry, prepared preparedExtensionInstall) error {
	if registry == nil {
		return errors.New("extension: registry is required")
	}
	if prepared.Manifest == nil {
		return errors.New("extension: manifest is required")
	}
	return registry.Install(prepared.Manifest, prepared.Path, prepared.Checksum)
}

func localExtensionRecord(info extensionpkg.ExtensionInfo, now func() time.Time) ExtensionRecord {
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
	}
	return extensionpkg.DescribeExtension(ext, false, now())
}

func extensionListBundle(items []ExtensionRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Extensions",
		[]string{"Name", "Version", "Type", "State", "Source", "Capabilities"},
		"extensions",
		[]string{"name", "version", "type", "state", "source", "capabilities"},
		func(item ExtensionRecord) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Version),
				stringOrDash(item.Type),
				stringOrDash(item.State),
				stringOrDash(item.Source),
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
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Version", Value: stringOrDash(item.Version)},
				{Label: "Type", Value: stringOrDash(item.Type)},
				{Label: "Source", Value: stringOrDash(item.Source)},
				{Label: "Enabled", Value: fmt.Sprintf("%t", item.Enabled)},
				{Label: "State", Value: stringOrDash(item.State)},
				{Label: "Daemon", Value: boolLabel(item.DaemonRunning, "running", "offline")},
				{Label: "PID", Value: intOrDash(item.PID)},
				{Label: "Uptime", Value: stringOrDash(formatExtensionUptime(item.UptimeSeconds))},
				{Label: "Health", Value: stringOrDash(joinExtensionHealth(item.Health, item.HealthMessage))},
				{Label: "Capabilities", Value: stringOrDash(strings.Join(item.Capabilities, ", "))},
				{Label: "Actions", Value: stringOrDash(strings.Join(item.Actions, ", "))},
				{Label: "Last Error", Value: stringOrDash(item.LastError)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("extension", []string{
				"name", "version", "type", "source", "enabled", "state", "daemon_running", "pid", "uptime_seconds", "health", "last_error", "capabilities", "actions",
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

func boolLabel(value bool, whenTrue string, whenFalse string) string {
	if value {
		return whenTrue
	}
	return whenFalse
}
