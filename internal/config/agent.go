package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

var (
	errFrontmatterMissing      = errors.New("config: missing YAML frontmatter")
	errFrontmatterUnterminated = errors.New("config: unterminated YAML frontmatter")
)

// AgentDef is the parsed representation of an AGENT.md file.
type AgentDef struct {
	Name        string      `yaml:"name"`
	Provider    string      `yaml:"provider"`
	Command     string      `yaml:"command,omitempty"`
	Model       string      `yaml:"model,omitempty"`
	Tools       []string    `yaml:"tools,omitempty"`
	Permissions string      `yaml:"permissions,omitempty"`
	MCPServers  []MCPServer `yaml:"mcp_servers,omitempty"`
	Prompt      string      `yaml:"-"`
}

// WorkspaceDiscoverySource identifies where a discovery root came from.
type WorkspaceDiscoverySource string

const (
	// WorkspaceDiscoverySourceWorkspace marks the primary workspace root.
	WorkspaceDiscoverySourceWorkspace WorkspaceDiscoverySource = "workspace"
	// WorkspaceDiscoverySourceAdditional marks an additional workspace root.
	WorkspaceDiscoverySourceAdditional WorkspaceDiscoverySource = "additional"
	// WorkspaceDiscoverySourceGlobal marks the global AGH home root.
	WorkspaceDiscoverySourceGlobal WorkspaceDiscoverySource = "global"
)

// WorkspaceDiscoveryRoot describes a filesystem root participating in multi-root resource discovery.
type WorkspaceDiscoveryRoot struct {
	Dir    string
	Source WorkspaceDiscoverySource
}

// LoadAgentDef loads an AGENT.md file from the configured AGH home directory.
func LoadAgentDef(name string, homePaths HomePaths) (AgentDef, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return AgentDef{}, errors.New("agent name is required")
	}

	path := filepath.Join(homePaths.AgentsDir, target, agentDefName)
	agent, err := LoadAgentDefFile(path)
	if err != nil {
		return AgentDef{}, err
	}
	if agent.Name != target {
		return AgentDef{}, fmt.Errorf("agent file %q defines name %q, expected %q", path, agent.Name, target)
	}

	return agent, nil
}

// LoadAgentDefFile loads and parses an AGENT.md file from an explicit path.
func LoadAgentDefFile(path string) (AgentDef, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return AgentDef{}, fmt.Errorf("read agent file %q: %w", path, err)
	}

	agent, err := ParseAgentDef(contents)
	if err != nil {
		return AgentDef{}, fmt.Errorf("parse agent file %q: %w", path, err)
	}

	return agent, nil
}

// WorkspaceDiscoveryRoots returns ordered discovery roots for workspace-scoped resources.
// Precedence is left to right: workspace root, additional roots, then the global AGH home.
func WorkspaceDiscoveryRoots(rootDir string, additionalDirs []string, homePaths HomePaths) []WorkspaceDiscoveryRoot {
	roots := make([]WorkspaceDiscoveryRoot, 0, len(additionalDirs)+2)

	if trimmed := strings.TrimSpace(rootDir); trimmed != "" {
		roots = append(roots, WorkspaceDiscoveryRoot{
			Dir:    trimmed,
			Source: WorkspaceDiscoverySourceWorkspace,
		})
	}

	for _, dir := range additionalDirs {
		if trimmed := strings.TrimSpace(dir); trimmed != "" {
			roots = append(roots, WorkspaceDiscoveryRoot{
				Dir:    trimmed,
				Source: WorkspaceDiscoverySourceAdditional,
			})
		}
	}

	if trimmed := strings.TrimSpace(homePaths.HomeDir); trimmed != "" {
		roots = append(roots, WorkspaceDiscoveryRoot{
			Dir:    trimmed,
			Source: WorkspaceDiscoverySourceGlobal,
		})
	}

	return roots
}

// AgentsDir returns the agent-definition directory for this discovery root.
func (r WorkspaceDiscoveryRoot) AgentsDir() string {
	if r.Source == WorkspaceDiscoverySourceGlobal {
		return filepath.Join(r.Dir, AgentsDirName)
	}

	return filepath.Join(r.Dir, DirName, AgentsDirName)
}

