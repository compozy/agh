package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills/bundled"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestComposedAssemblerAssemble(t *testing.T) {
	t.Parallel()

	t.Run("zero providers returns trimmed base prompt", func(t *testing.T) {
		t.Parallel()

		assembler := NewComposedAssembler()
		got := assemblePrompt(t, assembler, testPromptAgent("  Base prompt.\n"), t.TempDir())

		if got != "Base prompt." {
			t.Fatalf("Assemble() = %q, want %q", got, "Base prompt.")
		}
	})

	t.Run("prepend provider renders before base prompt", func(t *testing.T) {
		t.Parallel()

		assembler := NewComposedAssembler(
			WithPrependPromptProviders(staticPromptProvider("# Memory section")),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
		want := "# Memory section\n\nBase prompt."
		if got != want {
			t.Fatalf("Assemble() = %q, want %q", got, want)
		}
	})

	t.Run("append provider renders after base prompt", func(t *testing.T) {
		t.Parallel()

		assembler := NewComposedAssembler(
			WithAppendPromptProviders(staticPromptProvider("<available-skills />")),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
		want := "Base prompt.\n\n<available-skills />"
		if got != want {
			t.Fatalf("Assemble() = %q, want %q", got, want)
		}
	})

	t.Run("prepend and append providers preserve ordering", func(t *testing.T) {
		t.Parallel()

		assembler := NewComposedAssembler(
			WithPrependPromptProviders(staticPromptProvider("# Memory section")),
			WithAppendPromptProviders(staticPromptProvider("<available-skills />")),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
		want := "# Memory section\n\nBase prompt.\n\n<available-skills />"
		if got != want {
			t.Fatalf("Assemble() = %q, want %q", got, want)
		}
	})

	t.Run("nil providers are skipped", func(t *testing.T) {
		t.Parallel()

		var nilProvider session.PromptProvider
		assembler := NewComposedAssembler(
			WithPrependPromptProviders(nilProvider, staticPromptProvider("# Memory section")),
			WithAppendPromptProviders(nilProvider, staticPromptProvider("<available-skills />")),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
		want := "# Memory section\n\nBase prompt.\n\n<available-skills />"
		if got != want {
			t.Fatalf("Assemble() = %q, want %q", got, want)
		}
	})

	t.Run("provider errors are returned", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("boom")
		assembler := NewComposedAssembler(
			WithAppendPromptProviders(errorPromptProvider{err: wantErr}),
		)
		workspace := testResolvedWorkspace(t.TempDir())

		_, err := assembler.Assemble(
			context.Background(),
			testPromptAgent("Base prompt."),
			&workspace,
		)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Assemble() error = %v, want error wrapping %v", err, wantErr)
		}
	})

	t.Run("empty provider sections do not add whitespace", func(t *testing.T) {
		t.Parallel()

		assembler := NewComposedAssembler(
			WithPrependPromptProviders(staticPromptProvider("   \n\t")),
			WithAppendPromptProviders(staticPromptProvider(" \n ")),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
		if got != "Base prompt." {
			t.Fatalf("Assemble() = %q, want %q", got, "Base prompt.")
		}
	})

	t.Run("workspace is passed to all providers", func(t *testing.T) {
		t.Parallel()

		prepend := &recordingPromptProvider{section: "# Memory section"}
		appendProvider := &recordingPromptProvider{section: "<available-skills />"}
		workspace := filepath.Join(t.TempDir(), "workspace")

		assembler := NewComposedAssembler(
			WithPrependPromptProviders(prepend),
			WithAppendPromptProviders(appendProvider),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), workspace)
		want := "# Memory section\n\nBase prompt.\n\n<available-skills />"
		if got != want {
			t.Fatalf("Assemble() = %q, want %q", got, want)
		}
		if len(prepend.workspaces) != 1 || prepend.workspaces[0] != workspace {
			t.Fatalf("prepend provider workspaces = %v, want [%q]", prepend.workspaces, workspace)
		}
		if len(appendProvider.workspaces) != 1 || appendProvider.workspaces[0] != workspace {
			t.Fatalf("append provider workspaces = %v, want [%q]", appendProvider.workspaces, workspace)
		}
	})

	t.Run("nil assembler returns trimmed base prompt", func(t *testing.T) {
		t.Parallel()

		var assembler *ComposedAssembler
		workspace := testResolvedWorkspace(t.TempDir())
		got, err := assembler.Assemble(
			context.Background(),
			testPromptAgent("  Base prompt.\n"),
			&workspace,
		)
		if err != nil {
			t.Fatalf("Assemble() error = %v", err)
		}
		if got != "Base prompt." {
			t.Fatalf("Assemble() = %q, want %q", got, "Base prompt.")
		}
	})

	t.Run("empty and nil options are ignored", func(t *testing.T) {
		t.Parallel()

		assembler := NewComposedAssembler(
			nil,
			WithPrependPromptProviders(),
			WithAppendPromptProviders(),
		)

		got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
		if got != "Base prompt." {
			t.Fatalf("Assemble() = %q, want %q", got, "Base prompt.")
		}
	})
}

func TestComposedAssemblerRegressionMatchesMemoryAssembler(t *testing.T) {
	t.Parallel()

	env := newComposedAssemblerMemoryEnv(t)
	env.writeGlobalIndex(t, "- [Global](global.md) - global note")
	env.writeWorkspaceIndex(t, "- [Workspace](workspace.md) - workspace note")

	memoryAssembler := memory.NewAssembler(env.store)
	composedAssembler := NewComposedAssembler(
		WithPrependPromptProviders(memoryAssembler),
	)

	workspace := testResolvedWorkspace(env.workspace)
	got, err := composedAssembler.Assemble(context.Background(), env.agent, &workspace)
	if err != nil {
		t.Fatalf("ComposedAssembler.Assemble() error = %v", err)
	}

	want, err := memoryAssembler.Assemble(context.Background(), env.agent, &workspace)
	if err != nil {
		t.Fatalf("memory.Assemble() error = %v", err)
	}

	if got != want {
		t.Fatalf("memory-only regression mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestComposedAssemblerAssembleStartupUsesEligibleSectionOrdering(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
		MemoryPromptSectionEnabled: true,
		SkillsPromptSectionEnabled: true,
	})
	assembler := NewComposedAssembler(
		WithSectionSelector(NewSectionSelector(resolver)),
		WithPromptSectionDescriptors(
			PromptSectionDescriptor{
				Name:     string(HarnessPromptSectionNetwork),
				Position: PromptSectionPositionAppend,
				Order:    200,
				Provider: staticPromptProvider("network block"),
				Predicate: policyIncludesSection(
					HarnessPromptSectionNetwork,
				),
			},
			PromptSectionDescriptor{
				Name:     string(HarnessPromptSectionSkills),
				Position: PromptSectionPositionAppend,
				Order:    100,
				Provider: staticPromptProvider("skills block"),
				Predicate: policyIncludesSection(
					HarnessPromptSectionSkills,
				),
			},
			PromptSectionDescriptor{
				Name:     string(HarnessPromptSectionMemory),
				Position: PromptSectionPositionPrepend,
				Order:    100,
				Provider: staticPromptProvider("memory block"),
				Predicate: policyIncludesSection(
					HarnessPromptSectionMemory,
				),
			},
		),
	)

	got := assembleStartupPrompt(
		t,
		assembler,
		session.StartupPromptContext{
			SessionType: session.SessionTypeUser,
			Channel:     "builders",
		},
		testPromptAgent("Base prompt."),
		t.TempDir(),
	)

	want := "memory block\n\nBase prompt.\n\nskills block\n\nnetwork block"
	if got != want {
		t.Fatalf("AssembleStartup() = %q, want %q", got, want)
	}
}

func TestComposedAssemblerAppliesBudgetPolicies(t *testing.T) {
	t.Parallel()

	assembler := NewComposedAssembler(
		WithPromptSectionDescriptors(
			PromptSectionDescriptor{
				Name:           "trimmed",
				Position:       PromptSectionPositionPrepend,
				Order:          10,
				Budget:         5,
				BudgetBehavior: PromptSectionBudgetBehaviorTrim,
				Provider:       staticPromptProvider("123456789"),
			},
			PromptSectionDescriptor{
				Name:           "omitted",
				Position:       PromptSectionPositionAppend,
				Order:          20,
				Budget:         4,
				BudgetBehavior: PromptSectionBudgetBehaviorOmit,
				Provider:       staticPromptProvider("abcdef"),
			},
			PromptSectionDescriptor{
				Name:     "empty",
				Position: PromptSectionPositionAppend,
				Order:    30,
				Provider: staticPromptProvider("   \n\t"),
			},
		),
	)

	got := assemblePrompt(t, assembler, testPromptAgent("Base prompt."), t.TempDir())
	want := "12345\n\nBase prompt."
	if got != want {
		t.Fatalf("Assemble() = %q, want %q", got, want)
	}
}

func TestComposedAssemblerDeduplicatesEligibleSectionNames(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
		MemoryPromptSectionEnabled: true,
		SkillsPromptSectionEnabled: true,
	})
	assembler := NewComposedAssembler(
		WithSectionSelector(NewSectionSelector(resolver)),
		WithPromptSectionDescriptors(
			PromptSectionDescriptor{
				Name:     string(HarnessPromptSectionMemory),
				Position: PromptSectionPositionPrepend,
				Order:    100,
				Provider: staticPromptProvider("memory block"),
				Predicate: policyIncludesSection(
					HarnessPromptSectionMemory,
				),
			},
			PromptSectionDescriptor{
				Name:     string(HarnessPromptSectionNetwork),
				Position: PromptSectionPositionAppend,
				Order:    200,
				Provider: staticPromptProvider("network block"),
				Predicate: policyIncludesSection(
					HarnessPromptSectionNetwork,
				),
			},
			PromptSectionDescriptor{
				Name:     string(HarnessPromptSectionNetwork),
				Position: PromptSectionPositionAppend,
				Order:    210,
				Provider: staticPromptProvider("network block duplicate"),
				Predicate: policyIncludesSection(
					HarnessPromptSectionNetwork,
				),
			},
		),
	)

	got := assembleStartupPrompt(
		t,
		assembler,
		session.StartupPromptContext{
			SessionType: session.SessionTypeUser,
			Channel:     "builders",
		},
		testPromptAgent("Base prompt."),
		t.TempDir(),
	)

	if strings.Count(got, "network block") != 1 {
		t.Fatalf("network block occurrences = %d, want 1", strings.Count(got, "network block"))
	}
	if strings.Contains(got, "network block duplicate") {
		t.Fatalf("assembled prompt unexpectedly contains duplicate network block: %q", got)
	}
}

