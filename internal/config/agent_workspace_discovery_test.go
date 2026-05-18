package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWorkspaceAgentDefsRejectsMismatchedDirectoryNameContract(t *testing.T) {
	t.Parallel()

	t.Run("Should reject agent files whose declared name differs from the containing directory", func(t *testing.T) {
		t.Parallel()

		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		if err := EnsureHomeLayout(homePaths); err != nil {
			t.Fatalf("EnsureHomeLayout() error = %v", err)
		}

		root := t.TempDir()
		writeAgentDefinition(
			t,
			filepath.Join(root, DirName, AgentsDirName, "not-reviewer", agentDefName),
			"reviewer",
			"claude",
			"workspace-mismatch",
		)
		writeAgentDefinition(
			t,
			filepath.Join(homePaths.AgentsDir, "reviewer", agentDefName),
			"reviewer",
			"claude",
			"global-review",
		)

		_, err = LoadWorkspaceAgentDefs(root, nil, homePaths)
		if err == nil {
			t.Fatal("LoadWorkspaceAgentDefs() error = nil, want mismatched name failure")
		}
		if !strings.Contains(err.Error(), "not-reviewer") || !strings.Contains(err.Error(), "reviewer") {
			t.Fatalf("LoadWorkspaceAgentDefs() error = %v, want directory and declared name context", err)
		}
	})
}
