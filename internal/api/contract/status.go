package contract

import "time"

const (
	// StatusSchemaVersion identifies the public status/doctor payload contract.
	StatusSchemaVersion = "2026-05-20"
)

// StatusPayload is the hard-cut runtime status surface shared by HTTP, UDS, and CLI JSON.
type StatusPayload struct {
	SchemaVersion string                       `json:"schema_version"`
	GeneratedAt   time.Time                    `json:"generated_at"`
	Daemon        DaemonStatusPayload          `json:"daemon"`
	Sessions      SessionAggregatePayload      `json:"sessions"`
	Health        ObserveHealthPayload         `json:"health"`
	Memory        MemoryHealthPayload          `json:"memory"`
	Automation    AutomationHealthPayload      `json:"automation"`
	Tasks         TaskHealthPayload            `json:"tasks"`
	Bridges       BridgeAggregateHealthPayload `json:"bridges"`
	Providers     []ProviderStatusPayload      `json:"providers,omitempty"`
	MCPServers    []MCPServerStatusPayload     `json:"mcp_servers,omitempty"`
	Skills        SkillRuntimeStatusPayload    `json:"skills"`
	Config        ConfigRuntimeStatusPayload   `json:"config"`
	LogTail       LogTailStatusPayload         `json:"log_tail"`
}

// DoctorPayload is the diagnostic probe result shared by HTTP, UDS, and CLI JSON.
type DoctorPayload struct {
	SchemaVersion string               `json:"schema_version"`
	GeneratedAt   time.Time            `json:"generated_at"`
	DurationMS    int64                `json:"duration_ms"`
	Status        string               `json:"status"`
	Summary       DoctorSummaryPayload `json:"summary"`
	Items         []DiagnosticItem     `json:"items"`
}

// DoctorSummaryPayload provides stable severity counters for agents.
type DoctorSummaryPayload struct {
	Total            int            `json:"total"`
	CountsBySeverity map[string]int `json:"counts_by_severity"`
}

// ProviderStatusPayload reports one provider's auth readiness in the status surface.
type ProviderStatusPayload struct {
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name,omitempty"`
	Default          bool       `json:"default"`
	Mode             string     `json:"mode,omitempty"`
	EnvPolicy        string     `json:"env_policy,omitempty"`
	HomePolicy       string     `json:"home_policy,omitempty"`
	State            string     `json:"state"`
	Code             string     `json:"code,omitempty"`
	Message          string     `json:"message,omitempty"`
	StatusCommand    string     `json:"status_command,omitempty"`
	LoginCommand     string     `json:"login_command,omitempty"`
	LastProbeAt      *time.Time `json:"last_probe_at,omitempty"`
	SuggestedCommand string     `json:"suggested_command,omitempty"`
}

// MCPServerStatusPayload reports configured MCP server availability.
type MCPServerStatusPayload struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	AuthStatus    string `json:"auth_status,omitempty"`
	Configured    bool   `json:"configured"`
	Initialized   bool   `json:"initialized"`
	State         string `json:"state"`
	Probe         string `json:"probe,omitempty"`
	ToolCount     int    `json:"tool_count,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Diagnostic    string `json:"diagnostic,omitempty"`
	Transport     string `json:"transport,omitempty"`
	RuntimeStatus string `json:"runtime_status"`
}

// SkillRuntimeStatusPayload summarizes the skill registry's availability.
type SkillRuntimeStatusPayload struct {
	RuntimeAvailable bool                     `json:"runtime_available"`
	DiscoveredCount  int                      `json:"discovered_count"`
	DisabledCount    int                      `json:"disabled_count"`
	Diagnostics      []SkillDiagnosticPayload `json:"diagnostics,omitempty"`
}

// ConfigRuntimeStatusPayload reports daemon config validation and apply lifecycle state.
type ConfigRuntimeStatusPayload struct {
	Status          string `json:"status"`
	Validated       bool   `json:"validated"`
	ValidationError string `json:"validation_error,omitempty"`
	HomeDir         string `json:"home_dir,omitempty"`
	ConfigFile      string `json:"config_file,omitempty"`
	RestartRequired bool   `json:"restart_required"`
	ApplyState      string `json:"apply_state"`
}

// LogTailStatusPayload reports the log-tail capability advertised by settings.
type LogTailStatusPayload struct {
	Available bool   `json:"available"`
	Status    string `json:"status"`
}

// SessionAggregatePayload summarizes current session state.
type SessionAggregatePayload struct {
	Active   int            `json:"active"`
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status,omitempty"`
}

// TaskHealthPayload exposes observer-owned task health in the status surface.
type TaskHealthPayload struct {
	Status                     string                    `json:"status"`
	QueueDepthTotal            int                       `json:"queue_depth_total"`
	OldestQueuedAt             *time.Time                `json:"oldest_queued_at,omitempty"`
	OldestQueueAgeMilli        int64                     `json:"oldest_queue_age_ms"`
	QueueDepth                 []TaskQueueDepthPayload   `json:"queue_depth,omitempty"`
	StuckRuns                  []StuckTaskRunPayload     `json:"stuck_runs,omitempty"`
	ActiveOrphanRuns           int                       `json:"active_orphan_runs"`
	TaskTotals                 []TaskStatusTotalPayload  `json:"task_totals,omitempty"`
	RunTotals                  []TaskRunTotalPayload     `json:"run_totals,omitempty"`
	OwnerTotals                []TaskOwnerTotalPayload   `json:"owner_totals,omitempty"`
	ForcedStopsSinceStart      int                       `json:"forced_stops_since_start"`
	DuplicateIngressSinceStart int                       `json:"duplicate_ingress_since_start"`
	ChannelMismatchSinceStart  int                       `json:"channel_mismatch_since_start"`
	RecoverySinceStart         TaskRecoveryTotalsPayload `json:"recovery_since_start"`
}

type TaskQueueDepthPayload struct {
	NetworkChannel      string     `json:"network_channel,omitempty"`
	Count               int        `json:"count"`
	OldestQueuedAt      *time.Time `json:"oldest_queued_at,omitempty"`
	OldestQueueAgeMilli int64      `json:"oldest_queue_age_ms"`
}

type StuckTaskRunPayload struct {
	TaskID         string `json:"task_id"`
	RunID          string `json:"run_id"`
	Status         string `json:"status"`
	OriginKind     string `json:"origin_kind"`
	NetworkChannel string `json:"network_channel,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	AgeMillis      int64  `json:"age_ms"`
}

type TaskStatusTotalPayload struct {
	Scope          string `json:"scope"`
	Status         string `json:"status"`
	NetworkChannel string `json:"network_channel,omitempty"`
	Count          int    `json:"count"`
}

type TaskRunTotalPayload struct {
	Status         string `json:"status"`
	OriginKind     string `json:"origin_kind"`
	NetworkChannel string `json:"network_channel,omitempty"`
	Count          int    `json:"count"`
}

type TaskOwnerTotalPayload struct {
	OwnerKind string `json:"owner_kind"`
	OwnerRef  string `json:"owner_ref"`
	Count     int    `json:"count"`
}

type TaskRecoveryTotalsPayload struct {
	Requeued      int `json:"requeued"`
	MarkedRunning int `json:"marked_running"`
	Failed        int `json:"failed"`
}
