package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// HookSource identifies where a hook was declared.
type HookSource int

const (
	HookSourceNative HookSource = iota
	HookSourceConfig
	HookSourceAgentDefinition
	HookSourceSkill
)

var hookSourceNames = map[HookSource]string{
	HookSourceNative:          "native",
	HookSourceConfig:          "config",
	HookSourceAgentDefinition: "agent_definition",
	HookSourceSkill:           "skill",
}

// String returns the stable text form for the hook source.
func (s HookSource) String() string {
	name, ok := hookSourceNames[s]
	if !ok {
		return ""
	}
	return name
}

// MarshalText encodes the source as a string.
func (s HookSource) MarshalText() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return []byte(s.String()), nil
}

// UnmarshalText decodes the source from a string value.
func (s *HookSource) UnmarshalText(text []byte) error {
	value := strings.TrimSpace(string(text))
	for source, name := range hookSourceNames {
		if value == name {
			*s = source
			return nil
		}
	}
	return fmt.Errorf("hooks: invalid hook source %q", value)
}

// Validate ensures the source is one of the documented values.
func (s HookSource) Validate() error {
	if _, ok := hookSourceNames[s]; !ok {
		return fmt.Errorf("hooks: invalid hook source %d", s)
	}
	return nil
}

// HookMode controls whether a hook runs inline or in the background.
type HookMode string

const (
	HookModeSync  HookMode = "sync"
	HookModeAsync HookMode = "async"
)

// Validate ensures the mode is supported.
func (m HookMode) Validate() error {
	switch m {
	case HookModeSync, HookModeAsync:
		return nil
	default:
		return fmt.Errorf("hooks: invalid hook mode %q", m)
	}
}

// HookExecutorKind identifies the execution boundary for a hook.
type HookExecutorKind string

const (
	HookExecutorNative     HookExecutorKind = "native"
	HookExecutorSubprocess HookExecutorKind = "subprocess"
	HookExecutorWASM       HookExecutorKind = "wasm"
)

// Validate ensures the executor kind is supported.
func (k HookExecutorKind) Validate() error {
	switch k {
	case HookExecutorNative, HookExecutorSubprocess, HookExecutorWASM:
		return nil
	default:
		return fmt.Errorf("hooks: invalid hook executor kind %q", k)
	}
}

// HookRunOutcome classifies the result of one hook execution.
type HookRunOutcome string

const (
	HookRunOutcomeApplied  HookRunOutcome = "applied"
	HookRunOutcomeDenied   HookRunOutcome = "denied"
	HookRunOutcomeFailed   HookRunOutcome = "failed"
	HookRunOutcomeSkipped  HookRunOutcome = "skipped"
	HookRunOutcomeDropped  HookRunOutcome = "dropped"
	HookRunOutcomeRejected HookRunOutcome = "rejected"
)

