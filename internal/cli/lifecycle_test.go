package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestInstallUpdateAndUninstallReportManagedState(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	deps.getenv = func(key string) string {
		if key == managedEnvName {
			return "homebrew"
		}
		return ""
	}
	deps.runInstallWizard = func(context.Context, installWizardInput) (installWizardSelection, error) {
		return installWizardSelection{Provider: "claude", Model: "claude-sonnet-4-6"}, nil
	}

	installOut, _, err := executeRootCommand(t, deps, "install", "-o", "json")
	if err != nil {
		t.Fatalf("managed install error = %v", err)
	}
	var install installRecord
	if err := json.Unmarshal([]byte(installOut), &install); err != nil {
		t.Fatalf("json.Unmarshal(install) error = %v", err)
	}
	if !install.Managed || install.Manager != "homebrew" {
		t.Fatalf("install managed state = %#v, want homebrew", install)
	}

	updateOut, _, err := executeRootCommand(t, deps, "update", "-o", "json")
	if err != nil {
		t.Fatalf("managed update error = %v", err)
	}
	var update lifecycleRecord
	if err := json.Unmarshal([]byte(updateOut), &update); err != nil {
		t.Fatalf("json.Unmarshal(update) error = %v", err)
	}
	if update.Status != lifecycleStatusDeferred || !update.Managed || !strings.Contains(update.Recommendation, "brew") {
		t.Fatalf("managed update record = %#v, want deferred brew recommendation", update)
	}

	deps.resolveHome = func() (aghconfig.HomePaths, error) {
		return aghconfig.HomePaths{}, errors.New("broken local home")
	}

	updateOut, _, err = executeRootCommand(t, deps, "update", "-o", "json")
	if err != nil {
		t.Fatalf("managed update with broken local home error = %v", err)
	}
	if err := json.Unmarshal([]byte(updateOut), &update); err != nil {
		t.Fatalf("json.Unmarshal(update with broken home) error = %v", err)
	}
	if update.Status != lifecycleStatusDeferred || update.HomeDir != "" {
		t.Fatalf("managed update with broken local home = %#v, want deferred without home", update)
	}

	uninstallOut, _, err := executeRootCommand(t, deps, "uninstall", "--purge", "-o", "json")
	if err != nil {
		t.Fatalf("managed uninstall error = %v", err)
	}
	var uninstall lifecycleRecord
	if err := json.Unmarshal([]byte(uninstallOut), &uninstall); err != nil {
		t.Fatalf("json.Unmarshal(uninstall) error = %v", err)
	}
	if uninstall.Status != lifecycleStatusDeferred || !uninstall.Managed ||
		!strings.Contains(uninstall.Recommendation, "brew") {
		t.Fatalf("managed uninstall record = %#v, want deferred brew recommendation", uninstall)
	}
}

func TestUninstallRemovesRuntimeArtifactsIdempotentlyAndRequiresForceForPurge(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	homePaths, err := deps.resolveHome()
	if err != nil {
		t.Fatalf("resolveHome() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	deps.processAlive = func(int) bool { return false }
	for _, path := range []string{homePaths.DaemonSocket, homePaths.DaemonLock} {
		writeFile(t, path, "runtime artifact")
	}
	writeFile(t, homePaths.DaemonInfo, `{"pid":999999,"port":0,"started_at":"2026-04-03T12:00:00Z"}`)

	out, _, err := executeRootCommand(t, deps, "uninstall", "-o", "json")
	if err != nil {
		t.Fatalf("uninstall error = %v", err)
	}
	var record lifecycleRecord
	if err := json.Unmarshal([]byte(out), &record); err != nil {
		t.Fatalf("json.Unmarshal(uninstall) error = %v", err)
	}
	if record.Status != lifecycleStatusUninstalled || record.Purged {
		t.Fatalf("uninstall record = %#v, want uninstalled without purge", record)
	}
	if len(record.Removed) != 3 {
		t.Fatalf("uninstall removed = %#v, want three runtime artifacts", record.Removed)
	}
	for _, path := range []string{homePaths.DaemonSocket, homePaths.DaemonLock, homePaths.DaemonInfo} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("runtime artifact %q still exists or stat error = %v", path, err)
		}
	}
	if _, err := os.Stat(homePaths.HomeDir); err != nil {
		t.Fatalf("home dir stat after uninstall error = %v", err)
	}

	secondOut, _, err := executeRootCommand(t, deps, "uninstall", "-o", "json")
	if err != nil {
		t.Fatalf("second uninstall error = %v", err)
	}
	var second lifecycleRecord
	if err := json.Unmarshal([]byte(secondOut), &second); err != nil {
		t.Fatalf("json.Unmarshal(second uninstall) error = %v", err)
	}
	if second.Status != lifecycleStatusUninstalled || len(second.Removed) != 0 {
		t.Fatalf("second uninstall record = %#v, want idempotent no-op removal", second)
	}

	if _, _, err := executeRootCommand(t, deps, "uninstall", "--purge"); err == nil {
		t.Fatal("uninstall --purge error = nil, want --force requirement")
	}
	purgeOut, _, err := executeRootCommand(t, deps, "uninstall", "--purge", "--force", "-o", "json")
	if err != nil {
		t.Fatalf("uninstall --purge --force error = %v", err)
	}
	var purge lifecycleRecord
	if err := json.Unmarshal([]byte(purgeOut), &purge); err != nil {
		t.Fatalf("json.Unmarshal(purge) error = %v", err)
	}
	if !purge.Purged {
		t.Fatalf("purge record = %#v, want purged", purge)
	}
	if _, err := os.Stat(homePaths.HomeDir); !os.IsNotExist(err) {
		t.Fatalf("home dir still exists after purge or stat error = %v", err)
	}
}

