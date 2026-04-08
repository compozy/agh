// Package skills provides the core types and loading primitives for AgentSkills
// `SKILL.md` files.
package skills

import (
	"io/fs"
	"time"
)

// SkillMeta maps YAML frontmatter fields per the AgentSkills spec.
type SkillMeta struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Version     string         `yaml:"version,omitempty"`
	Metadata    map[string]any `yaml:"metadata,omitempty"`
}

// Skill is the complete in-memory representation of a parsed skill file.
type Skill struct {
	Meta          SkillMeta
	Content       string
	Source        SkillSource
	Dir           string
	FilePath      string
	Enabled       bool
	MCPServers    []MCPServerDecl
	Hooks         []HookDecl
	Provenance    *Provenance
	InstalledFrom string
}

// SkillSource identifies where a skill was loaded from.
type SkillSource int

const (
	// SourceBundled is the lowest-precedence source backed by go:embed files.
	SourceBundled SkillSource = iota
	// SourceMarketplace identifies skills installed from a marketplace registry.
	SourceMarketplace
	// SourceUser identifies skills loaded from the user-level skill directories.
	SourceUser
	// SourceAdditional identifies skills loaded from additional workspace roots.
	SourceAdditional
	// SourceWorkspace is the highest-precedence source from `<workspace>/.agh/skills/`.
	SourceWorkspace
)

// MCPServerDecl declares an MCP server dependency in skill frontmatter.
type MCPServerDecl struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// HookDecl declares a lifecycle hook in skill frontmatter.
type HookDecl struct {
	Event   HookEvent         `yaml:"event"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Timeout time.Duration     `yaml:"timeout,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// HookEvent identifies when a hook fires.
type HookEvent string

const (
	HookSessionCreated HookEvent = "on_session_created"
	HookSessionStopped HookEvent = "on_session_stopped"
)

// Provenance stores marketplace install metadata for a skill.
type Provenance struct {
	Hash        string    `json:"hash"`
	Registry    string    `json:"registry"`
	Slug        string    `json:"slug"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

// WarningSeverity describes the impact of a loader or verifier warning.
type WarningSeverity int

const (
	SeverityInfo WarningSeverity = iota
	SeverityWarning
	SeverityCritical
)

// Warning captures a verification or loading concern associated with a skill.
type Warning struct {
	Severity WarningSeverity
	Message  string
	Pattern  string
}

// RegistryConfig controls how the registry discovers global skills.
type RegistryConfig struct {
	BundledFS      fs.FS
	UserSkillsDir  string
	UserAgentsDir  string
	DisabledSkills []string
}