// HookMatcher narrows when a hook is eligible to run.
type HookMatcher struct {
	AgentName          string `json:"agent_name,omitempty" yaml:"agent_name,omitempty"`
	AgentType          string `json:"agent_type,omitempty" yaml:"agent_type,omitempty"`
	WorkspaceID        string `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
	WorkspaceRoot      string `json:"workspace_root,omitempty" yaml:"workspace_root,omitempty"`
	SessionType        string `json:"session_type,omitempty" yaml:"session_type,omitempty"`
	InputClass         string `json:"input_class,omitempty" yaml:"input_class,omitempty"`
	ACPEventType       string `json:"acp_event_type,omitempty" yaml:"acp_event_type,omitempty"`
	TurnID             string `json:"turn_id,omitempty" yaml:"turn_id,omitempty"`
	ToolName           string `json:"tool_name,omitempty" yaml:"tool_name,omitempty"`
	ToolNamespace      string `json:"tool_namespace,omitempty" yaml:"tool_namespace,omitempty"`
	ToolReadOnly       *bool  `json:"tool_read_only,omitempty" yaml:"tool_read_only,omitempty"`
	DecisionClass      string `json:"decision_class,omitempty" yaml:"decision_class,omitempty"`
	MessageRole        string `json:"message_role,omitempty" yaml:"message_role,omitempty"`
	MessageDeltaType   string `json:"message_delta_type,omitempty" yaml:"message_delta_type,omitempty"`
	CompactionReason   string `json:"compaction_reason,omitempty" yaml:"compaction_reason,omitempty"`
	CompactionStrategy string `json:"compaction_strategy,omitempty" yaml:"compaction_strategy,omitempty"`
}

// HookDecl is the declarative record supplied by config, agent definitions, or skills.
type HookDecl struct {
	Name         string            `json:"name" yaml:"name"`
	Event        HookEvent         `json:"event" yaml:"event"`
	Source       HookSource        `json:"source" yaml:"source"`
	Mode         HookMode          `json:"mode,omitempty" yaml:"mode,omitempty"`
	Required     bool              `json:"required,omitempty" yaml:"required,omitempty"`
	Priority     int               `json:"priority,omitempty" yaml:"priority,omitempty"`
	Timeout      time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Matcher      HookMatcher       `json:"matcher,omitempty" yaml:"matcher,omitempty"`
	ExecutorKind HookExecutorKind  `json:"executor_kind,omitempty" yaml:"executor_kind,omitempty"`
	Command      string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args         []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Executor is the execution seam for hook implementations.
type Executor interface {
	Kind() HookExecutorKind
	Execute(ctx context.Context, hook RegisteredHook, payload []byte) ([]byte, error)
}

// RegisteredHook is the normalized hook ready for dispatch.
type RegisteredHook struct {
	Name     string
	Event    HookEvent
	Source   HookSource
	Mode     HookMode
	Required bool
	Priority int
	Timeout  time.Duration
	Matcher  HookMatcher
	Executor Executor
	Metadata map[string]string
}

// Validate ensures the registered hook satisfies the task-01 invariants.
func (h RegisteredHook) Validate() error {
	if strings.TrimSpace(h.Name) == "" {
		return fmt.Errorf("hooks: hook name is required")
	}
	if err := h.Event.Validate(); err != nil {
		return err
	}
	if err := h.Source.Validate(); err != nil {
		return err
	}
	if err := h.Mode.Validate(); err != nil {
		return err
	}
	if h.Required && h.Mode != HookModeSync {
		return fmt.Errorf("hooks: required hook %q must use sync mode", h.Name)
	}
	if h.Mode == HookModeSync && !h.Event.SyncEligible() {
		return fmt.Errorf("hooks: event %q does not allow sync hooks", h.Event)
	}
	if h.Timeout < 0 {
		return fmt.Errorf("hooks: hook %q timeout must be non-negative", h.Name)
	}
	return nil
}

// ResolvedHook is the registry snapshot record with executor binding attached.
type ResolvedHook struct {
	RegisteredHook
	Decl HookDecl
}

// Validate ensures the resolved hook is internally consistent.
func (h ResolvedHook) Validate() error {
	if err := h.RegisteredHook.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(h.Decl.Name) == "" {
		return nil
	}
	if h.Decl.Name != h.Name {
		return fmt.Errorf("hooks: resolved hook %q does not match declaration %q", h.Name, h.Decl.Name)
	}
	return nil
}

// HookRunRecord captures one hook execution for observability and audit.
type HookRunRecord struct {
	HookName      string          `json:"hook_name"`
	Event         HookEvent       `json:"event"`
	Source        HookSource      `json:"source"`
	Mode          HookMode        `json:"mode"`
	Duration      time.Duration   `json:"duration"`
	Outcome       HookRunOutcome  `json:"outcome"`
	DispatchDepth int             `json:"dispatch_depth"`
	PatchApplied  json.RawMessage `json:"patch_applied,omitempty"`
	Error         string          `json:"error,omitempty"`
	Required      bool            `json:"required,omitempty"`
}
