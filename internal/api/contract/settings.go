package contract

import (
	"time"

	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
)

type SettingsScopeKind string

const (
	SettingsScopeGlobal    SettingsScopeKind = "global"
	SettingsScopeWorkspace SettingsScopeKind = "workspace"
	SettingsScopeAgent     SettingsScopeKind = "agent"
)

type SettingsGlobalScopeKind string

const (
	SettingsGlobalScope SettingsGlobalScopeKind = "global"
)

type SettingsAgentScopeKind string

const (
	SettingsAgentScopeGlobal SettingsAgentScopeKind = "global"
	SettingsAgentScopeAgent  SettingsAgentScopeKind = "agent"
)

type SettingsWorkspaceScopeKind string

const (
	SettingsWorkspaceScopeGlobal    SettingsWorkspaceScopeKind = "global"
	SettingsWorkspaceScopeWorkspace SettingsWorkspaceScopeKind = "workspace"
)

type SettingsSectionName string

const (
	SettingsSectionGeneral         SettingsSectionName = "general"
	SettingsSectionMemory          SettingsSectionName = "memory"
	SettingsSectionSkills          SettingsSectionName = "skills"
	SettingsSectionAutomation      SettingsSectionName = "automation"
	SettingsSectionNetwork         SettingsSectionName = "network"
	SettingsSectionObservability   SettingsSectionName = "observability"
	SettingsSectionHooksExtensions SettingsSectionName = "hooks-extensions"
)

type SettingsCollectionName string

const (
	SettingsCollectionProviders  SettingsCollectionName = "providers"
	SettingsCollectionMCPServers SettingsCollectionName = "mcp-servers"
	SettingsCollectionSandboxes  SettingsCollectionName = "sandboxes"
	SettingsCollectionHooks      SettingsCollectionName = "hooks"
)

type SettingsWriteTargetKind string

const (
	SettingsWriteTargetGlobalConfig        SettingsWriteTargetKind = "global-config"
	SettingsWriteTargetWorkspaceConfig     SettingsWriteTargetKind = "workspace-config"
	SettingsWriteTargetGlobalMCPSidecar    SettingsWriteTargetKind = "global-mcp-sidecar"
	SettingsWriteTargetWorkspaceMCPSidecar SettingsWriteTargetKind = "workspace-mcp-sidecar"
	SettingsWriteTargetGlobalAgentFile     SettingsWriteTargetKind = "global-agent-file"
	SettingsWriteTargetWorkspaceAgentFile  SettingsWriteTargetKind = "workspace-agent-file"
)

type SettingsTargetSelector string

const (
	SettingsTargetAuto    SettingsTargetSelector = "auto"
	SettingsTargetConfig  SettingsTargetSelector = "config"
	SettingsTargetSidecar SettingsTargetSelector = "sidecar"
)

type SettingsMutationBehavior string

const (
	SettingsMutationBehaviorAppliedNow      SettingsMutationBehavior = "applied_now"
	SettingsMutationBehaviorRestartRequired SettingsMutationBehavior = "restart_required"
	SettingsMutationBehaviorActionTrigger   SettingsMutationBehavior = "action_trigger"
)

type SettingsPermissionMode string

const (
	SettingsPermissionModeDenyAll      SettingsPermissionMode = "deny-all"
	SettingsPermissionModeApproveReads SettingsPermissionMode = "approve-reads"
	SettingsPermissionModeApproveAll   SettingsPermissionMode = "approve-all"
)

type SettingsSourceKind string

const (
	SettingsSourceBuiltinProvider     SettingsSourceKind = "builtin-provider"
	SettingsSourceGlobalConfig        SettingsSourceKind = "global-config"
	SettingsSourceWorkspaceConfig     SettingsSourceKind = "workspace-config"
	SettingsSourceGlobalMCPSidecar    SettingsSourceKind = "global-mcp-sidecar"
	SettingsSourceWorkspaceMCPSidecar SettingsSourceKind = "workspace-mcp-sidecar"
	SettingsSourceGlobalAgentFile     SettingsSourceKind = "global-agent-file"
	SettingsSourceWorkspaceAgentFile  SettingsSourceKind = "workspace-agent-file"
)

type RestartOperationStatus string

const (
	RestartOperationPending        RestartOperationStatus = "pending"
	RestartOperationStopping       RestartOperationStatus = "stopping"
	RestartOperationWaitingRelease RestartOperationStatus = "waiting_release"
	RestartOperationStarting       RestartOperationStatus = "starting"
	RestartOperationReady          RestartOperationStatus = "ready"
	RestartOperationFailed         RestartOperationStatus = "failed"
)

type SettingsStreamTransport string

const (
	SettingsStreamTransportSSE SettingsStreamTransport = "sse"
)

type SettingsUpdateStatusKind string

const (
	SettingsUpdateStatusCurrent     SettingsUpdateStatusKind = "current"
	SettingsUpdateStatusAvailable   SettingsUpdateStatusKind = "available"
	SettingsUpdateStatusUpdated     SettingsUpdateStatusKind = "updated"
	SettingsUpdateStatusDeferred    SettingsUpdateStatusKind = "deferred"
	SettingsUpdateStatusUnsupported SettingsUpdateStatusKind = "unsupported"
	SettingsUpdateStatusFailed      SettingsUpdateStatusKind = "failed"
)

