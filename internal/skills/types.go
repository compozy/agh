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
	Meta     SkillMeta
	Content  string
	Source   SkillSource
	Dir      string
	FilePath string
	Enabled  bool
}

// SkillSource identifies where a skill was loaded from.
type SkillSource int

const (
	// SourceBundled is the lowest-precedence source backed by go:embed files.
	SourceBundled SkillSource = iota
	// SourceUser identifies skills loaded from the user-level skill directories.
	SourceUser
	// SourceAgents identifies skills loaded from `<workspace>/.agents/skills/`.
	SourceAgents
	// SourceWorkspace is the highest-precedence source from `<workspace>/.agh/skills/`.
	SourceWorkspace
)

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

// fileSnapshot tracks file metadata used to detect staleness.
type fileSnapshot struct {
	path    string
	modTime time.Time
	size    int64
}
