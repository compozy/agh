package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestAssemblerAssemble(t *testing.T) {
	t.Parallel()

	t.Run("global index only", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeGlobalIndex(t, "- [Global](global.md) - global note")

		got := env.assemble(t)
		if !strings.Contains(got, "## Global MEMORY.md Index") {
			t.Fatalf("assembled prompt missing global section: %q", got)
		}
		if strings.Contains(got, "## Workspace MEMORY.md Index") {
			t.Fatalf("assembled prompt unexpectedly included workspace section: %q", got)
		}
	})

	t.Run("workspace index only", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeWorkspaceIndex(t, "- [Workspace](workspace.md) - workspace note")

		got := env.assemble(t)
		if !strings.Contains(got, "## Workspace MEMORY.md Index") {
			t.Fatalf("assembled prompt missing workspace section: %q", got)
		}
		if strings.Contains(got, "## Global MEMORY.md Index") {
			t.Fatalf("assembled prompt unexpectedly included global section: %q", got)
		}
	})

	t.Run("both indexes", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeGlobalIndex(t, "- [Global](global.md) - global note")
		env.writeWorkspaceIndex(t, "- [Workspace](workspace.md) - workspace note")

		got := env.assemble(t)
		if !strings.Contains(got, "## Global MEMORY.md Index") || !strings.Contains(got, "## Workspace MEMORY.md Index") {
			t.Fatalf("assembled prompt missing expected sections: %q", got)
		}
	})

	t.Run("empty indexes returns original prompt", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		got := env.assemble(t)
		if got != env.agent.Prompt {
			t.Fatalf("assembled prompt = %q, want original prompt %q", got, env.agent.Prompt)
		}
	})

	t.Run("includes taxonomy instructions", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeGlobalIndex(t, "- [Global](global.md) - global note")

		got := env.assemble(t)
		for _, want := range []string{"## Memory Taxonomy", "`user`", "`feedback`", "`project`", "`reference`"} {
			if !strings.Contains(got, want) {
				t.Fatalf("assembled prompt missing taxonomy content %q: %q", want, got)
			}
		}
	})

	t.Run("includes agh memory command reference", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeGlobalIndex(t, "- [Global](global.md) - global note")

		got := env.assemble(t)
		for _, want := range []string{"## Memory Commands", "`agh memory list`", "`agh memory read <filename>`", "`agh memory write <filename> --type <type> --description <desc> --content <content>`"} {
			if !strings.Contains(got, want) {
				t.Fatalf("assembled prompt missing command reference %q: %q", want, got)
			}
		}
	})

	t.Run("includes staleness policy", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeGlobalIndex(t, "- [Global](global.md) - global note")

		got := env.assemble(t)
		if !strings.Contains(got, "## Staleness Policy") || !strings.Contains(got, "Memories older than 1 day should be verified") {
			t.Fatalf("assembled prompt missing staleness policy: %q", got)
		}
	})

	t.Run("memory context before agent prompt", func(t *testing.T) {
		t.Parallel()

		env := newAssemblerTestEnv(t)
		env.writeGlobalIndex(t, "- [Global](global.md) - global note")

		got := env.assemble(t)
		memoryIndex := strings.Index(got, "# Persistent Memory")
		agentIndex := strings.Index(got, env.agent.Prompt)
		if memoryIndex < 0 || agentIndex < 0 {
			t.Fatalf("assembled prompt missing expected components: %q", got)
		}
		if memoryIndex >= agentIndex {
			t.Fatalf("memory context index = %d, agent prompt index = %d, want memory before agent prompt", memoryIndex, agentIndex)
		}
	})
}

type assemblerTestEnv struct {
	store     *Store
	assembler *Assembler
	workspace string
	agent     aghconfig.AgentDef
}

func newAssemblerTestEnv(t *testing.T) assemblerTestEnv {
	t.Helper()

	baseDir := t.TempDir()
	workspace := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}

	store := NewStore(filepath.Join(baseDir, "home", "memory"))
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("Store.EnsureDirs() error = %v", err)
	}

	return assemblerTestEnv{
		store:     store,
		assembler: NewAssembler(store),
		workspace: workspace,
		agent: aghconfig.AgentDef{
			Name:     "coder",
			Provider: "claude",
			Prompt:   "You are a coding assistant.",
		},
	}
}

func (e assemblerTestEnv) assemble(t *testing.T) string {
	t.Helper()

	got, err := e.assembler.Assemble(context.Background(), e.agent, e.workspace)
	if err != nil {
		t.Fatalf("Assembler.Assemble() error = %v", err)
	}
	return got
}

func (e assemblerTestEnv) writeGlobalIndex(t *testing.T, content string) {
	t.Helper()
	writeAssemblerFileForTest(t, filepath.Join(e.store.globalDir, indexFilename), content)
}

func (e assemblerTestEnv) writeWorkspaceIndex(t *testing.T, content string) {
	t.Helper()
	writeAssemblerFileForTest(t, filepath.Join(e.store.ForWorkspace(e.workspace).workspaceDir, indexFilename), content)
}

func writeAssemblerFileForTest(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