type SettingsGlobalSectionResponseMetaPayload struct {
	Section         SettingsSectionName       `json:"section"`
	Scope           SettingsGlobalScopeKind   `json:"scope"`
	AvailableScopes []SettingsGlobalScopeKind `json:"available_scopes"`
}

type SettingsSkillsSectionResponseMetaPayload struct {
	Section         SettingsSectionName      `json:"section"`
	Scope           SettingsAgentScopeKind   `json:"scope"`
	WorkspaceID     string                   `json:"workspace_id,omitempty"`
	AgentName       string                   `json:"agent_name,omitempty"`
	AvailableScopes []SettingsAgentScopeKind `json:"available_scopes"`
}

type SettingsGlobalCollectionResponseMetaPayload struct {
	Collection      SettingsCollectionName    `json:"collection"`
	Scope           SettingsGlobalScopeKind   `json:"scope"`
	AvailableScopes []SettingsGlobalScopeKind `json:"available_scopes"`
}

type SettingsGlobalWorkspaceCollectionResponseMetaPayload struct {
	Collection      SettingsCollectionName       `json:"collection"`
	Scope           SettingsWorkspaceScopeKind   `json:"scope"`
	WorkspaceID     string                       `json:"workspace_id,omitempty"`
	AvailableScopes []SettingsWorkspaceScopeKind `json:"available_scopes"`
}

type SettingsGeneralConfigPayload struct {
	Defaults       SettingsDefaultsPayload    `json:"defaults"`
	Limits         SettingsLimitsPayload      `json:"limits"`
	Permissions    SettingsPermissionsPayload `json:"permissions"`
	SessionTimeout string                     `json:"session_timeout"`
	HTTP           SettingsHTTPPayload        `json:"http"`
	Daemon         SettingsDaemonPayload      `json:"daemon"`
}

type SettingsDefaultsPayload struct {
	Agent    string `json:"agent"`
	Provider string `json:"provider,omitempty"`
	Sandbox  string `json:"sandbox,omitempty"`
}

type SettingsLimitsPayload struct {
	MaxSessions         int `json:"max_sessions"`
	MaxConcurrentAgents int `json:"max_concurrent_agents"`
}

type SettingsPermissionsPayload struct {
	Mode SettingsPermissionMode `json:"mode"`
}

type SettingsHTTPPayload struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type SettingsDaemonPayload struct {
	Socket string `json:"socket"`
}

type SettingsMemoryConfigPayload struct {
	Enabled    bool                            `json:"enabled"`
	GlobalDir  string                          `json:"global_dir,omitempty"`
	Controller SettingsMemoryControllerPayload `json:"controller"`
	Recall     SettingsMemoryRecallPayload     `json:"recall"`
	Decisions  SettingsMemoryDecisionsPayload  `json:"decisions"`
	Extractor  SettingsMemoryExtractorPayload  `json:"extractor"`
	Dream      SettingsMemoryDreamPayload      `json:"dream"`
	Session    SettingsMemorySessionPayload    `json:"session"`
	Daily      SettingsMemoryDailyPayload      `json:"daily"`
	File       SettingsMemoryFilePayload       `json:"file"`
	Provider   SettingsMemoryProviderPayload   `json:"provider"`
	Workspace  SettingsMemoryWorkspacePayload  `json:"workspace"`
}

type SettingsMemoryControllerPayload struct {
	Mode            string                                `json:"mode"`
	MaxLatency      string                                `json:"max_latency"`
	DefaultOpOnFail string                                `json:"default_op_on_fail"`
	LLM             SettingsMemoryControllerLLMPayload    `json:"llm"`
	Policy          SettingsMemoryControllerPolicyPayload `json:"policy"`
}

type SettingsMemoryControllerLLMPayload struct {
	Enabled       bool   `json:"enabled"`
	Model         string `json:"model"`
	TopK          int    `json:"top_k"`
	PromptVersion string `json:"prompt_version"`
	Timeout       string `json:"timeout"`
	MaxTokensOut  int    `json:"max_tokens_out"`
}

type SettingsMemoryControllerPolicyPayload struct {
	MaxContentChars int      `json:"max_content_chars"`
	MaxWritesPerMin int      `json:"max_writes_per_min"`
	AllowOrigins    []string `json:"allow_origins"`
}

type SettingsMemoryRecallPayload struct {
	TopK                   int                                  `json:"top_k"`
	RawCandidates          int                                  `json:"raw_candidates"`
	Fusion                 string                               `json:"fusion"`
	IncludeAlreadySurfaced bool                                 `json:"include_already_surfaced"`
	IncludeSystem          bool                                 `json:"include_system"`
	Weights                SettingsMemoryRecallWeightsPayload   `json:"weights"`
	Freshness              SettingsMemoryRecallFreshnessPayload `json:"freshness"`
	Signals                SettingsMemoryRecallSignalsPayload   `json:"signals"`
}

type SettingsMemoryRecallWeightsPayload struct {
	BM25Unicode  float64 `json:"bm25_unicode"`
	BM25Trigram  float64 `json:"bm25_trigram"`
	Recency      float64 `json:"recency"`
	RecallSignal float64 `json:"recall_signal"`
}