func TestComposedAssemblerAssembleStartupLoadsBundledNetworkSectionDescriptor(t *testing.T) {
	t.Parallel()

	resolver := NewHarnessContextResolver(HarnessRuntimeSignals{})
	assembler := NewComposedAssembler(
		WithSectionSelector(NewSectionSelector(resolver)),
		WithPromptSectionDescriptors(defaultStartupPromptSectionDescriptors(nil, nil)...),
	)

	got := assembleStartupPrompt(
		t,
		assembler,
		session.StartupPromptContext{
			SessionType: session.SessionTypeUser,
			Channel:     "builders",
		},
		testPromptAgent("Base prompt."),
		t.TempDir(),
	)

	networkSkill, err := bundled.LoadContent(bundledNetworkSkillName)
	if err != nil {
		t.Fatalf("LoadContent(%q) error = %v", bundledNetworkSkillName, err)
	}
	if !strings.Contains(got, networkSkill) {
		t.Fatalf("AssembleStartup() = %q, want bundled network skill content", got)
	}
}

type recordingPromptProvider struct {
	section    string
	err        error
	workspaces []string
}

func (p *recordingPromptProvider) PromptSection(
	_ context.Context,
	workspace *workspacepkg.ResolvedWorkspace,
) (string, error) {
	if workspace != nil {
		p.workspaces = append(p.workspaces, workspace.RootDir)
	}
	if p.err != nil {
		return "", p.err
	}
	return p.section, nil
}

