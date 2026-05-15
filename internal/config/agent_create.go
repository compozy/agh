package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

var (
	// ErrInvalidAgentDefinition marks validation failures while authoring an AGENT.md file.
	ErrInvalidAgentDefinition = errors.New("config: invalid agent definition")
	// ErrAgentDefinitionExists marks a create request that would overwrite an existing AGENT.md file.
	ErrAgentDefinitionExists = errors.New("config: agent definition already exists")
)

// AgentDefinitionDraft captures the simple AGENT.md fields supported by authoring surfaces.
type AgentDefinitionDraft struct {
	Name         string
	Provider     string
	Command      string
	Model        string
	Tools        []string
	Toolsets     []string
	DenyTools    []string
	Permissions  string
	Skills       AgentSkillsConfig
	CategoryPath []string
	Prompt       string
}

// CreateAgentDefFile renders, validates, and persists one AGENT.md definition.
func CreateAgentDefFile(path string, draft AgentDefinitionDraft, overwrite bool) (AgentDef, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return AgentDef{}, fmt.Errorf("config: agent definition path is required")
	}

	contents, agent, err := RenderAgentDefinition(draft)
	if err != nil {
		return AgentDef{}, err
	}
	if err := ensureAgentDefinitionWritable(normalizedPath, overwrite); err != nil {
		return AgentDef{}, err
	}
	if err := writePersistedFile(normalizedPath, contents); err != nil {
		return AgentDef{}, fmt.Errorf("config: write agent definition %q: %w", normalizedPath, err)
	}

	agent.SourcePath = filepath.Clean(normalizedPath)
	return agent, nil
}

// RenderAgentDefinition renders a draft to AGENT.md bytes and validates by parsing the result.
func RenderAgentDefinition(draft AgentDefinitionDraft) ([]byte, AgentDef, error) {
	agentName := NormalizeAgentName(draft.Name)
	if err := ValidateAgentName(agentName); err != nil {
		return nil, AgentDef{}, errors.Join(ErrInvalidAgentDefinition, err)
	}
	agent := AgentDef{
		Name:         agentName,
		Provider:     strings.TrimSpace(draft.Provider),
		Command:      strings.TrimSpace(draft.Command),
		Model:        strings.TrimSpace(draft.Model),
		Tools:        trimAgentDefinitionAtoms(draft.Tools),
		Toolsets:     trimAgentDefinitionAtoms(draft.Toolsets),
		DenyTools:    trimAgentDefinitionAtoms(draft.DenyTools),
		Permissions:  strings.TrimSpace(draft.Permissions),
		Skills:       AgentSkillsConfig{Disabled: trimAgentDefinitionAtoms(draft.Skills.Disabled)},
		CategoryPath: trimAgentDefinitionAtoms(draft.CategoryPath),
		Prompt:       strings.TrimSpace(draft.Prompt),
	}
	agent.Skills = normalizeAgentSkillsConfig(agent.Skills)
	agent.CategoryPath = normalizeAgentCategoryPath(agent.CategoryPath)
	if err := agent.Validate(); err != nil {
		return nil, AgentDef{}, errors.Join(ErrInvalidAgentDefinition, err)
	}

	parsed := parsedAgentDef{}
	applyAgentDefToParsed(&parsed, agent)
	frontmatter, err := yaml.Marshal(parsed)
	if err != nil {
		return nil, AgentDef{}, fmt.Errorf("config: render agent frontmatter: %w", err)
	}
	contents := renderAgentMarkdown(frontmatter, agent.Prompt)
	validated, err := ParseAgentDef(contents)
	if err != nil {
		return nil, AgentDef{}, errors.Join(
			ErrInvalidAgentDefinition,
			fmt.Errorf("config: validate generated agent definition: %w", err),
		)
	}
	if validated.Name != agentName {
		return nil, AgentDef{}, errors.Join(
			ErrInvalidAgentDefinition,
			fmt.Errorf("config: generated agent name %q does not match %q", validated.Name, agentName),
		)
	}
	return contents, validated, nil
}

func ensureAgentDefinitionWritable(path string, overwrite bool) error {
	if overwrite {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return errors.Join(
			ErrAgentDefinitionExists,
			fmt.Errorf("config: agent definition already exists at %s", path),
		)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("config: inspect agent definition %q: %w", path, err)
	}
	return nil
}

func trimAgentDefinitionAtoms(values []string) []string {
	atoms := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		atoms = append(atoms, trimmed)
	}
	return atoms
}
