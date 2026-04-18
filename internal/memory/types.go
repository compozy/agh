// Package memory manages persistent dual-scope memory files and MEMORY.md indexes.
package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Type identifies the closed persistent-memory taxonomy.
type Type string

const (
	// MemoryTypeUser stores user-level preferences and recurring facts.
	MemoryTypeUser Type = "user"
	// MemoryTypeFeedback stores recurring quality and review feedback.
	MemoryTypeFeedback Type = "feedback"
	// MemoryTypeProject stores workspace-specific project knowledge.
	MemoryTypeProject Type = "project"
	// MemoryTypeReference stores workspace-specific external references.
	MemoryTypeReference Type = "reference"
)

// Scope identifies which memory directory a file belongs to.
type Scope string

const (
	// ScopeGlobal targets the global memory directory.
	ScopeGlobal Scope = "global"
	// ScopeWorkspace targets the workspace memory directory.
	ScopeWorkspace Scope = "workspace"
)

// Header contains validated metadata parsed from a memory file frontmatter.
type Header struct {
	Filename    string    `json:"filename"              yaml:"-"`
	FilePath    string    `json:"-"                     yaml:"-"`
	ModTime     time.Time `json:"mod_time"              yaml:"-"`
	Name        string    `json:"name"                  yaml:"name"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Type        Type      `json:"type"                  yaml:"type"`
	AgentName   string    `json:"agent_name,omitempty"  yaml:"agent_name,omitempty"`
}

// SearchOptions controls catalog-backed or fallback memory search behavior.
type SearchOptions struct {
	Scope     Scope
	Workspace string
	Limit     int
}

// SearchResult is one ranked memory search hit.
type SearchResult struct {
	Filename    string    `json:"filename"`
	Scope       Scope     `json:"scope"`
	Workspace   string    `json:"workspace,omitempty"`
	Type        Type      `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Score       float64   `json:"score"`
	Snippet     string    `json:"snippet,omitempty"`
	ModTime     time.Time `json:"mod_time"`
}

// ReindexOptions controls which scopes are rebuilt into the derived catalog.
type ReindexOptions struct {
	Scope     Scope
	Workspace string
}

// ReindexResult reports the outcome of a catalog rebuild.
type ReindexResult struct {
	IndexedFiles int       `json:"indexed_files"`
	Scope        Scope     `json:"scope,omitempty"`
	Workspace    string    `json:"workspace,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}

// HealthStats summarizes derived-catalog state for operator surfaces.
type HealthStats struct {
	IndexedFiles  int        `json:"indexed_files"`
	OrphanedFiles int        `json:"orphaned_files"`
	LastReindex   *time.Time `json:"last_reindex"`
}

// Backend captures the memory backend surface used by daemon, API, and CLI layers.
type Backend interface {
	List(scope Scope) ([]Header, error)
	Read(scope Scope, filename string) ([]byte, error)
	Write(scope Scope, filename string, content []byte) error
	Delete(scope Scope, filename string) error
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Reindex(ctx context.Context, opts ReindexOptions) (ReindexResult, error)
	LoadPromptIndex(scope Scope) (content string, truncated bool, err error)
}

// Normalize returns the normalized representation of the memory type.
func (t Type) Normalize() Type {
	return Type(strings.ToLower(strings.TrimSpace(string(t))))
}

// Validate reports whether the memory type belongs to the closed taxonomy.
func (t Type) Validate() error {
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
func DefaultScopeForType(t Type) (Scope, error) {
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
func (h *Header) Normalize() {
	h.Name = strings.TrimSpace(h.Name)
	h.Description = strings.TrimSpace(h.Description)
	h.Type = h.Type.Normalize()
	h.AgentName = strings.TrimSpace(h.AgentName)
}

// Validate reports whether the parsed memory header is complete and valid.
func (h *Header) Validate() error {
	h.Normalize()

	if h.Name == "" {
		return fmt.Errorf("memory name is required")
	}

	if err := h.Type.Validate(); err != nil {
		return err
	}

	return nil
}
