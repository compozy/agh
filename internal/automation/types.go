package automation

import "time"

// DefaultTimezone is the default schedule timezone used by automation config.
const DefaultTimezone = "UTC"

// DefaultMaxConcurrentJobs is the default global automation concurrency limit.
const DefaultMaxConcurrentJobs = 5

// AutomationScope identifies the visibility boundary of an automation resource.
type AutomationScope string

const (
	// AutomationScopeGlobal targets daemon-wide automation without a workspace binding.
	AutomationScopeGlobal AutomationScope = "global"
	// AutomationScopeWorkspace targets automation bound to a specific workspace.
	AutomationScopeWorkspace AutomationScope = "workspace"
)

// JobSource identifies where a job or trigger definition originated.
type JobSource string

const (
	// JobSourceConfig identifies a TOML-backed automation definition.
	JobSourceConfig JobSource = "config"
	// JobSourceDynamic identifies a runtime-created automation definition.
	JobSourceDynamic JobSource = "dynamic"
)

// ScheduleMode identifies how a scheduled job determines its next fire time.
type ScheduleMode string

const (
	// ScheduleModeCron evaluates a cron expression.
	ScheduleModeCron ScheduleMode = "cron"
	// ScheduleModeEvery evaluates a Go duration interval.
	ScheduleModeEvery ScheduleMode = "every"
	// ScheduleModeAt evaluates a one-shot RFC3339 timestamp.
	ScheduleModeAt ScheduleMode = "at"
)

// RetryStrategy identifies how failed runs should be retried.
type RetryStrategy string

const (
	// RetryStrategyNone disables retries after a failed run.
	RetryStrategyNone RetryStrategy = "none"
	// RetryStrategyBackoff retries failed runs with exponential backoff.
	RetryStrategyBackoff RetryStrategy = "backoff"
)

// RunStatus identifies the current lifecycle state of an automation run.
type RunStatus string

const (
	// RunScheduled reports a run that has been accepted but not yet started.
	RunScheduled RunStatus = "scheduled"
	// RunRunning reports a run that is actively dispatching or executing.
	RunRunning RunStatus = "running"
	// RunCompleted reports a run that finished successfully.
	RunCompleted RunStatus = "completed"
	// RunFailed reports a run that finished with an error.
	RunFailed RunStatus = "failed"
	// RunCancelled reports a run that was cancelled before completion.
	RunCancelled RunStatus = "cancelled"
)

// ActivationSource identifies which ingress path produced an activation envelope.
type ActivationSource string

const (
	// ActivationSourceObserver identifies observer-backed trigger ingress.
	ActivationSourceObserver ActivationSource = "observer"
	// ActivationSourceHook identifies hook-backed trigger ingress.
	ActivationSourceHook ActivationSource = "hook"
	// ActivationSourceWebhook identifies external webhook ingress.
	ActivationSourceWebhook ActivationSource = "webhook"
	// ActivationSourceExtension identifies extension-provided ingress.
	ActivationSourceExtension ActivationSource = "extension"
)

// Job is the canonical scheduled automation definition used by runtime and storage layers.
type Job struct {
	ID          string          `json:"id"`
	Scope       AutomationScope `json:"scope"`
	Name        string          `json:"name"`
	AgentName   string          `json:"agent_name"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Prompt      string          `json:"prompt"`
	Schedule    *ScheduleSpec   `json:"schedule,omitempty"`
	Enabled     bool            `json:"enabled"`
	Retry       RetryConfig     `json:"retry"`
	FireLimit   FireLimitConfig `json:"fire_limit"`
	Source      JobSource       `json:"source"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ScheduleSpec describes how a job should be scheduled.
type ScheduleSpec struct {
	Mode     ScheduleMode `json:"mode" toml:"mode"`
	Expr     string       `json:"expr,omitempty" toml:"expr,omitempty"`
	Interval string       `json:"interval,omitempty" toml:"interval,omitempty"`
	Time     string       `json:"time,omitempty" toml:"time,omitempty"`
}

// Trigger is the canonical event-driven automation definition used by runtime and storage layers.
type Trigger struct {
	ID           string            `json:"id"`
	Scope        AutomationScope   `json:"scope"`
	Name         string            `json:"name"`
	AgentName    string            `json:"agent_name"`
	WorkspaceID  string            `json:"workspace_id,omitempty"`
	Prompt       string            `json:"prompt"`
	Event        string            `json:"event"`
	Filter       map[string]string `json:"filter,omitempty"`
	Enabled      bool              `json:"enabled"`
	Retry        RetryConfig       `json:"retry"`
	FireLimit    FireLimitConfig   `json:"fire_limit"`
	Source       JobSource         `json:"source"`
	WebhookID    string            `json:"webhook_id,omitempty"`
	EndpointSlug string            `json:"endpoint_slug,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// RetryConfig defines retry behavior for a failed automation run.
type RetryConfig struct {
	Strategy   RetryStrategy `json:"strategy" toml:"strategy"`
	MaxRetries int           `json:"max_retries" toml:"max_retries"`
	BaseDelay  string        `json:"base_delay" toml:"base_delay"`
}

// FireLimitConfig caps how often a job or trigger may fire within a rolling window.
type FireLimitConfig struct {
	Max    int    `json:"max" toml:"max"`
	Window string `json:"window" toml:"window"`
}

// Run records the execution state of a single automation fire.
type Run struct {
	ID        string     `json:"id"`
	JobID     string     `json:"job_id,omitempty"`
	TriggerID string     `json:"trigger_id,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	Status    RunStatus  `json:"status"`
	Attempt   int        `json:"attempt"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// ActivationEnvelope is the normalized trigger input regardless of source.
type ActivationEnvelope struct {
	Kind        string           `json:"kind"`
	Scope       AutomationScope  `json:"scope"`
	WorkspaceID string           `json:"workspace_id,omitempty"`
	Source      ActivationSource `json:"source"`
	Data        map[string]any   `json:"data"`
}
