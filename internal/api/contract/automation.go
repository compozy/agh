package contract

import (
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
)

// AutomationResourceStatusPayload reports total and enabled counts for one
// automation definition family.
type AutomationResourceStatusPayload struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
}

// AutomationHealthPayload describes additive automation health state inside
// the observe health response.
type AutomationHealthPayload struct {
	Enabled          bool                            `json:"enabled"`
	Jobs             AutomationResourceStatusPayload `json:"jobs"`
	Triggers         AutomationResourceStatusPayload `json:"triggers"`
	SchedulerRunning bool                            `json:"scheduler_running"`
	NextFire         *time.Time                      `json:"next_fire,omitempty"`
}

// JobPayload is the shared automation job response payload.
type JobPayload struct {
	ID          string                        `json:"id"`
	Scope       automationpkg.AutomationScope `json:"scope"`
	Name        string                        `json:"name"`
	AgentName   string                        `json:"agent_name"`
	WorkspaceID string                        `json:"workspace_id,omitempty"`
	Prompt      string                        `json:"prompt"`
	Schedule    *automationpkg.ScheduleSpec   `json:"schedule,omitempty"`
	Enabled     bool                          `json:"enabled"`
	Retry       automationpkg.RetryConfig     `json:"retry"`
	FireLimit   automationpkg.FireLimitConfig `json:"fire_limit"`
	Source      automationpkg.JobSource       `json:"source"`
	CreatedAt   time.Time                     `json:"created_at"`
	UpdatedAt   time.Time                     `json:"updated_at"`
	NextRun     *time.Time                    `json:"next_run,omitempty"`
}

// TriggerPayload is the shared automation trigger response payload.
type TriggerPayload struct {
	ID           string                        `json:"id"`
	Scope        automationpkg.AutomationScope `json:"scope"`
	Name         string                        `json:"name"`
	AgentName    string                        `json:"agent_name"`
	WorkspaceID  string                        `json:"workspace_id,omitempty"`
	Prompt       string                        `json:"prompt"`
	Event        string                        `json:"event"`
	Filter       map[string]string             `json:"filter,omitempty"`
	Enabled      bool                          `json:"enabled"`
	Retry        automationpkg.RetryConfig     `json:"retry"`
	FireLimit    automationpkg.FireLimitConfig `json:"fire_limit"`
	Source       automationpkg.JobSource       `json:"source"`
	WebhookID    string                        `json:"webhook_id,omitempty"`
	EndpointSlug string                        `json:"endpoint_slug,omitempty"`
	CreatedAt    time.Time                     `json:"created_at"`
	UpdatedAt    time.Time                     `json:"updated_at"`
}

// RunPayload is the shared automation run response payload.
type RunPayload struct {
	ID        string                  `json:"id"`
	JobID     string                  `json:"job_id,omitempty"`
	TriggerID string                  `json:"trigger_id,omitempty"`
	SessionID string                  `json:"session_id,omitempty"`
	Status    automationpkg.RunStatus `json:"status"`
	Attempt   int                     `json:"attempt"`
	StartedAt *time.Time              `json:"started_at,omitempty"`
	EndedAt   *time.Time              `json:"ended_at,omitempty"`
	Error     string                  `json:"error,omitempty"`
}

// WebhookDeliveryPayload is the shared webhook dispatch response payload.
type WebhookDeliveryPayload struct {
	Matched int          `json:"matched"`
	Runs    []RunPayload `json:"runs,omitempty"`
}

// CreateJobRequest is the shared automation job create payload.
type CreateJobRequest struct {
	Scope       automationpkg.AutomationScope  `json:"scope"`
	Name        string                         `json:"name"`
	AgentName   string                         `json:"agent_name"`
	WorkspaceID string                         `json:"workspace_id,omitempty"`
	Prompt      string                         `json:"prompt"`
	Schedule    automationpkg.ScheduleSpec     `json:"schedule"`
	Enabled     *bool                          `json:"enabled,omitempty"`
	Retry       *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit   *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
}

// UpdateJobRequest is the shared automation job patch payload.
type UpdateJobRequest struct {
	Name        *string                        `json:"name,omitempty"`
	AgentName   *string                        `json:"agent_name,omitempty"`
	WorkspaceID *string                        `json:"workspace_id,omitempty"`
	Prompt      *string                        `json:"prompt,omitempty"`
	Schedule    *automationpkg.ScheduleSpec    `json:"schedule,omitempty"`
	Enabled     *bool                          `json:"enabled,omitempty"`
	Retry       *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit   *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
}

// HasChanges reports whether the patch includes any mutable field.
func (r UpdateJobRequest) HasChanges() bool {
	return r.Name != nil ||
		r.AgentName != nil ||
		r.WorkspaceID != nil ||
		r.Prompt != nil ||
		r.Schedule != nil ||
		r.Enabled != nil ||
		r.Retry != nil ||
		r.FireLimit != nil
}

// CreateTriggerRequest is the shared automation trigger create payload.
type CreateTriggerRequest struct {
	Scope         automationpkg.AutomationScope  `json:"scope"`
	Name          string                         `json:"name"`
	AgentName     string                         `json:"agent_name"`
	WorkspaceID   string                         `json:"workspace_id,omitempty"`
	Prompt        string                         `json:"prompt"`
	Event         string                         `json:"event"`
	Filter        map[string]string              `json:"filter,omitempty"`
	Enabled       *bool                          `json:"enabled,omitempty"`
	Retry         *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit     *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
	WebhookID     string                         `json:"webhook_id,omitempty"`
	EndpointSlug  string                         `json:"endpoint_slug,omitempty"`
	WebhookSecret string                         `json:"webhook_secret,omitempty"`
}

// UpdateTriggerRequest is the shared automation trigger patch payload.
type UpdateTriggerRequest struct {
	Name          *string                        `json:"name,omitempty"`
	AgentName     *string                        `json:"agent_name,omitempty"`
	WorkspaceID   *string                        `json:"workspace_id,omitempty"`
	Prompt        *string                        `json:"prompt,omitempty"`
	Event         *string                        `json:"event,omitempty"`
	Filter        map[string]string              `json:"filter,omitempty"`
	Enabled       *bool                          `json:"enabled,omitempty"`
	Retry         *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit     *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
	WebhookID     *string                        `json:"webhook_id,omitempty"`
	EndpointSlug  *string                        `json:"endpoint_slug,omitempty"`
	WebhookSecret *string                        `json:"webhook_secret,omitempty"`
}

// HasChanges reports whether the patch includes any mutable field.
func (r UpdateTriggerRequest) HasChanges() bool {
	return r.Name != nil ||
		r.AgentName != nil ||
		r.WorkspaceID != nil ||
		r.Prompt != nil ||
		r.Event != nil ||
		r.Filter != nil ||
		r.Enabled != nil ||
		r.Retry != nil ||
		r.FireLimit != nil ||
		r.WebhookID != nil ||
		r.EndpointSlug != nil ||
		r.WebhookSecret != nil
}
