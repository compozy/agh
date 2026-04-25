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

// Operation identifies a durable memory operation surfaced in operator history.
type Operation string

const (
	// OperationWrite records a memory document write.
	OperationWrite Operation = "memory.write"
	// OperationDelete records a memory document delete.
	OperationDelete Operation = "memory.delete"
	// OperationSearch records a memory search query.
	OperationSearch Operation = "memory.search"
	// OperationReindex records a derived catalog reindex.
	OperationReindex Operation = "memory.reindex"
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

// OperationHistoryQuery filters durable memory operation history.
type OperationHistoryQuery struct {
	Scope     Scope
	Workspace string
	Operation Operation
	Since     time.Time
	Limit     int
}

// OperationRecord is one redacted durable memory operation history row.
type OperationRecord struct {
	ID        string    `json:"id"`
	Operation Operation `json:"operation"`
	Scope     Scope     `json:"scope,omitempty"`
	Workspace string    `json:"workspace,omitempty"`
	Filename  string    `json:"filename,omitempty"`
	AgentName string    `json:"agent_name,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthStats summarizes derived-catalog state for operator surfaces.
type HealthStats struct {
	IndexedFiles    int        `json:"indexed_files"`
	OrphanedFiles   int        `json:"orphaned_files"`
	LastReindex     *time.Time `json:"last_reindex"`
	OperationCount  int        `json:"operation_count"`
	LastOperationAt *time.Time `json:"last_operation_at"`
}

// Backend captures the memory backend surface used by daemon, API, and CLI layers.
type Backend interface {
	List(scope Scope) ([]Header, error)
	Read(scope Scope, filename string) ([]byte, error)
	Write(scope Scope, filename string, content []byte) error
	Delete(scope Scope, filename string) error
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Reindex(ctx context.Context, opts ReindexOptions) (ReindexResult, error)
	History(ctx context.Context, query OperationHistoryQuery) ([]OperationRecord, error)
	LoadPromptIndex(scope Scope) (content string, truncated bool, err error)
}

// ContextRefKind identifies a future runtime context reference family.
type ContextRefKind string

const (
	// ContextRefFile is reserved for future @file references.
	ContextRefFile ContextRefKind = "file"
	// ContextRefFolder is reserved for future @folder references.
	ContextRefFolder ContextRefKind = "folder"
	// ContextRefGit is reserved for future @git references.
	ContextRefGit ContextRefKind = "git"
	// ContextRefURL is reserved for future @url references.
	ContextRefURL ContextRefKind = "url"
)

// ContextRef describes a future memory context reference. Task 07 only defines
// this contract; prompt assembly must not call a resolver yet.
type ContextRef struct {
	Kind ContextRefKind `json:"kind"`
	URI  string         `json:"uri"`
}

// TokenBudget is the future bounded context budget passed to context resolvers.
type TokenBudget struct {
	MaxTokens int `json:"max_tokens"`
}

// ResolvedContext is the future prompt-safe context result produced by a resolver.
type ResolvedContext struct {
	Items       []ResolvedContextItem `json:"items"`
	UsedTokens  int                   `json:"used_tokens"`
	Truncated   bool                  `json:"truncated"`
	Redactions  []string              `json:"redactions,omitempty"`
	GeneratedAt time.Time             `json:"generated_at"`
}

// ResolvedContextItem is one future prompt-safe context item.
type ResolvedContextItem struct {
	Ref       ContextRef `json:"ref"`
	Title     string     `json:"title,omitempty"`
	Content   string     `json:"content"`
	Tokens    int        `json:"tokens"`
	Truncated bool       `json:"truncated"`
}

// ContextRefResolver is the future narrow seam for @file/@folder/@git/@url
// resolution. It is intentionally not wired into runtime prompt assembly yet.
type ContextRefResolver interface {
	Resolve(ctx context.Context, refs []ContextRef, budget TokenBudget) (ResolvedContext, error)
}

// ProviderHookEvent identifies a future memory provider lifecycle hook point.
type ProviderHookEvent string

const (
	// ProviderHookOnTurnStart is reserved for future pre-turn provider hooks.
	ProviderHookOnTurnStart ProviderHookEvent = "on_turn_start"
	// ProviderHookOnSessionEnd is reserved for future session-end provider hooks.
	ProviderHookOnSessionEnd ProviderHookEvent = "on_session_end"
	// ProviderHookOnPreCompress is reserved for future pre-compression provider hooks.
	ProviderHookOnPreCompress ProviderHookEvent = "on_pre_compress"
)

// ProviderHookRequest is the future provider-hook input envelope.
type ProviderHookRequest struct {
	Event       ProviderHookEvent `json:"event"`
	SessionID   string            `json:"session_id,omitempty"`
	TurnID      string            `json:"turn_id,omitempty"`
	Workspace   string            `json:"workspace,omitempty"`
	TokenBudget TokenBudget       `json:"token_budget"`
}

// ProviderHookResult is the future provider-hook result envelope.
type ProviderHookResult struct {
	Context ResolvedContext `json:"context"`
	Notes   []string        `json:"notes,omitempty"`
}

// ProviderHookRunner is the future narrow seam for memory provider lifecycle
// hooks. Task 07 only defines it; no provider hook is executed by runtime prompts.
type ProviderHookRunner interface {
	RunMemoryHook(ctx context.Context, req ProviderHookRequest) (ProviderHookResult, error)
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

// Normalize returns the normalized operation string.
func (o Operation) Normalize() Operation {
	return Operation(strings.ToLower(strings.TrimSpace(string(o))))
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
