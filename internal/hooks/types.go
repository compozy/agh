package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HookSource identifies where a hook was declared.
type HookSource uint8

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

// HookSkillSource captures the existing skill-registry precedence without
// importing internal/skills into the hooks base package.
type HookSkillSource string

const (
	HookSkillSourceBundled     HookSkillSource = "bundled"
	HookSkillSourceMarketplace HookSkillSource = "marketplace"
	HookSkillSourceUser        HookSkillSource = "user"
	HookSkillSourceAdditional  HookSkillSource = "additional"
	HookSkillSourceWorkspace   HookSkillSource = "workspace"
)

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

// Validate ensures the skill source is one of the documented values when set.
func (s HookSkillSource) Validate() error {
	switch s {
	case "":
		return nil
	case HookSkillSourceBundled,
		HookSkillSourceMarketplace,
		HookSkillSourceUser,
		HookSkillSourceAdditional,
		HookSkillSourceWorkspace:
		return nil
	default:
		return fmt.Errorf("hooks: invalid hook skill source %q", s)
	}
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

// Validate ensures the outcome is one of the documented execution results.
func (o HookRunOutcome) Validate() error {
	switch o {
	case HookRunOutcomeApplied,
		HookRunOutcomeDenied,
		HookRunOutcomeFailed,
		HookRunOutcomeSkipped,
		HookRunOutcomeDropped,
		HookRunOutcomeRejected:
		return nil
	default:
		return fmt.Errorf("hooks: invalid hook run outcome %q", o)
	}
}

// HookMatcher narrows when a hook is eligible to run.
type HookMatcher struct {
	AgentName          string           `json:"agent_name,omitempty"          yaml:"agent_name,omitempty"`
	AgentType          string           `json:"agent_type,omitempty"          yaml:"agent_type,omitempty"`
	WorkspaceID        string           `json:"workspace_id,omitempty"        yaml:"workspace_id,omitempty"`
	WorkspaceRoot      string           `json:"workspace_root,omitempty"      yaml:"workspace_root,omitempty"`
	SessionType        string           `json:"session_type,omitempty"        yaml:"session_type,omitempty"`
	SandboxID          string           `json:"sandbox_id,omitempty"          yaml:"sandbox_id,omitempty"`
	SandboxBackend     string           `json:"sandbox_backend,omitempty"     yaml:"sandbox_backend,omitempty"`
	SandboxProfile     string           `json:"sandbox_profile,omitempty"     yaml:"sandbox_profile,omitempty"`
	SyncDirection      string           `json:"sync_direction,omitempty"      yaml:"sync_direction,omitempty"`
	InputClass         string           `json:"input_class,omitempty"         yaml:"input_class,omitempty"`
	ACPEventType       string           `json:"acp_event_type,omitempty"      yaml:"acp_event_type,omitempty"`
	TurnID             string           `json:"turn_id,omitempty"             yaml:"turn_id,omitempty"`
	ToolID             string           `json:"tool_id,omitempty"             yaml:"tool_id,omitempty"`
	ToolName           string           `json:"tool_name,omitempty"           yaml:"tool_name,omitempty"`
	ToolReadOnly       *bool            `json:"tool_read_only,omitempty"      yaml:"tool_read_only,omitempty"`
	DecisionClass      string           `json:"decision_class,omitempty"      yaml:"decision_class,omitempty"`
	MessageRole        string           `json:"message_role,omitempty"        yaml:"message_role,omitempty"`
	MessageDeltaType   string           `json:"message_delta_type,omitempty"  yaml:"message_delta_type,omitempty"`
	CompactionReason   string           `json:"compaction_reason,omitempty"   yaml:"compaction_reason,omitempty"`
	CompactionStrategy string           `json:"compaction_strategy,omitempty" yaml:"compaction_strategy,omitempty"`
	Autonomy           *AutonomyMatcher `json:"autonomy,omitempty"            yaml:"autonomy,omitempty"`
}

// AutonomyMatcher narrows autonomy hooks by task, coordinator, and spawn correlation fields.
type AutonomyMatcher struct {
	TaskID                string `json:"task_id,omitempty"                 yaml:"task_id,omitempty"`
	RunID                 string `json:"run_id,omitempty"                  yaml:"run_id,omitempty"`
	WorkflowID            string `json:"workflow_id,omitempty"             yaml:"workflow_id,omitempty"`
	CoordinationChannelID string `json:"coordination_channel_id,omitempty" yaml:"coordination_channel_id,omitempty"`
	CoordinatorSessionID  string `json:"coordinator_session_id,omitempty"  yaml:"coordinator_session_id,omitempty"`
	ParentSessionID       string `json:"parent_session_id,omitempty"       yaml:"parent_session_id,omitempty"`
	RootSessionID         string `json:"root_session_id,omitempty"         yaml:"root_session_id,omitempty"`
	ChildSessionID        string `json:"child_session_id,omitempty"        yaml:"child_session_id,omitempty"`
	SpawnRole             string `json:"spawn_role,omitempty"              yaml:"spawn_role,omitempty"`
	ReleaseReason         string `json:"release_reason,omitempty"          yaml:"release_reason,omitempty"`
}

// HookDecl is the declarative record supplied by config, agent definitions, or skills.
type HookDecl struct {
	Name         string            `json:"name"                    yaml:"name"`
	Event        HookEvent         `json:"event"                   yaml:"event"`
	Mode         HookMode          `json:"mode,omitempty"          yaml:"mode,omitempty"`
	Matcher      HookMatcher       `json:"matcher"                 yaml:"matcher,omitempty"`
	ExecutorKind HookExecutorKind  `json:"executor_kind,omitempty" yaml:"executor_kind,omitempty"`
	Command      string            `json:"command,omitempty"       yaml:"command,omitempty"`
	Args         []string          `json:"args,omitempty"          yaml:"args,omitempty"`
	WorkingDir   string            `json:"-"                       yaml:"-"`
	Env          map[string]string `json:"env,omitempty"           yaml:"env,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"      yaml:"metadata,omitempty"`
	SkillSource  HookSkillSource   `json:"-"                       yaml:"-"`
	Timeout      time.Duration     `json:"timeout,omitempty"       yaml:"timeout,omitempty"`
	Priority     int               `json:"priority,omitempty"      yaml:"priority,omitempty"`
	Enabled      *bool             `json:"enabled,omitempty"       yaml:"enabled,omitempty"`
	Source       HookSource        `json:"source"                  yaml:"source"`
	Required     bool              `json:"required,omitempty"      yaml:"required,omitempty"`
	PrioritySet  bool              `json:"-"                       yaml:"-"`
}

// EnabledValue reports whether a declaration participates in dispatch.
func (d HookDecl) EnabledValue() bool {
	return d.Enabled == nil || *d.Enabled
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
func (h *ResolvedHook) Validate() error {
	if h == nil {
		return errors.New("hooks: resolved hook is required")
	}
	if err := h.RegisteredHook.Validate(); err != nil {
		return err
	}
	if h.Executor == nil {
		return fmt.Errorf("hooks: resolved hook %q executor is required", h.Name)
	}
	if strings.TrimSpace(h.Decl.Name) == "" {
		return nil
	}
	if err := h.Decl.SkillSource.Validate(); err != nil {
		return err
	}
	if h.Decl.ExecutorKind != "" && h.Executor.Kind() != h.Decl.ExecutorKind {
		return fmt.Errorf(
			"hooks: resolved hook %q executor kind %q does not match declaration %q",
			h.Name,
			h.Executor.Kind(),
			h.Decl.ExecutorKind,
		)
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
	RecordedAt    time.Time       `json:"recorded_at"`
}
