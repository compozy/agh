package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/spf13/cobra"
)

const (
	managedEnvName             = "AGH_MANAGED"
	lifecycleStatusDeferred    = "deferred"
	lifecycleStatusManual      = "manual"
	lifecycleStatusUninstalled = "uninstalled"
)

type managedState struct {
	Managed bool   `json:"managed"`
	Manager string `json:"manager,omitempty"`
}

type lifecycleRecord struct {
	Command        string   `json:"command"`
	Status         string   `json:"status"`
	Managed        bool     `json:"managed"`
	Manager        string   `json:"manager,omitempty"`
	HomeDir        string   `json:"home_dir,omitempty"`
	Message        string   `json:"message"`
	Recommendation string   `json:"recommendation,omitempty"`
	DaemonStopped  bool     `json:"daemon_stopped,omitempty"`
	Removed        []string `json:"removed,omitempty"`
	Purged         bool     `json:"purged,omitempty"`
}

func detectManagedState(deps commandDeps) managedState {
	manager := ""
	if deps.getenv != nil {
		manager = strings.TrimSpace(deps.getenv(managedEnvName))
	}
	return managedState{
		Managed: manager != "",
		Manager: manager,
	}
}

func requireUnmanagedForMutation(deps commandDeps, action string) error {
	state := detectManagedState(deps)
	if !state.Managed {
		return nil
	}
	return fmt.Errorf(
		"cli: AGH is managed by %s; refusing to %s through this binary. %s",
		state.Manager,
		strings.TrimSpace(action),
		managedRecommendation(state.Manager, action),
	)
}

func managedRecommendation(manager string, action string) string {
	normalizedManager := strings.ToLower(strings.TrimSpace(manager))
	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	if normalizedAction == "" {
		normalizedAction = "change AGH"
	}

	switch {
	case strings.Contains(normalizedManager, "brew") || strings.Contains(normalizedManager, "homebrew"):
		if strings.Contains(normalizedAction, "uninstall") {
			return "Use `brew uninstall agh` or the formula name used to install AGH."
		}
		return "Use `brew upgrade agh` or the formula name used to install AGH."
	case strings.Contains(normalizedManager, "scoop"):
		if strings.Contains(normalizedAction, "uninstall") {
			return "Use `scoop uninstall agh`."
		}
		return "Use `scoop update agh`."
	case strings.Contains(normalizedManager, "nix"):
		return "Update or remove AGH through your Nix configuration and run `nixos-rebuild switch`."
	case strings.Contains(normalizedManager, "apt"), strings.Contains(normalizedManager, "deb"):
		if strings.Contains(normalizedAction, "uninstall") {
			return "Use `sudo apt remove agh` or the package name used to install AGH."
		}
		return "Use `sudo apt update && sudo apt upgrade agh` or the package name used to install AGH."
	case strings.Contains(normalizedManager, "dnf"), strings.Contains(normalizedManager, "rpm"):
		if strings.Contains(normalizedAction, "uninstall") {
			return "Use `sudo dnf remove agh` or the package name used to install AGH."
		}
		return "Use `sudo dnf upgrade agh` or the package name used to install AGH."
	default:
		return "Use the package manager that set AGH_MANAGED instead of mutating this install directly."
	}
}

func newUpdateCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Report how to update the AGH binary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			state := detectManagedState(deps)
			record := lifecycleRecord{
				Command: "update",
				Managed: state.Managed,
				Manager: state.Manager,
			}
			if state.Managed {
				record.Status = lifecycleStatusDeferred
				record.Message = "AGH is managed by an external package manager; no local update was performed."
				record.Recommendation = managedRecommendation(state.Manager, "update AGH")
				return writeCommandOutput(cmd, lifecycleBundle("Update", record))
			}

			homePaths, err := deps.resolveHome()
			if err != nil {
				return err
			}
			record.HomeDir = homePaths.HomeDir
			record.Status = lifecycleStatusManual
			record.Message = "No in-place updater is configured for this unmanaged AGH binary; no files were changed."
			record.Recommendation = "Install a newer release archive, rerun `go install`, or rebuild from source."
			return writeCommandOutput(cmd, lifecycleBundle("Update", record))
		},
	}
}

