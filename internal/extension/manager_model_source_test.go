package extensionpkg

import (
	"testing"

	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerListModelSourceRows(t *testing.T) {
	t.Run("Should call subprocess models list", func(t *testing.T) {
		withDaemonVersion(t, "0.5.0")
		env := newRegistryTestEnv(t)
		fixture := createManagerTestExtension(t, managerTestManifest("ext-models", managerManifestOptions{
			command:      helperCommand(t),
			args:         helperArgs(),
			withEnv:      helperEnv("model_source_success", ""),
			capabilities: []string{extensionprotocol.CapabilityProvideModelSource},
		}), nil)
		installManagerFixture(t, env.registry, fixture, SourceUser, true)

		manager := NewManager(env.registry)
		if err := manager.Start(testutil.Context(t)); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Stop(testutil.Context(t)); err != nil {
				t.Fatalf("Stop() cleanup error = %v", err)
			}
		})

		rows, err := manager.ListModelSourceRows(
			testutil.Context(t),
			"ext-models",
			extensioncontract.ModelSourceListParams{ProviderID: "codex", Refresh: true},
		)
		if err != nil {
			t.Fatalf("ListModelSourceRows() error = %v, want nil", err)
		}
		if len(rows) != 1 || rows[0].ModelID != "subprocess-model" || rows[0].SourceID != "extension:ext-models" {
			t.Fatalf("ListModelSourceRows() = %#v, want subprocess model source row", rows)
		}
	})

	t.Run("Should deny missing model source capability", func(t *testing.T) {
		withDaemonVersion(t, "0.5.0")
		env := newRegistryTestEnv(t)
		fixture := createManagerTestExtension(t, managerTestManifest("ext-no-models", managerManifestOptions{
			command:      helperCommand(t),
			args:         helperArgs(),
			withEnv:      helperEnv("model_source_success", ""),
			capabilities: []string{"memory.backend"},
		}), nil)
		installManagerFixture(t, env.registry, fixture, SourceUser, true)

		manager := NewManager(env.registry)
		if err := manager.Start(testutil.Context(t)); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Stop(testutil.Context(t)); err != nil {
				t.Fatalf("Stop() cleanup error = %v", err)
			}
		})

		_, err := manager.ListModelSourceRows(
			testutil.Context(t),
			"ext-no-models",
			extensioncontract.ModelSourceListParams{ProviderID: "codex"},
		)
		if err == nil {
			t.Fatal("ListModelSourceRows() error = nil, want denied service method")
		}
	})
}