type SettingsMemoryRecallFreshnessPayload struct {
	BannerAfterDays int `json:"banner_after_days"`
}

type SettingsMemoryRecallSignalsPayload struct {
	QueueCapacity  int  `json:"queue_capacity"`
	WorkerRetryMax int  `json:"worker_retry_max"`
	MetricsEnabled bool `json:"metrics_enabled"`
}

type SettingsMemoryDecisionsPayload struct {
	PruneAfterAppliedDays int   `json:"prune_after_applied_days"`
	KeepAuditSummary      bool  `json:"keep_audit_summary"`
	MaxPostContentBytes   int64 `json:"max_post_content_bytes"`
}

type SettingsMemoryExtractorPayload struct {
	Enabled          bool                                `json:"enabled"`
	Mode             string                              `json:"mode"`
	ThrottleTurns    int                                 `json:"throttle_turns"`
	Deadline         string                              `json:"deadline"`
	SandboxInboxOnly bool                                `json:"sandbox_inbox_only"`
	InboxPath        string                              `json:"inbox_path"`
	DLQPath          string                              `json:"dlq_path"`
	Model            string                              `json:"model"`
	Queue            SettingsMemoryExtractorQueuePayload `json:"queue"`
}

type SettingsMemoryExtractorQueuePayload struct {
	Capacity    int `json:"capacity"`
	CoalesceMax int `json:"coalesce_max"`
}

type SettingsMemoryDreamPayload struct {
	Enabled       bool                              `json:"enabled"`
	Agent         string                            `json:"agent"`
	MinHours      float64                           `json:"min_hours"`
	MinSessions   int                               `json:"min_sessions"`
	Debounce      string                            `json:"debounce"`
	PromptVersion string                            `json:"prompt_version"`
	CheckInterval string                            `json:"check_interval"`
	Gates         SettingsMemoryDreamGatesPayload   `json:"gates"`
	Scoring       SettingsMemoryDreamScoringPayload `json:"scoring"`
}

type SettingsMemoryDreamGatesPayload struct {
	MinUnpromoted  int     `json:"min_unpromoted"`
	MinRecallCount int     `json:"min_recall_count"`
	MinScore       float64 `json:"min_score"`
}

type SettingsMemoryDreamScoringPayload struct {
	RecencyHalfLifeDays int                                      `json:"recency_half_life_days"`
	Weights             SettingsMemoryDreamScoringWeightsPayload `json:"weights"`
}

type SettingsMemoryDreamScoringWeightsPayload struct {
	Frequency float64 `json:"frequency"`
	Relevance float64 `json:"relevance"`
	Recency   float64 `json:"recency"`
	Freshness float64 `json:"freshness"`
}

type SettingsMemorySessionPayload struct {
	LedgerFormat     string `json:"ledger_format"`
	LedgerRoot       string `json:"ledger_root"`
	EventsPurgeGrace string `json:"events_purge_grace"`
	ColdArchiveDays  int    `json:"cold_archive_days"`
	HardDeleteDays   int    `json:"hard_delete_days"`
	MaxArchiveBytes  int64  `json:"max_archive_bytes"`
	UnboundPartition string `json:"unbound_partition"`
}

type SettingsMemoryDailyPayload struct {
	MaxBytes        int64  `json:"max_bytes"`
	MaxLines        int    `json:"max_lines"`
	RotateFormat    string `json:"rotate_format"`
	DreamingWindow  int    `json:"dreaming_window"`
	ColdArchiveDays int    `json:"cold_archive_days"`
	HardDeleteDays  int    `json:"hard_delete_days"`
	MaxArchiveBytes int64  `json:"max_archive_bytes"`
	SweepHour       int    `json:"sweep_hour"`
	ArchivePath     string `json:"archive_path"`
}

type SettingsMemoryFilePayload struct {
	MaxLines int   `json:"max_lines"`
	MaxBytes int64 `json:"max_bytes"`
}

type SettingsMemoryProviderPayload struct {
	Name             string `json:"name"`
	Timeout          string `json:"timeout"`
	FailureThreshold int    `json:"failure_threshold"`
	Cooldown         string `json:"cooldown"`
}

type SettingsMemoryWorkspacePayload struct {
	TOMLPath   string `json:"toml_path"`
	AutoCreate bool   `json:"auto_create"`
}

type SettingsMarketplacePayload struct {
	Registry string `json:"registry"`
	BaseURL  string `json:"base_url,omitempty"`
}

type SettingsSkillsConfigPayload struct {
	Enabled                 bool                       `json:"enabled"`
	DisabledSkills          []string                   `json:"disabled_skills,omitempty"`
	PollInterval            string                     `json:"poll_interval"`
	AllowedMarketplaceMCP   []string                   `json:"allowed_marketplace_mcp,omitempty"`
	AllowedMarketplaceHooks []string                   `json:"allowed_marketplace_hooks,omitempty"`
	Marketplace             SettingsMarketplacePayload `json:"marketplace"`
}

type SettingsAutomationConfigPayload struct {
	Enabled           bool                            `json:"enabled"`
	Timezone          string                          `json:"timezone"`
	MaxConcurrentJobs int                             `json:"max_concurrent_jobs"`
	DefaultFireLimit  automationmodel.FireLimitConfig `json:"default_fire_limit"`
}