type errorPromptProvider struct {
	err error
}

func (p errorPromptProvider) PromptSection(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) {
	return "", p.err
}

type staticPromptProvider string

func (p staticPromptProvider) PromptSection(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) {
	return string(p), nil
}

type composedAssemblerMemoryEnv struct {
	store     *memory.Store
	globalDir string
	workspace string
	agent     aghconfig.AgentDef
}

func newComposedAssemblerMemoryEnv(t *testing.T) composedAssemblerMemoryEnv {
	t.Helper()

	baseDir := t.TempDir()
	workspace := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", workspace, err)
	}

	store := memory.NewStore(filepath.Join(baseDir, "home", "memory"))
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("Store.EnsureDirs() error = %v", err)
	}

	return composedAssemblerMemoryEnv{
		store:     store,
		globalDir: filepath.Join(baseDir, "home", "memory"),
		workspace: workspace,
		agent:     testPromptAgent("  You are a coding assistant.\n"),
	}
}

func (e composedAssemblerMemoryEnv) writeGlobalIndex(t *testing.T, content string) {
	t.Helper()
	writeComposedAssemblerFile(t, filepath.Join(e.globalDir, "MEMORY.md"), content)
}

func (e composedAssemblerMemoryEnv) writeWorkspaceIndex(t *testing.T, content string) {
	t.Helper()
	writeComposedAssemblerFile(t, filepath.Join(e.workspace, aghconfig.DirName, "memory", "MEMORY.md"), content)
}

func writeComposedAssemblerFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assemblePrompt(t *testing.T, assembler *ComposedAssembler, agent aghconfig.AgentDef, workspace string) string {
	t.Helper()

	resolvedWorkspace := testResolvedWorkspace(workspace)
	got, err := assembler.Assemble(context.Background(), agent, &resolvedWorkspace)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	return got
}

func assembleStartupPrompt(
	t *testing.T,
	assembler *ComposedAssembler,
	startup session.StartupPromptContext,
	agent aghconfig.AgentDef,
	workspace string,
) string {
	t.Helper()

	resolvedWorkspace := testResolvedWorkspace(workspace)
	got, err := assembler.AssembleStartup(context.Background(), startup, agent, &resolvedWorkspace)
	if err != nil {
		t.Fatalf("AssembleStartup() error = %v", err)
	}
	return got
}

func testPromptAgent(prompt string) aghconfig.AgentDef {
	return aghconfig.AgentDef{
		Name:     "coder",
		Provider: "claude",
		Prompt:   prompt,
	}
}

func testResolvedWorkspace(root string) workspacepkg.ResolvedWorkspace {
	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{RootDir: root},
	}
}
