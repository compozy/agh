package e2e

import "testing"

func TestPrepareRuntimeLayoutCreatesIsolatedPaths(t *testing.T) {
	t.Parallel()

	first := prepareRuntimeLayout(t, RuntimeHarnessOptions{})
	second := prepareRuntimeLayout(t, RuntimeHarnessOptions{})

	if first.HomePaths.HomeDir == second.HomePaths.HomeDir {
		t.Fatalf("first.HomePaths.HomeDir = %q, want different isolated home", first.HomePaths.HomeDir)
	}
	if first.HomePaths.DatabaseFile == second.HomePaths.DatabaseFile {
		t.Fatalf(
			"first.HomePaths.DatabaseFile = %q, want different isolated database path",
			first.HomePaths.DatabaseFile,
		)
	}
	if first.WorkspaceRoot == second.WorkspaceRoot {
		t.Fatalf("first.WorkspaceRoot = %q, want different isolated workspace path", first.WorkspaceRoot)
	}
	if first.Artifacts.RootDir() == second.Artifacts.RootDir() {
		t.Fatalf("first.Artifacts.RootDir() = %q, want different isolated artifact path", first.Artifacts.RootDir())
	}
}

func TestPrepareRuntimeLayoutEnablesNetworkOnlyWhenRequested(t *testing.T) {
	t.Parallel()

	disabled := prepareRuntimeLayout(t, RuntimeHarnessOptions{})
	if disabled.Config.Network.Enabled {
		t.Fatal("disabled.Config.Network.Enabled = true, want false by default")
	}

	enabled := prepareRuntimeLayout(t, RuntimeHarnessOptions{EnableNetwork: true})
	if !enabled.Config.Network.Enabled {
		t.Fatal("enabled.Config.Network.Enabled = false, want true when EnableNetwork is requested")
	}
}