type SettingsNetworkConfigPayload struct {
	Enabled        bool   `json:"enabled"`
	DefaultChannel string `json:"default_channel"`
	Port           int    `json:"port"`
	MaxPayload     int    `json:"max_payload"`
	GreetInterval  int    `json:"greet_interval"`
	MaxReplayAge   int    `json:"max_replay_age"`
	MaxQueueDepth  int    `json:"max_queue_depth"`
}

type SettingsObservabilityConfigPayload struct {
	Enabled        bool                                   `json:"enabled"`
	RetentionDays  int                                    `json:"retention_days"`
	MaxGlobalBytes int64                                  `json:"max_global_bytes"`
	Transcripts    SettingsObservabilityTranscriptPayload `json:"transcripts"`
}

type SettingsObservabilityTranscriptPayload struct {
	Enabled            bool  `json:"enabled"`
	SegmentBytes       int   `json:"segment_bytes"`
	MaxBytesPerSession int64 `json:"max_bytes_per_session"`
}

type SettingsExtensionsConfigPayload struct {
	Marketplace SettingsMarketplacePayload        `json:"marketplace"`
	Resources   SettingsExtensionResourcesPayload `json:"resources"`
}

type SettingsExtensionResourcesPayload struct {
	AllowedKinds           []string                          `json:"allowed_kinds,omitempty"`
	MaxScope               resources.ResourceScopeKind       `json:"max_scope,omitempty"`
	SnapshotRateLimit      SettingsExtensionRateLimitPayload `json:"snapshot_rate_limit"`
	OperatorWriteRateLimit SettingsExtensionRateLimitPayload `json:"operator_write_rate_limit"`
}

type SettingsExtensionRateLimitPayload struct {
	Requests int    `json:"requests"`
	Window   string `json:"window"`
	Queue    int    `json:"queue"`
}

type SettingsConfigPathsPayload struct {
	HomeDir          string `json:"home_dir"`
	GlobalConfig     string `json:"global_config"`
	GlobalMCPSidecar string `json:"global_mcp_sidecar"`
	LogFile          string `json:"log_file"`
	DaemonInfo       string `json:"daemon_info"`
}

type SettingsDaemonRuntimePayload struct {
	Available      bool       `json:"available"`
	Status         string     `json:"status,omitempty"`
	PID            int        `json:"pid,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	UptimeSeconds  int64      `json:"uptime_seconds"`
	Socket         string     `json:"socket,omitempty"`
	HTTPHost       string     `json:"http_host,omitempty"`
	HTTPPort       int        `json:"http_port,omitempty"`
	ActiveSessions int        `json:"active_sessions"`
	ActiveAgents   int        `json:"active_agents"`
	TotalSessions  int        `json:"total_sessions"`
	Version        string     `json:"version,omitempty"`
}

type SettingsMemoryHealthPayload struct {
	Available          bool       `json:"available"`
	FileCount          int        `json:"file_count"`
	DreamEnabled       bool       `json:"dream_enabled"`
	LastConsolidatedAt *time.Time `json:"last_consolidated_at,omitempty"`
}

type SettingsAutomationRuntimePayload struct {
	Available        bool       `json:"available"`
	Running          bool       `json:"running"`
	SchedulerRunning bool       `json:"scheduler_running"`
	JobTotal         int        `json:"job_total"`
	JobEnabled       int        `json:"job_enabled"`
	TriggerTotal     int        `json:"trigger_total"`
	TriggerEnabled   int        `json:"trigger_enabled"`
	NextFire         *time.Time `json:"next_fire,omitempty"`
	LastSyncedAt     *time.Time `json:"last_synced_at,omitempty"`
}

type SettingsNetworkRuntimePayload struct {
	Available       bool   `json:"available"`
	Enabled         bool   `json:"enabled"`
	Status          string `json:"status,omitempty"`
	ListenerHost    string `json:"listener_host,omitempty"`
	ListenerPort    int    `json:"listener_port,omitempty"`
	LocalPeers      int    `json:"local_peers"`
	RemotePeers     int    `json:"remote_peers"`
	Channels        int    `json:"channels"`
	QueuedMessages  int    `json:"queued_messages"`
	QueuedSessions  int    `json:"queued_sessions"`
	DeliveryWorkers int    `json:"delivery_workers"`
}

type SettingsObservabilityRuntimePayload struct {
	Available          bool   `json:"available"`
	Status             string `json:"status,omitempty"`
	GlobalDBSizeBytes  int64  `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64  `json:"session_db_size_bytes"`
	ActiveSessions     int    `json:"active_sessions"`
	ActiveAgents       int    `json:"active_agents"`
	UptimeSeconds      int64  `json:"uptime_seconds"`
}

type SettingsLogTailCapabilityPayload struct {
	Available bool                    `json:"available"`
	StreamURL string                  `json:"stream_url,omitempty"`
	Transport SettingsStreamTransport `json:"transport,omitempty"`
}

type SettingsActionMetadataPayload struct {
	Name      string                   `json:"name"`
	Available bool                     `json:"available"`
	Behavior  SettingsMutationBehavior `json:"behavior"`
}

