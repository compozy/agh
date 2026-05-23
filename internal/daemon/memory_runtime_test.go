package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compozy/agh/internal/memory"
	memcontract "github.com/compozy/agh/internal/memory/contract"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestDaemonMemoryProposalSinkTargetStore(t *testing.T) {
	t.Parallel()

	t.Run("Should normalize workspace-root candidates to the stable workspace identity", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		workspaceRoot := filepath.Join(baseDir, "workspace")
		if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		identity, err := workspacepkg.EnsureIdentity(testutil.Context(t), workspaceRoot)
		if err != nil {
			t.Fatalf("EnsureIdentity() error = %v", err)
		}
		sink := daemonMemoryProposalSink{
			base: memory.NewStore(
				filepath.Join(baseDir, "global", "memory"),
				memory.WithCatalogDatabasePath(filepath.Join(baseDir, "agh.db")),
			),
		}
		candidate := memcontract.Candidate{
			WorkspaceID: "ws-registration",
			Scope:       memcontract.ScopeWorkspace,
			Frontmatter: memcontract.Header{
				Scope: memcontract.ScopeWorkspace,
				Type:  memcontract.TypeProject,
			},
			Metadata: map[string]string{
				"workspace_root": workspaceRoot,
			},
		}

		_, normalized, err := sink.targetStore(testutil.Context(t), candidate)
		if err != nil {
			t.Fatalf("targetStore() error = %v", err)
		}
		if normalized.WorkspaceID != identity.WorkspaceID {
			t.Fatalf(
				"candidate workspace id = %q, want stable identity %q",
				normalized.WorkspaceID,
				identity.WorkspaceID,
			)
		}
	})
}
