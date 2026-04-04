// Package memory manages persistent dual-scope memory files and MEMORY.md indexes.
package memory

import (
	"fmt"
	"strings"
	"time"
)

// MemoryType identifies the closed persistent-memory taxonomy.
type MemoryType string

const (
	// MemoryTypeUser stores user-level preferences and recurring facts.
	MemoryTypeUser MemoryType = "user"
	// MemoryTypeFeedback stores recurring quality and review feedback.
	MemoryTypeFeedback MemoryType = "feedback"
	// MemoryTypeProject stores workspace-specific project knowledge.
	MemoryTypeProject MemoryType = "project"
	// MemoryTypeReference stores workspace-specific external references.
	MemoryTypeReference MemoryType = "reference"
)

// Scope identifies which memory directory a file belongs to.
type Scope string

const (
	// ScopeGlobal targets the global memory directory.
	ScopeGlobal Scope = "global"
	// ScopeWorkspace targets the workspace memory directory.
	ScopeWorkspace Scope = "workspace"
)

// MemoryHeader contains validated metadata parsed from a memory file frontmatter.
type MemoryHeader struct {
	Filename    string     `json:"filename" yaml:"-"`
	FilePath    string     `json:"-" yaml:"-"`
	ModTime     time.Time  `json:"mod_time" yaml:"-"`
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Type        MemoryType `json:"type" yaml:"type"`
	AgentName   string     `json:"agent_name,omitempty" yaml:"agent_name,omitempty"`
}

// Normalize returns the normalized representation of the memory type.
func (t MemoryType) Normalize() MemoryType {
	return MemoryType(strings.ToLower(strings.TrimSpace(string(t))))
}

// Validate reports whether the memory type belongs to the closed taxonomy.
func (t MemoryType) Validate() error {
	switch t.Normalize() {
	case MemoryTypeUser, MemoryTypeFeedback, MemoryTypeProject, MemoryTypeReference:
		return nil
	case "":
		return fmt.Errorf("memory type is required")
	default:
		return fmt.Errorf("unsupported memory type %q", t)
	}
}

// DefaultScopeForType resolves the default persistence scope for a memory type.
func DefaultScopeForType(t MemoryType) (Scope, error) {
	switch t.Normalize() {
	case MemoryTypeUser, MemoryTypeFeedback:
		return ScopeGlobal, nil
	case MemoryTypeProject, MemoryTypeReference:
		return ScopeWorkspace, nil
	case "":
		return "", fmt.Errorf("memory type is required")
	default:
		return "", fmt.Errorf("unsupported memory type %q", t)
	}
}

// Normalize returns the normalized representation of the scope.
func (s Scope) Normalize() Scope {
	return Scope(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the scope is supported.
func (s Scope) Validate() error {
	switch s.Normalize() {
	case ScopeGlobal, ScopeWorkspace:
		return nil
	case "":
		return fmt.Errorf("scope is required")
	default:
		return fmt.Errorf("unsupported scope %q", s)
	}
}

// Normalize trims and normalizes the parsed memory header metadata in place.
func (h *MemoryHeader) Normalize() {
	h.Name = strings.TrimSpace(h.Name)
	h.Description = strings.TrimSpace(h.Description)
	h.Type = h.Type.Normalize()
	h.AgentName = strings.TrimSpace(h.AgentName)
}

// Validate reports whether the parsed memory header is complete and valid.
func (h *MemoryHeader) Validate() error {
	h.Normalize()

	if h.Name == "" {
		return fmt.Errorf("memory name is required")
	}

	if err := h.Type.Validate(); err != nil {
		return err
	}

	return nil
}