type SettingsGeneralActionsPayload struct {
	Restart SettingsActionMetadataPayload `json:"restart"`
}

type SettingsMemoryActionsPayload struct {
	Consolidate SettingsActionMetadataPayload `json:"consolidate"`
}

type SettingsOperationalLinkPayload struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type SettingsTransportParityPayload struct {
	Known          bool `json:"known"`
	SettingsHTTP   bool `json:"settings_http"`
	SettingsUDS    bool `json:"settings_uds"`
	ExtensionsHTTP bool `json:"extensions_http"`
	ExtensionsUDS  bool `json:"extensions_uds"`
}

type SettingsInstalledExtensionPayload struct {
	Name          string   `json:"name"`
	Version       string   `json:"version,omitempty"`
	Enabled       bool     `json:"enabled"`
	State         string   `json:"state,omitempty"`
	Health        string   `json:"health,omitempty"`
	HealthMessage string   `json:"health_message,omitempty"`
	LastError     string   `json:"last_error,omitempty"`
	RequiresEnv   []string `json:"requires_env,omitempty"`
	MissingEnv    []string `json:"missing_env,omitempty"`
}

type SettingsSourceRefPayload struct {
	Kind        SettingsSourceKind `json:"kind"`
	Scope       SettingsScopeKind  `json:"scope"`
	WorkspaceID string             `json:"workspace_id,omitempty"`
	AgentName   string             `json:"agent_name,omitempty"`
}

type SettingsSourceMetadataPayload struct {
	EffectiveSource  SettingsSourceRefPayload   `json:"effective_source"`
	ShadowedSources  []SettingsSourceRefPayload `json:"shadowed_sources,omitempty"`
	AvailableTargets []SettingsWriteTargetKind  `json:"available_targets"`
}

type SettingsProviderSettingsPayload struct {
	Command         string                                  `json:"command,omitempty"`
	DisplayName     string                                  `json:"display_name,omitempty"`
	Models          *SettingsProviderModelsPayload          `json:"models,omitempty"`
	Harness         string                                  `json:"harness,omitempty"`
	RuntimeProvider string                                  `json:"runtime_provider,omitempty"`
	Transport       string                                  `json:"transport,omitempty"`
	BaseURL         string                                  `json:"base_url,omitempty"`
	AuthMode        string                                  `json:"auth_mode,omitempty"`
	EnvPolicy       string                                  `json:"env_policy,omitempty"`
	HomePolicy      string                                  `json:"home_policy,omitempty"`
	AuthStatusCmd   string                                  `json:"auth_status_command,omitempty"`
	AuthLoginCmd    string                                  `json:"auth_login_command,omitempty"`
	CredentialSlots []SettingsProviderCredentialSlotPayload `json:"credential_slots,omitempty"`
}

type SettingsProviderModelsPayload struct {
	Default   string                                  `json:"default,omitempty"`
	Curated   []SettingsProviderModelPayload          `json:"curated,omitempty"`
	Discovery *SettingsProviderModelsDiscoveryPayload `json:"discovery,omitempty"`
}

