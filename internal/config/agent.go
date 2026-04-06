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
