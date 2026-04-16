//go:build integration

package daytona

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
)

func TestDaytonaProviderIntegrationFullLifecycle(t *testing.T) {
	apiKey := os.Getenv("DAYTONA_API_KEY")
	if apiKey == "" {
		t.Skip("DAYTONA_API_KEY is required for Daytona provider integration tests")
	}
	snapshot := os.Getenv("DAYTONA_SNAPSHOT")
	image := os.Getenv("DAYTONA_IMAGE")
	if snapshot == "" && image == "" {
		t.Skip("DAYTONA_SNAPSHOT or DAYTONA_IMAGE is required for Daytona provider integration tests")
	}

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "input.txt"), "hello from local")
	provider := NewProvider(WithLogger(nil))
	startupSource := environment.DaytonaStartupSourceImage
	startupRef := image
	if snapshot != "" {
		startupSource = environment.DaytonaStartupSourceSnapshot
		startupRef = snapshot
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	prepared, err := provider.Prepare(ctx, environment.PrepareRequest{
		SessionID:     "integration-daytona",
		WorkspaceID:   "workspace-integration",
		EnvironmentID: "env-integration",
		LocalRootDir:  root,
		Environment: environment.Resolved{
			Profile:        "daytona-integration",
			Backend:        environment.BackendDaytona,
			SyncMode:       environment.SyncModeSessionBidirectional,
			Persistence:    environment.PersistenceTransient,
			RuntimeRootDir: "/home/daytona/agh-integration",
			Daytona: &environment.DaytonaConfig{
				APIURL:        os.Getenv("DAYTONA_API_URL"),
				Image:         image,
				Snapshot:      snapshot,
				StartupSource: startupSource,
				StartupRef:    startupRef,
			},
		},
		AgentCommand: "cat",
		AgentEnv:     []string{"AGH_SESSION_ID=integration-daytona"},
		Permissions:  string(config.PermissionModeApproveAll),
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		if err := provider.Destroy(cleanupCtx, prepared.State); err != nil {
			t.Logf("Destroy() cleanup error = %v", err)
		}
	})

	if _, err := provider.SyncToRuntime(ctx, prepared.State, environment.SyncOptions{
		Reason: environment.SyncReasonStart,
	}); err != nil {
		t.Fatalf("SyncToRuntime() error = %v", err)
	}
	remoteContent, err := prepared.ToolHost.ReadTextFile(ctx, "input.txt")
	if err != nil {
		t.Fatalf("ToolHost.ReadTextFile() error = %v", err)
	}
	if remoteContent != "hello from local" {
		t.Fatalf("remote content = %q, want local content", remoteContent)
	}
	handle, err := prepared.Launcher.Launch(ctx, environment.LaunchSpec{
		Command: "cat",
		Cwd:     prepared.RuntimeRootDir,
		Env:     []string{"AGH_SESSION_ID=integration-daytona"},
	})
	if err != nil {
		t.Fatalf("Launch(cat) error = %v", err)
	}
	if _, err := handle.Stdin().Write([]byte("echo test")); err != nil {
		t.Fatalf("handle.Stdin().Write() error = %v", err)
	}
	if err := handle.Stdin().Close(); err != nil {
		t.Fatalf("handle.Stdin().Close() error = %v", err)
	}
	output, err := io.ReadAll(handle.Stdout())
	if err != nil {
		t.Fatalf("ReadAll(handle.Stdout()) error = %v", err)
	}
	if string(output) != "echo test" {
		t.Fatalf("SSH cat output = %q, want echo test", string(output))
	}
	if err := prepared.ToolHost.WriteTextFile(ctx, "output.txt", "hello from runtime"); err != nil {
		t.Fatalf("ToolHost.WriteTextFile() error = %v", err)
	}
	if _, err := provider.SyncFromRuntime(ctx, prepared.State, environment.SyncOptions{
		Reason: environment.SyncReasonStop,
	}); err != nil {
		t.Fatalf("SyncFromRuntime() error = %v", err)
	}
	assertFileContent(t, filepath.Join(root, "output.txt"), "hello from runtime")
}