type SettingsProviderModelsDiscoveryPayload struct {
	Enabled  *bool  `json:"enabled,omitempty"`
	Command  string `json:"command,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
}

type SettingsProviderModelPayload struct {
	ID                     string   `json:"id"`
	DisplayName            string   `json:"display_name,omitempty"`
	ContextWindow          *int64   `json:"context_window,omitempty"`
	MaxInputTokens         *int64   `json:"max_input_tokens,omitempty"`
	MaxOutputTokens        *int64   `json:"max_output_tokens,omitempty"`
	SupportsTools          *bool    `json:"supports_tools,omitempty"`
	SupportsReasoning      *bool    `json:"supports_reasoning,omitempty"`
	ReasoningEfforts       []string `json:"reasoning_efforts,omitempty"`
	DefaultReasoningEffort string   `json:"default_reasoning_effort,omitempty"`
	CostInputPerMillion    *float64 `json:"cost_input_per_million,omitempty"`
	CostOutputPerMillion   *float64 `json:"cost_output_per_million,omitempty"`
}

type SettingsProviderCredentialSlotPayload struct {
	Name      string `json:"name"`
	TargetEnv string `json:"target_env"`
	SecretRef string `json:"secret_ref"`
	Kind      string `json:"kind,omitempty"`
	Required  bool   `json:"required"`
}

type SettingsProviderCredentialStatusPayload struct {
	Name      string `json:"name"`
	TargetEnv string `json:"target_env"`
	SecretRef string `json:"secret_ref"`
	Kind      string `json:"kind,omitempty"`
	Required  bool   `json:"required"`
	Present   bool   `json:"present"`
	Source    string `json:"source,omitempty"`
}

type SettingsProviderAuthStatusPayload struct {
	Mode       string `json:"mode"`
	EnvPolicy  string `json:"env_policy"`
	HomePolicy string `json:"home_policy"`
	State      string `json:"state"`
	Message    string `json:"message,omitempty"`
	StatusCmd  string `json:"status_command,omitempty"`
	LoginCmd   string `json:"login_command,omitempty"`
}

type SettingsProviderSecretWritePayload struct {
	Name      string `json:"name,omitempty"`
	SecretRef string `json:"secret_ref"`
	Kind      string `json:"kind,omitempty"`
	Value     string `json:"value"`
}

type SettingsProviderFallbackPayload struct {
	Source   SettingsSourceRefPayload        `json:"source"`
	Settings SettingsProviderSettingsPayload `json:"settings"`
}

type SettingsProviderItemPayload struct {
	Name             string                                    `json:"name"`
	Settings         SettingsProviderSettingsPayload           `json:"settings"`
	Default          bool                                      `json:"default"`
	CommandAvailable bool                                      `json:"command_available"`
	Credentials      []SettingsProviderCredentialStatusPayload `json:"credentials,omitempty"`
	AuthStatus       *SettingsProviderAuthStatusPayload        `json:"auth_status,omitempty"`
	SourceMetadata   SettingsSourceMetadataPayload             `json:"source_metadata"`
	Fallback         *SettingsProviderFallbackPayload          `json:"fallback,omitempty"`
}

type SettingsMCPServerPayload struct {
	Name      string                        `json:"name"`
	Transport string                        `json:"transport,omitempty"`
	Command   string                        `json:"command,omitempty"`
	Args      []string                      `json:"args,omitempty"`
	Env       map[string]string             `json:"env,omitempty"`
	SecretEnv map[string]string             `json:"secret_env,omitempty"`
	URL       string                        `json:"url,omitempty"`
	Auth      *SettingsMCPAuthConfigPayload `json:"auth,omitempty"`
}

type SettingsMCPAuthConfigPayload struct {
	Type             string   `json:"type,omitempty"`
	IssuerURL        string   `json:"issuer_url,omitempty"`
	MetadataURL      string   `json:"metadata_url,omitempty"`
	AuthorizationURL string   `json:"authorization_url,omitempty"`
	TokenURL         string   `json:"token_url,omitempty"`
	RevocationURL    string   `json:"revocation_url,omitempty"`
	ClientID         string   `json:"client_id,omitempty"`
	ClientSecretRef  string   `json:"client_secret_ref,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
}

type SettingsMCPSecretValuesPayload struct {
	SecretEnv         map[string]string `json:"secret_env,omitempty"`
	OAuthClientSecret *string           `json:"oauth_client_secret,omitempty"`
}

