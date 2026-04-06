package skills

import (
	"context"
	"fmt"
	"slices"
	"strings"

	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	catalogDescriptionLimit  = 200
	catalogEllipsis          = "..."
	catalogUsageInstructions = "Use `agh skill view <name>` to load full instructions for any skill.\n" +
		"Use `agh skill view <name> --file <path>` to read a specific skill resource file."
)

var (
	catalogTextReplacer = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	catalogAttrReplacer = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
)

// CatalogProvider builds the workspace-scoped skill catalog section expected by
// the composed prompt assembly pipeline.
type CatalogProvider struct {
	registry *Registry
}

// NewCatalogProvider constructs a CatalogProvider backed by the provided registry.
func NewCatalogProvider(registry *Registry) *CatalogProvider {
	return &CatalogProvider{registry: registry}
}

// PromptSection loads the workspace-scoped skills and returns their XML-like
// catalog representation.
func (cp *CatalogProvider) PromptSection(ctx context.Context, workspace workspacepkg.ResolvedWorkspace) (string, error) {
	if cp == nil || cp.registry == nil {
		return "", nil
	}

	skills, err := cp.registry.ForWorkspace(ctx, workspace)
	if err != nil {
		return "", fmt.Errorf("skills: build catalog for workspace %q: %w", catalogWorkspaceLabel(workspace), err)
	}

	return BuildCatalog(skills), nil
}

// BuildCatalog renders the XML-like available-skills block injected into agent
// system prompts.
func BuildCatalog(skills []*Skill) string {
	type catalogEntry struct {
		name        string
		description string
	}

	entries := make([]catalogEntry, 0, len(skills))
	for _, skill := range skills {
		if skill == nil {
			continue
		}

		name := strings.TrimSpace(skill.Meta.Name)
		if name == "" {
			continue
		}

		entries = append(entries, catalogEntry{
			name:        name,
			description: truncateCatalogDescription(skill.Meta.Description),
		})
	}

	if len(entries) == 0 {
		return ""
	}

	slices.SortFunc(entries, func(left, right catalogEntry) int {
		return strings.Compare(left.name, right.name)
	})

	var builder strings.Builder
	builder.Grow(len(entries) * 64)
	builder.WriteString("<available-skills>\n")
	for _, entry := range entries {
		builder.WriteString(`  <skill name="`)
		builder.WriteString(escapeCatalogAttr(entry.name))
		builder.WriteString(`">`)
		builder.WriteString(escapeCatalogText(entry.description))
		builder.WriteString("</skill>\n")
	}
	builder.WriteString("</available-skills>\n\n")
	builder.WriteString(catalogUsageInstructions)

	return builder.String()
}

func truncateCatalogDescription(description string) string {
	runes := []rune(description)
	if len(runes) <= catalogDescriptionLimit {
		return description
	}

	return string(runes[:catalogDescriptionLimit-len(catalogEllipsis)]) + catalogEllipsis
}

func escapeCatalogText(value string) string {
	return catalogTextReplacer.Replace(value)
}

func escapeCatalogAttr(value string) string {
	return catalogAttrReplacer.Replace(value)
}

func catalogWorkspaceLabel(workspace workspacepkg.ResolvedWorkspace) string {
	if name := strings.TrimSpace(workspace.Name); name != "" {
		return name
	}
	if root := strings.TrimSpace(workspace.RootDir); root != "" {
		return root
	}
	if id := strings.TrimSpace(workspace.ID); id != "" {
		return id
	}
	return "<global>"
}
