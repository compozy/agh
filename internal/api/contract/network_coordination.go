package contract

import (
	"time"

	"github.com/compozy/agh/internal/network/participation"
)

// NetworkCoordinationPayload is the workspace coordination setting plus invitation state.
type NetworkCoordinationPayload struct {
	WorkspaceID string                                `json:"workspace_id"`
	Enabled     bool                                  `json:"enabled"`
	Revision    int64                                 `json:"revision"`
	UpdatedAt   time.Time                             `json:"updated_at,omitzero"`
	UpdatedBy   string                                `json:"updated_by,omitempty"`
	Invitation  *NetworkCoordinationInvitationPayload `json:"invitation,omitempty"`
}

// NetworkCoordinationResponse wraps one coordination payload.
type NetworkCoordinationResponse struct {
	Coordination NetworkCoordinationPayload `json:"coordination"`
}

// PutNetworkCoordinationRequest toggles the workspace coordination opt-in.
type PutNetworkCoordinationRequest struct {
	Enabled bool `json:"enabled"`
}

// NetworkCoordinationInvitationPayload is the daemon-persisted dismissal state for one scope.
type NetworkCoordinationInvitationPayload struct {
	Scope       string     `json:"scope"`
	TaskID      string     `json:"task_id,omitempty"`
	Dismissed   bool       `json:"dismissed"`
	DismissedAt *time.Time `json:"dismissed_at,omitempty"`
	DismissedBy string     `json:"dismissed_by,omitempty"`
}

// PutNetworkCoordinationInvitationRequest persists or resets invitation dismissal.
// Scope is "workspace" or "task". When dismissed is false, the row is deleted (reset).
type PutNetworkCoordinationInvitationRequest struct {
	Scope     string `json:"scope"`
	TaskID    string `json:"task_id,omitempty"`
	Dismissed bool   `json:"dismissed"`
}

// NetworkUsageDetailPayload is one wake's charged usage truth.
type NetworkUsageDetailPayload struct {
	WakeID          string     `json:"wake_id"`
	TaskRunID       string     `json:"task_run_id"`
	OwnerKey        string     `json:"owner_key"`
	WorkspaceID     string     `json:"workspace_id"`
	Channel         string     `json:"channel"`
	RootID          string     `json:"root_id"`
	Depth           int        `json:"depth"`
	State           string     `json:"state"`
	UsageState      string     `json:"usage_state"`
	ChargedWallTime string     `json:"charged_wall_time"`
	InputTokens     int64      `json:"input_tokens"`
	OutputTokens    int64      `json:"output_tokens"`
	ReservedAt      time.Time  `json:"reserved_at"`
	SettledAt       *time.Time `json:"settled_at,omitempty"`
	Reason          string     `json:"reason,omitempty"`
}

// NetworkUsageSummaryPayload aggregates exactly the returned wake details.
type NetworkUsageSummaryPayload struct {
	WakeCount            int    `json:"wake_count"`
	ReservedWakeCount    int    `json:"reserved_wake_count"`
	ActualWakeCount      int    `json:"actual_wake_count"`
	UnavailableWakeCount int    `json:"unavailable_wake_count"`
	ChargedWallTime      string `json:"charged_wall_time"`
	InputTokens          int64  `json:"input_tokens"`
	OutputTokens         int64  `json:"output_tokens"`
}

// NetworkBudgetUsagePayload reports durable bound consumption for one execution owner.
type NetworkBudgetUsagePayload struct {
	OwnerKey         string    `json:"owner_key"`
	WakesUsed        int       `json:"wakes_used"`
	WallTimeUsed     string    `json:"wall_time_used"`
	InputTokensUsed  int64     `json:"input_tokens_used"`
	OutputTokensUsed int64     `json:"output_tokens_used"`
	ExhaustedReason  string    `json:"exhausted_reason,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NetworkUsageResponse is the workspace-scoped usage meter.
type NetworkUsageResponse struct {
	WorkspaceID string                      `json:"workspace_id"`
	Details     []NetworkUsageDetailPayload `json:"details"`
	Total       NetworkUsageSummaryPayload  `json:"total"`
	Budget      *NetworkBudgetUsagePayload  `json:"budget,omitempty"`
}

// Compile-time assertion that participation types remain the wire shapes.
var (
	_ = participation.Request{}
	_ = participation.Spec{}
)