type SettingsMCPAuthStatusPayload struct {
	ServerName       string     `json:"server_name"`
	Status           string     `json:"status"`
	RemoteURL        string     `json:"remote_url,omitempty"`
	AuthType         string     `json:"auth_type,omitempty"`
	ClientID         string     `json:"client_id,omitempty"`
	Issuer           string     `json:"issuer,omitempty"`
	Scopes           []string   `json:"scopes,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	Refreshable      bool       `json:"refreshable"`
	TokenPresent     bool       `json:"token_present"`
	RevocationURL    string     `json:"revocation_url,omitempty"`
	Diagnostic       string     `json:"diagnostic,omitempty"`
	AuthorizationURL string     `json:"authorization_url,omitempty"`
}

type SettingsMCPServerItemPayload struct {
	Name           string                        `json:"name"`
	Transport      string                        `json:"transport"`
	Command        string                        `json:"command,omitempty"`
	Args           []string                      `json:"args,omitempty"`
	Env            map[string]string             `json:"env,omitempty"`
	SecretEnv      map[string]string             `json:"secret_env,omitempty"`
	URL            string                        `json:"url,omitempty"`
	Auth           *SettingsMCPAuthConfigPayload `json:"auth,omitempty"`
	AuthStatus     *SettingsMCPAuthStatusPayload `json:"auth_status,omitempty"`
	Scope          SettingsScopeKind             `json:"scope"`
	WorkspaceID    string                        `json:"workspace_id,omitempty"`
	SourceMetadata SettingsSourceMetadataPayload `json:"source_metadata"`
}

type SettingsSandboxProfilePayload struct {
	Backend     string                         `json:"backend"`
	SyncMode    string                         `json:"sync_mode,omitempty"`
	Persistence string                         `json:"persistence,omitempty"`
	RuntimeRoot string                         `json:"runtime_root,omitempty"`
	Env         map[string]string              `json:"env,omitempty"`
	SecretEnv   map[string]string              `json:"secret_env,omitempty"`
	Network     *SettingsSandboxNetworkPayload `json:"network,omitempty"`
	Daytona     *SettingsSandboxDaytonaPayload `json:"daytona,omitempty"`
}

type SettingsSandboxNetworkPayload struct {
	AllowPublicIngress bool     `json:"allow_public_ingress,omitempty"`
	AllowOutbound      bool     `json:"allow_outbound,omitempty"`
	AllowList          []string `json:"allow_list,omitempty"`
	DenyList           []string `json:"deny_list,omitempty"`
	Required           bool     `json:"required,omitempty"`
}

type SettingsSandboxDaytonaPayload struct {
	APIURL      string `json:"api_url,omitempty"`
	Target      string `json:"target,omitempty"`
	Image       string `json:"image,omitempty"`
	Snapshot    string `json:"snapshot,omitempty"`
	Class       string `json:"class,omitempty"`
	AutoStop    string `json:"auto_stop,omitempty"`
	AutoArchive string `json:"auto_archive,omitempty"`
}

type SettingsSandboxItemPayload struct {
	Name                string                        `json:"name"`
	Profile             SettingsSandboxProfilePayload `json:"profile"`
	WorkspaceUsageCount int                           `json:"workspace_usage_count"`
	SourceMetadata      SettingsSourceMetadataPayload `json:"source_metadata"`
}

type SettingsHookDeclarationPayload struct {
	Name         string                    `json:"name"`
	Event        hookspkg.HookEvent        `json:"event"`
	Mode         hookspkg.HookMode         `json:"mode,omitempty"`
	Required     bool                      `json:"required,omitempty"`
	Priority     int                       `json:"priority,omitempty"`
	Timeout      string                    `json:"timeout,omitempty"`
	Matcher      hookspkg.HookMatcher      `json:"matcher"`
	ExecutorKind hookspkg.HookExecutorKind `json:"executor_kind,omitempty"`
	Command      string                    `json:"command,omitempty"`
	Args         []string                  `json:"args,omitempty"`
	Env          map[string]string         `json:"env,omitempty"`
	SecretEnv    map[string]string         `json:"secret_env,omitempty"`
	Metadata     map[string]string         `json:"metadata,omitempty"`
}

type SettingsHookItemPayload struct {
	Name           string                         `json:"name"`
	Declaration    SettingsHookDeclarationPayload `json:"declaration"`
	SourceMetadata SettingsSourceMetadataPayload  `json:"source_metadata"`
}

type UpdateSettingsGeneralRequest struct {
	Config SettingsGeneralConfigPayload `json:"config"`
}

type UpdateSettingsMemoryRequest struct {
	Config SettingsMemoryConfigPayload `json:"config"`
}

type UpdateSettingsSkillsRequest struct {
	Config SettingsSkillsConfigPayload `json:"config"`
}

type UpdateSettingsAutomationRequest struct {
	Config SettingsAutomationConfigPayload `json:"config"`
}

type UpdateSettingsNetworkRequest struct {
	Config SettingsNetworkConfigPayload `json:"config"`
}

type UpdateSettingsObservabilityRequest struct {
	Config SettingsObservabilityConfigPayload `json:"config"`
}

type UpdateSettingsHooksExtensionsRequest struct {
	Config SettingsExtensionsConfigPayload `json:"config"`
}

type PutSettingsProviderRequest struct {
	Settings SettingsProviderSettingsPayload      `json:"settings"`
	Secrets  []SettingsProviderSecretWritePayload `json:"secrets,omitempty"`
}

type PutSettingsMCPServerRequest struct {
	Server       SettingsMCPServerPayload        `json:"server"`
	SecretValues *SettingsMCPSecretValuesPayload `json:"secret_values,omitempty"`
}

type PutSettingsSandboxRequest struct {
	Profile SettingsSandboxProfilePayload `json:"profile"`
}

type PutSettingsHookRequest struct {
	Declaration SettingsHookDeclarationPayload `json:"declaration"`
}

type SettingsGeneralResponse struct {
	SettingsGlobalSectionResponseMetaPayload
	ConfigPaths SettingsConfigPathsPayload    `json:"config_paths"`
	Config      SettingsGeneralConfigPayload  `json:"config"`
	Runtime     SettingsDaemonRuntimePayload  `json:"runtime"`
	Actions     SettingsGeneralActionsPayload `json:"actions"`
}

type SettingsMemoryResponse struct {
	SettingsGlobalSectionResponseMetaPayload
	Config  SettingsMemoryConfigPayload  `json:"config"`
	Health  SettingsMemoryHealthPayload  `json:"health"`
	Actions SettingsMemoryActionsPayload `json:"actions"`
}

type SettingsSkillsResponse struct {
	SettingsSkillsSectionResponseMetaPayload
	Config           SettingsSkillsConfigPayload      `json:"config"`
	DiscoveredCount  int                              `json:"discovered_count"`
	DisabledCount    int                              `json:"disabled_count"`
	RuntimeAvailable bool                             `json:"runtime_available"`
	Links            []SettingsOperationalLinkPayload `json:"links,omitempty"`
}

type SettingsAutomationResponse struct {
	SettingsGlobalSectionResponseMetaPayload
	Config  SettingsAutomationConfigPayload  `json:"config"`
	Runtime SettingsAutomationRuntimePayload `json:"runtime"`
	Links   []SettingsOperationalLinkPayload `json:"links,omitempty"`
}

type SettingsNetworkResponse struct {
	SettingsGlobalSectionResponseMetaPayload
	Config  SettingsNetworkConfigPayload     `json:"config"`
	Runtime SettingsNetworkRuntimePayload    `json:"runtime"`
	Links   []SettingsOperationalLinkPayload `json:"links,omitempty"`
}

type SettingsObservabilityResponse struct {
	SettingsGlobalSectionResponseMetaPayload
	Config  SettingsObservabilityConfigPayload  `json:"config"`
	Runtime SettingsObservabilityRuntimePayload `json:"runtime"`
	LogTail SettingsLogTailCapabilityPayload    `json:"log_tail"`
}

type SettingsHooksExtensionsResponse struct {
	SettingsGlobalSectionResponseMetaPayload
	Hooks           []SettingsHookItemPayload           `json:"hooks,omitempty"`
	Config          SettingsExtensionsConfigPayload     `json:"config"`
	Installed       []SettingsInstalledExtensionPayload `json:"installed,omitempty"`
	TransportParity SettingsTransportParityPayload      `json:"transport_parity"`
}

type SettingsProvidersResponse struct {
	SettingsGlobalCollectionResponseMetaPayload
	Providers []SettingsProviderItemPayload `json:"providers"`
}

type SettingsProviderResponse struct {
	Provider SettingsProviderItemPayload `json:"provider"`
}

type SettingsMCPServersResponse struct {
	SettingsGlobalWorkspaceCollectionResponseMetaPayload
	MCPServers []SettingsMCPServerItemPayload `json:"mcp_servers"`
}

type SettingsSandboxesResponse struct {
	SettingsGlobalCollectionResponseMetaPayload
	Sandboxes []SettingsSandboxItemPayload `json:"sandboxes"`
}

type SettingsSandboxResponse struct {
	Sandbox SettingsSandboxItemPayload `json:"sandbox"`
}

type SettingsHooksResponse struct {
	SettingsGlobalCollectionResponseMetaPayload
	Hooks []SettingsHookItemPayload `json:"hooks"`
}

type SettingsHookResponse struct {
	Hook SettingsHookItemPayload `json:"hook"`
}

type SettingsGlobalSectionMutationResult struct {
	Section         SettingsSectionName      `json:"section"`
	Scope           SettingsGlobalScopeKind  `json:"scope"`
	WriteTarget     SettingsWriteTargetKind  `json:"write_target,omitempty"`
	Behavior        SettingsMutationBehavior `json:"behavior"`
	Applied         bool                     `json:"applied"`
	RestartRequired bool                     `json:"restart_required"`
	RestartScope    string                   `json:"restart_scope,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
}