// SkillsDir returns the skill-definition directory for this discovery root.
func (r WorkspaceDiscoveryRoot) SkillsDir() string {
	if r.Source == WorkspaceDiscoverySourceGlobal {
		return filepath.Join(r.Dir, SkillsDirName)
	}

	return filepath.Join(r.Dir, DirName, SkillsDirName)
}

// LoadWorkspaceAgentDefs loads workspace-visible agents using root, additional, then global precedence.
func LoadWorkspaceAgentDefs(rootDir string, additionalDirs []string, homePaths HomePaths) ([]AgentDef, error) {
	roots := WorkspaceDiscoveryRoots(rootDir, additionalDirs, homePaths)
	if len(roots) == 0 {
		return nil, nil
	}

	agents := make([]AgentDef, 0)
	seen := make(map[string]struct{})

	for _, root := range roots {
		entries, err := os.ReadDir(root.AgentsDir())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read agents directory %q: %w", root.AgentsDir(), err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			agentPath := filepath.Join(root.AgentsDir(), entry.Name(), agentDefName)
			agent, err := LoadAgentDefFile(agentPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}

			if _, ok := seen[agent.Name]; ok {
				continue
			}

			seen[agent.Name] = struct{}{}
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// ParseAgentDef parses a Markdown file with YAML frontmatter into an AgentDef.
func ParseAgentDef(content []byte) (AgentDef, error) {
	var agent AgentDef

	body, err := parseFrontmatter(content, &agent)
	if err != nil {
		return AgentDef{}, err
	}

	agent.Name = strings.TrimSpace(agent.Name)
	agent.Provider = strings.TrimSpace(agent.Provider)
	agent.Command = strings.TrimSpace(agent.Command)
	agent.Model = strings.TrimSpace(agent.Model)
	agent.Permissions = strings.TrimSpace(agent.Permissions)
	agent.Prompt = strings.TrimSpace(body)
	if len(agent.Tools) == 0 {
		agent.Tools = []string{"*"}
	}

	if err := agent.Validate(); err != nil {
		return AgentDef{}, err
	}

	return agent, nil
}

// Validate ensures the parsed agent definition is usable.
func (a AgentDef) Validate() error {
	switch {
	case strings.TrimSpace(a.Name) == "":
		return errors.New("agent name is required")
	case strings.TrimSpace(a.Prompt) == "":
		return errors.New("agent prompt is required")
	}

	if strings.TrimSpace(a.Permissions) != "" {
		if err := PermissionMode(a.Permissions).Validate("agent.permissions"); err != nil {
			return err
		}
	}

	for i, server := range a.MCPServers {
		if err := server.Validate(fmt.Sprintf("agent.mcp_servers[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func parseFrontmatter(content []byte, dest any) (string, error) {
	normalized := normalizeLineEndings(content)
	if !bytes.HasPrefix(normalized, []byte("---")) {
		return "", errFrontmatterMissing
	}

	openLineEnd, ok := nextLineBoundary(normalized, 0)
	if !ok || string(normalized[:openLineEnd]) != "---" {
		return "", errFrontmatterMissing
	}

	offset := openLineEnd
	if offset < len(normalized) && normalized[offset] == '\n' {
		offset++
	}

	closeStart, closeEnd, ok := findClosingDelimiter(normalized, offset)
	if !ok {
		return "", errFrontmatterUnterminated
	}

	if err := yaml.UnmarshalWithOptions(normalized[offset:closeStart], dest, yaml.Strict()); err != nil {
		return "", fmt.Errorf("decode YAML frontmatter: %w", err)
	}

	bodyStart := closeEnd
	if bodyStart < len(normalized) && normalized[bodyStart] == '\n' {
		bodyStart++
	}

	return string(normalized[bodyStart:]), nil
}

func normalizeLineEndings(content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), "\r\n", "\n"))
}

func nextLineBoundary(content []byte, start int) (int, bool) {
	if start >= len(content) {
		return len(content), true
	}

	if idx := bytes.IndexByte(content[start:], '\n'); idx >= 0 {
		return start + idx, true
	}

	return len(content), true
}

func findClosingDelimiter(content []byte, start int) (int, int, bool) {
	lineStart := start
	for lineStart <= len(content) {
		lineEnd, ok := nextLineBoundary(content, lineStart)
		if !ok {
			return 0, 0, false
		}
		if string(content[lineStart:lineEnd]) == "---" {
			return lineStart, lineEnd, true
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
	}

	return 0, 0, false
}