func newUninstallCommand(deps commandDeps) *cobra.Command {
	var (
		purge bool
		force bool
	)

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop AGH and remove runtime launch artifacts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			state := detectManagedState(deps)
			if state.Managed {
				record := lifecycleRecord{
					Command:        "uninstall",
					Managed:        state.Managed,
					Manager:        state.Manager,
					Status:         lifecycleStatusDeferred,
					Message:        "AGH is managed by an external package manager; no local uninstall changes were made.",
					Recommendation: managedRecommendation(state.Manager, "uninstall AGH"),
				}
				return writeCommandOutput(cmd, lifecycleBundle("Uninstall", record))
			}

			if purge && !force {
				return errors.New("cli: --purge requires --force to remove AGH home data")
			}

			runtime, err := loadRuntimeContext(deps)
			if err != nil {
				return err
			}

			record := lifecycleRecord{
				Command: "uninstall",
				HomeDir: runtime.HomePaths.HomeDir,
				Managed: state.Managed,
				Manager: state.Manager,
			}

			stopped, err := stopDaemonForUninstall(cmd.Context(), deps, runtime)
			if err != nil {
				return err
			}
			record.DaemonStopped = stopped

			removed, err := removeRuntimeArtifacts(runtime.HomePaths)
			if err != nil {
				return err
			}
			record.Removed = removed

			if purge {
				if err := os.RemoveAll(runtime.HomePaths.HomeDir); err != nil {
					return fmt.Errorf("cli: purge AGH home %q: %w", runtime.HomePaths.HomeDir, err)
				}
				record.Purged = true
			}

			record.Status = lifecycleStatusUninstalled
			record.Message = "AGH runtime launch artifacts were removed; persistent data was preserved."
			if record.Purged {
				record.Message = "AGH runtime launch artifacts and AGH home data were removed."
			}
			return writeCommandOutput(cmd, lifecycleBundle("Uninstall", record))
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "Remove the AGH home directory after stopping runtime artifacts")
	cmd.Flags().BoolVar(&force, "force", false, "Confirm destructive purge of AGH home data")
	return cmd
}

func stopDaemonForUninstall(ctx context.Context, deps commandDeps, runtime *runtimeContext) (bool, error) {
	info, running, err := daemonInfo(runtime.HomePaths, deps)
	if err != nil {
		return false, err
	}
	if !running {
		return false, nil
	}
	if err := deps.signalProcess(info.PID, syscall.SIGTERM); err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return true, nil
		}
		return false, fmt.Errorf("cli: stop daemon for uninstall: %w", err)
	}
	if _, err := waitForDaemonStop(ctx, deps, runtime, info); err != nil {
		return false, err
	}
	return true, nil
}

func removeRuntimeArtifacts(homePaths aghconfig.HomePaths) ([]string, error) {
	candidates := []string{
		homePaths.DaemonSocket,
		homePaths.DaemonLock,
		homePaths.DaemonInfo,
	}
	removed := make([]string, 0, len(candidates))
	for _, path := range candidates {
		deleted, err := removeFileIfExists(path)
		if err != nil {
			return nil, err
		}
		if deleted {
			removed = append(removed, path)
		}
	}
	return removed, nil
}

func removeFileIfExists(path string) (bool, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false, nil
	}
	err := os.Remove(trimmed)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("cli: remove runtime artifact %q: %w", trimmed, err)
	}
}

func lifecycleBundle(title string, record lifecycleRecord) outputBundle {
	rows := []keyValue{
		{Label: "Status", Value: stringOrDash(record.Status)},
		{Label: "Managed", Value: fmt.Sprintf("%t", record.Managed)},
		{Label: "Manager", Value: stringOrDash(record.Manager)},
		{Label: "Home", Value: stringOrDash(record.HomeDir)},
		{Label: "Message", Value: stringOrDash(record.Message)},
	}
	if record.Recommendation != "" {
		rows = append(rows, keyValue{Label: "Recommendation", Value: record.Recommendation})
	}
	if record.DaemonStopped {
		rows = append(rows, keyValue{Label: "Daemon Stopped", Value: "true"})
	}
	if len(record.Removed) > 0 {
		rows = append(rows, keyValue{Label: "Removed", Value: strings.Join(record.Removed, ", ")})
	}
	rows = append(rows, keyValue{Label: "Purged", Value: fmt.Sprintf("%t", record.Purged)})

	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanSection(title, rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				strings.ToLower(title),
				[]string{
					"command",
					"status",
					"managed",
					"manager",
					"home_dir",
					"message",
					"recommendation",
					"daemon_stopped",
					"removed",
					"purged",
				},
				[]string{
					record.Command,
					record.Status,
					fmt.Sprintf("%t", record.Managed),
					record.Manager,
					record.HomeDir,
					record.Message,
					record.Recommendation,
					fmt.Sprintf("%t", record.DaemonStopped),
					strings.Join(record.Removed, ", "),
					fmt.Sprintf("%t", record.Purged),
				},
			), nil
		},
	}
}

func ensureWriteTargetParent(target aghconfig.WriteTarget) error {
	path := strings.TrimSpace(target.Path())
	if path == "" {
		return errors.New("cli: config write target path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cli: create config write target directory %q: %w", filepath.Dir(path), err)
	}
	return nil
}