type SettingsSkillsMutationResult struct {
	Section         SettingsSectionName      `json:"section"`
	Scope           SettingsAgentScopeKind   `json:"scope"`
	WriteTarget     SettingsWriteTargetKind  `json:"write_target,omitempty"`
	WorkspaceID     string                   `json:"workspace_id,omitempty"`
	AgentName       string                   `json:"agent_name,omitempty"`
	Behavior        SettingsMutationBehavior `json:"behavior"`
	Applied         bool                     `json:"applied"`
	RestartRequired bool                     `json:"restart_required"`
	RestartScope    string                   `json:"restart_scope,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
}

type SettingsGlobalCollectionMutationResult struct {
	Section         SettingsCollectionName   `json:"section"`
	Scope           SettingsGlobalScopeKind  `json:"scope"`
	WriteTarget     SettingsWriteTargetKind  `json:"write_target,omitempty"`
	Behavior        SettingsMutationBehavior `json:"behavior"`
	Applied         bool                     `json:"applied"`
	RestartRequired bool                     `json:"restart_required"`
	RestartScope    string                   `json:"restart_scope,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
}

type SettingsGlobalWorkspaceCollectionMutationResult struct {
	Section         SettingsCollectionName     `json:"section"`
	Scope           SettingsWorkspaceScopeKind `json:"scope"`
	WriteTarget     SettingsWriteTargetKind    `json:"write_target,omitempty"`
	WorkspaceID     string                     `json:"workspace_id,omitempty"`
	Behavior        SettingsMutationBehavior   `json:"behavior"`
	Applied         bool                       `json:"applied"`
	RestartRequired bool                       `json:"restart_required"`
	RestartScope    string                     `json:"restart_scope,omitempty"`
	Warnings        []string                   `json:"warnings,omitempty"`
}

type RestartActionResponse struct {
	OperationID        string                 `json:"operation_id"`
	Status             RestartOperationStatus `json:"status"`
	StatusURL          string                 `json:"status_url"`
	ActiveSessionCount int                    `json:"active_session_count"`
}

type RestartActionStatus struct {
	OperationID        string                 `json:"operation_id"`
	Status             RestartOperationStatus `json:"status"`
	OldPID             int                    `json:"old_pid"`
	OldStartedAt       time.Time              `json:"old_started_at"`
	OldSocketPath      string                 `json:"old_socket_path"`
	NewPID             int                    `json:"new_pid,omitempty"`
	ActiveSessionCount int                    `json:"active_session_count"`
	FailureReason      string                 `json:"failure_reason,omitempty"`
	StartedAt          time.Time              `json:"started_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
}

type SettingsUpdateResponse struct {
	Supported      bool                     `json:"supported"`
	Managed        bool                     `json:"managed"`
	InstallMethod  string                   `json:"install_method"`
	CurrentVersion string                   `json:"current_version"`
	LatestVersion  string                   `json:"latest_version,omitempty"`
	Available      bool                     `json:"available"`
	Status         SettingsUpdateStatusKind `json:"status"`
	Recommendation string                   `json:"recommendation,omitempty"`
	ReleaseURL     string                   `json:"release_url,omitempty"`
	CheckedAt      *time.Time               `json:"checked_at,omitempty"`
	LastError      string                   `json:"last_error,omitempty"`
}