func TestUpdateReportsManualPathForUnmanagedBinary(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	out, _, err := executeRootCommand(t, deps, "update", "-o", "json")
	if err != nil {
		t.Fatalf("unmanaged update error = %v", err)
	}
	var record lifecycleRecord
	if err := json.Unmarshal([]byte(out), &record); err != nil {
		t.Fatalf("json.Unmarshal(update) error = %v", err)
	}
	if record.Status != lifecycleStatusManual || record.Managed ||
		!strings.Contains(record.Recommendation, "go install") {
		t.Fatalf("unmanaged update record = %#v, want manual go install recommendation", record)
	}
}

func TestConfigEditUsesEditorAndValidatesResult(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	homePaths, err := deps.resolveHome()
	if err != nil {
		t.Fatalf("resolveHome() error = %v", err)
	}
	editorPath := filepath.Join(t.TempDir(), "editor with spaces.sh")
	writeFile(
		t,
		editorPath,
		"#!/bin/sh\nfor target do :; done\nprintf '\\n[defaults]\\nprovider = \"claude\"\\n' >> \"$target\"\n",
	)
	if err := os.Chmod(editorPath, 0o700); err != nil {
		t.Fatalf("chmod editor error = %v", err)
	}
	deps.getenv = func(key string) string {
		if key == "EDITOR" {
			return "'" + editorPath + "' --append"
		}
		return ""
	}

	if _, _, err := executeRootCommand(t, deps, "config", "edit", "-o", "json"); err != nil {
		t.Fatalf("config edit error = %v", err)
	}
	cfg, err := aghconfig.LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig(after edit) error = %v", err)
	}
	if cfg.Defaults.Provider != "claude" {
		t.Fatalf("Defaults.Provider after edit = %q, want claude", cfg.Defaults.Provider)
	}
}

func TestUninstallContinuesWhenRunningDaemonAlreadyExited(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	homePaths, err := deps.resolveHome()
	if err != nil {
		t.Fatalf("resolveHome() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	for _, path := range []string{homePaths.DaemonSocket, homePaths.DaemonLock} {
		writeFile(t, path, "runtime artifact")
	}
	writeFile(t, homePaths.DaemonInfo, `{"pid":999999,"port":0,"started_at":"2026-04-03T12:00:00Z"}`)
	deps.processAlive = func(int) bool { return true }
	deps.signalProcess = func(int, syscall.Signal) error { return os.ErrProcessDone }

	out, _, err := executeRootCommand(t, deps, "uninstall", "-o", "json")
	if err != nil {
		t.Fatalf("uninstall after already-exited daemon error = %v", err)
	}
	var record lifecycleRecord
	if err := json.Unmarshal([]byte(out), &record); err != nil {
		t.Fatalf("json.Unmarshal(uninstall) error = %v", err)
	}
	if record.Status != lifecycleStatusUninstalled || !record.DaemonStopped || len(record.Removed) != 3 {
		t.Fatalf("uninstall record = %#v, want stopped uninstall with artifacts removed", record)
	}
}
