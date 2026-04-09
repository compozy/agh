package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/frontmatter"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

// AgentDef is the parsed representation of an AGENT.md file.
type AgentDef struct {
	Name        string              `yaml:"name" toml:"name"`
	Provider    string              `yaml:"provider" toml:"provider"`
	Command     string              `yaml:"command,omitempty" toml:"command,omitempty"`
	Model       string              `yaml:"model,omitempty" toml:"model,omitempty"`
	Tools       []string            `yaml:"tools,omitempty" toml:"tools,omitempty"`
	Permissions string              `yaml:"permissions,omitempty" toml:"permissions,omitempty"`
	MCPServers  []MCPServer         `yaml:"mcp_servers,omitempty" toml:"mcp_servers,omitempty"`
	Hooks       []hookspkg.HookDecl `yaml:"hooks,omitempty" toml:"hooks,omitempty"`
	Prompt      string              `yaml:"-"`
}

type parsedAgentDef struct {
	Name        string                  `yaml:"name" toml:"name"`
	Provider    string                  `yaml:"provider" toml:"provider"`
	Command     string                  `yaml:"command,omitempty" toml:"command,omitempty"`
	Model       string                  `yaml:"model,omitempty" toml:"model,omitempty"`
	Tools       []string                `yaml:"tools,omitempty" toml:"tools,omitempty"`
	Permissions string                  `yaml:"permissions,omitempty" toml:"permissions,omitempty"`
	MCPServers  []MCPServer             `yaml:"mcp_servers,omitempty" toml:"mcp_servers,omitempty"`
	Hooks       []parsedHookDeclaration `yaml:"hooks,omitempty" toml:"hooks,omitempty"`
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

var (
	// ErrMissingAgentFrontmatter reports a missing YAML frontmatter block in AGENT.md content.
	ErrMissingAgentFrontmatter = errors.New("config: missing YAML frontmatter")
	// ErrUnterminatedAgentFrontmatter reports an unterminated YAML frontmatter block in AGENT.md content.
	ErrUnterminatedAgentFrontmatter = errors.New("config: unterminated YAML frontmatter")
)

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
	var parsed parsedAgentDef

	body, err := frontmatter.Decode(content, func(data []byte) error {
		return decodeAgentFrontmatter(data, &parsed)
	})
	if err != nil {
		return AgentDef{}, wrapFrontmatterError(err)
	}

	agent := AgentDef{
		Name:        strings.TrimSpace(parsed.Name),
		Provider:    strings.TrimSpace(parsed.Provider),
		Command:     strings.TrimSpace(parsed.Command),
		Model:       strings.TrimSpace(parsed.Model),
		Tools:       cloneStrings(parsed.Tools),
		Permissions: strings.TrimSpace(parsed.Permissions),
		MCPServers:  cloneMCPServers(parsed.MCPServers),
		Prompt:      strings.TrimSpace(body),
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
	if len(parsed.Hooks) > 0 {
		agent.Hooks = make([]hookspkg.HookDecl, 0, len(parsed.Hooks))
		for idx, raw := range parsed.Hooks {
			decl, err := raw.toHookDecl(hookspkg.HookSourceAgentDefinition, agent.Name)
			if err != nil {
				return AgentDef{}, fmt.Errorf("agent.hooks[%d]: %w", idx, err)
			}
			agent.Hooks = append(agent.Hooks, decl)
		}
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
	for i, hook := range a.Hooks {
		if err := hookspkg.ValidateHookDecl(hook); err != nil {
			return fmt.Errorf("agent.hooks[%d]: %w", i, err)
		}
	}

	return nil
}

func wrapFrontmatterError(err error) error {
	switch {
	case errors.Is(err, frontmatter.ErrMissing):
		return mappedFrontmatterError{
			message: ErrMissingAgentFrontmatter.Error(),
			causes:  []error{ErrMissingAgentFrontmatter, err},
		}
	case errors.Is(err, frontmatter.ErrUnterminated):
		return mappedFrontmatterError{
			message: ErrUnterminatedAgentFrontmatter.Error(),
			causes:  []error{ErrUnterminatedAgentFrontmatter, err},
		}
	default:
		return err
	}
}

func decodeAgentFrontmatter(data []byte, parsed *parsedAgentDef) error {
	if err := yaml.UnmarshalWithOptions(data, parsed, yaml.Strict()); err == nil {
		return nil
	} else {
		var parsedTOML parsedAgentDef
		meta, tomlErr := toml.Decode(string(data), &parsedTOML)
		if tomlErr != nil {
			return fmt.Errorf("decode agent frontmatter: yaml: %w; toml: %v", err, tomlErr)
		}
		if undecoded := meta.Undecoded(); len(undecoded) > 0 {
			return fmt.Errorf("decode agent frontmatter: unknown field %q", undecoded[0].String())
		}
		*parsed = parsedTOML
		return nil
	}
}

type mappedFrontmatterError struct {
	message string
	causes  []error
}

func (e mappedFrontmatterError) Error() string {
	return e.message
}

func (e mappedFrontmatterError) Unwrap() []error {
	return e.causes
}
