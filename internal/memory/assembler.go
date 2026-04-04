package memory

import (
	"context"
	"fmt"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	memoryPromptIntro = `# Persistent Memory

Only prompt-safe MEMORY.md indexes are injected here. Read full memory files on demand when relevant.`
	memoryTaxonomySection = `## Memory Taxonomy

- ` + "`user`" + `: stable preferences or working style that apply across projects.
- ` + "`feedback`" + `: recurring quality signals, review feedback, or mistakes to avoid next time.
- ` + "`project`" + `: current codebase decisions, ongoing work, and project-specific constraints.
- ` + "`reference`" + `: external facts, docs, or system references worth re-reading on demand.`
	memoryCommandsSection = `## Memory Commands

- ` + "`agh memory list`" + ` shows discoverable memory files in the current scope.
- ` + "`agh memory read <filename>`" + ` reads the full content of one memory file.
- ` + "`agh memory write <filename> --type <type> --description <desc> --content <content>`" + ` writes or updates durable memory.`
	memoryStalenessSection = `## Staleness Policy

- Memories older than 1 day should be verified against the current repository or system state before asserting them as fact.`
)

// Assembler loads prompt-safe memory indexes and prepends them to the agent prompt.
type Assembler struct {
	store *Store
}

// NewAssembler constructs a prompt assembler for the provided store.
func NewAssembler(store *Store) *Assembler {
	return &Assembler{store: store}
}

// Assemble renders the dual-scope memory context ahead of the agent system prompt.
func (a *Assembler) Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace string) (string, error) {
	basePrompt := strings.TrimSpace(agent.Prompt)
	if a == nil || a.store == nil {
		return basePrompt, nil
	}
	if err := contextErr(ctx); err != nil {
		return "", err
	}

	globalIndex, globalTruncated, err := a.store.LoadIndex(ScopeGlobal)
	if err != nil {
		return "", fmt.Errorf("memory: load global index: %w", err)
	}
	if err := contextErr(ctx); err != nil {
		return "", err
	}

	workspaceIndex, workspaceTruncated, err := a.store.ForWorkspace(workspace).LoadIndex(ScopeWorkspace)
	if err != nil {
		return "", fmt.Errorf("memory: load workspace index: %w", err)
	}

	globalIndex = strings.TrimSpace(globalIndex)
	workspaceIndex = strings.TrimSpace(workspaceIndex)
	if globalIndex == "" && workspaceIndex == "" {
		return basePrompt, nil
	}

	contextBlock := renderMemoryContext(memoryContext{
		globalIndex:        globalIndex,
		globalTruncated:    globalTruncated,
		workspaceIndex:     workspaceIndex,
		workspaceTruncated: workspaceTruncated,
	})
	if basePrompt == "" {
		return contextBlock, nil
	}

	return contextBlock + "\n\n" + basePrompt, nil
}

type memoryContext struct {
	globalIndex        string
	globalTruncated    bool
	workspaceIndex     string
	workspaceTruncated bool
}

func renderMemoryContext(ctx memoryContext) string {
	sections := []string{
		memoryPromptIntro,
		renderMemoryIndexSection("Global MEMORY.md Index", ctx.globalIndex, ctx.globalTruncated),
		renderMemoryIndexSection("Workspace MEMORY.md Index", ctx.workspaceIndex, ctx.workspaceTruncated),
		memoryTaxonomySection,
		memoryCommandsSection,
		memoryStalenessSection,
	}

	parts := make([]string, 0, len(sections))
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}

	return strings.Join(parts, "\n\n")
}

func renderMemoryIndexSection(title string, index string, truncated bool) string {
	content := strings.TrimSpace(index)
	if content == "" {
		return ""
	}

	lines := []string{"## " + strings.TrimSpace(title)}
	if truncated {
		lines = append(lines, "_Index truncated to fit prompt limits._")
	}
	lines = append(lines, content)
	return strings.Join(lines, "\n\n")
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
