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

type SettingsSectionResponseMetaPayload struct {
	Section         SettingsSectionName `json:"section"`
	Scope           SettingsScopeKind   `json:"scope"`
	WorkspaceID     string              `json:"workspace_id,omitempty"`
	AvailableScopes []SettingsScopeKind `json:"available_scopes"`
}

type SettingsCollectionResponseMetaPayload struct {
	Collection      SettingsCollectionName `json:"collection"`
	Scope           SettingsScopeKind      `json:"scope"`
	WorkspaceID     string                 `json:"workspace_id,omitempty"`
	AvailableScopes []SettingsScopeKind    `json:"available_scopes"`
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
	Enabled   bool                       `json:"enabled"`
	GlobalDir string                     `json:"global_dir,omitempty"`
	Dream     SettingsMemoryDreamPayload `json:"dream"`
}

type SettingsMemoryDreamPayload struct {
	Enabled       bool    `json:"enabled"`
	Agent         string  `json:"agent"`
	MinHours      float64 `json:"min_hours"`
	MinSessions   int     `json:"min_sessions"`
	CheckInterval string  `json:"check_interval"`
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
}

type SettingsSourceMetadataPayload struct {
	EffectiveSource  SettingsSourceRefPayload   `json:"effective_source"`
	ShadowedSources  []SettingsSourceRefPayload `json:"shadowed_sources,omitempty"`
	AvailableTargets []SettingsWriteTargetKind  `json:"available_targets"`
}

type SettingsProviderSettingsPayload struct {
	Command         string                                  `json:"command,omitempty"`
	DisplayName     string                                  `json:"display_name,omitempty"`
	DefaultModel    string                                  `json:"default_model,omitempty"`
	Harness         string                                  `json:"harness,omitempty"`
	RuntimeProvider string                                  `json:"runtime_provider,omitempty"`
	Transport       string                                  `json:"transport,omitempty"`
	BaseURL         string                                  `json:"base_url,omitempty"`
	CredentialSlots []SettingsProviderCredentialSlotPayload `json:"credential_slots,omitempty"`
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
	SettingsSectionResponseMetaPayload
	ConfigPaths SettingsConfigPathsPayload    `json:"config_paths"`
	Config      SettingsGeneralConfigPayload  `json:"config"`
	Runtime     SettingsDaemonRuntimePayload  `json:"runtime"`
	Actions     SettingsGeneralActionsPayload `json:"actions"`
}

type SettingsMemoryResponse struct {
	SettingsSectionResponseMetaPayload
	Config  SettingsMemoryConfigPayload  `json:"config"`
	Health  SettingsMemoryHealthPayload  `json:"health"`
	Actions SettingsMemoryActionsPayload `json:"actions"`
}

type SettingsSkillsResponse struct {
	SettingsSectionResponseMetaPayload
	Config           SettingsSkillsConfigPayload      `json:"config"`
	DiscoveredCount  int                              `json:"discovered_count"`
	DisabledCount    int                              `json:"disabled_count"`
	RuntimeAvailable bool                             `json:"runtime_available"`
	Links            []SettingsOperationalLinkPayload `json:"links,omitempty"`
}

type SettingsAutomationResponse struct {
	SettingsSectionResponseMetaPayload
	Config  SettingsAutomationConfigPayload  `json:"config"`
	Runtime SettingsAutomationRuntimePayload `json:"runtime"`
	Links   []SettingsOperationalLinkPayload `json:"links,omitempty"`
}

type SettingsNetworkResponse struct {
	SettingsSectionResponseMetaPayload
	Config  SettingsNetworkConfigPayload     `json:"config"`
	Runtime SettingsNetworkRuntimePayload    `json:"runtime"`
	Links   []SettingsOperationalLinkPayload `json:"links,omitempty"`
}

type SettingsObservabilityResponse struct {
	SettingsSectionResponseMetaPayload
	Config  SettingsObservabilityConfigPayload  `json:"config"`
	Runtime SettingsObservabilityRuntimePayload `json:"runtime"`
	LogTail SettingsLogTailCapabilityPayload    `json:"log_tail"`
}

type SettingsHooksExtensionsResponse struct {
	SettingsSectionResponseMetaPayload
	Hooks           []SettingsHookItemPayload           `json:"hooks,omitempty"`
	Config          SettingsExtensionsConfigPayload     `json:"config"`
	Installed       []SettingsInstalledExtensionPayload `json:"installed,omitempty"`
	TransportParity SettingsTransportParityPayload      `json:"transport_parity"`
}

type SettingsProvidersResponse struct {
	SettingsCollectionResponseMetaPayload
	Providers []SettingsProviderItemPayload `json:"providers"`
}

type SettingsProviderResponse struct {
	Provider SettingsProviderItemPayload `json:"provider"`
}

type SettingsMCPServersResponse struct {
	SettingsCollectionResponseMetaPayload
	MCPServers []SettingsMCPServerItemPayload `json:"mcp_servers"`
}

type SettingsSandboxesResponse struct {
	SettingsCollectionResponseMetaPayload
	Sandboxes []SettingsSandboxItemPayload `json:"sandboxes"`
}

type SettingsSandboxResponse struct {
	Sandbox SettingsSandboxItemPayload `json:"sandbox"`
}

type SettingsHooksResponse struct {
	SettingsCollectionResponseMetaPayload
	Hooks []SettingsHookItemPayload `json:"hooks"`
}

type SettingsHookResponse struct {
	Hook SettingsHookItemPayload `json:"hook"`
}

type MutationResult struct {
	Section         SettingsSectionName      `json:"section"`
	Scope           SettingsScopeKind        `json:"scope"`
	WriteTarget     SettingsWriteTargetKind  `json:"write_target,omitempty"`
	WorkspaceID     string                   `json:"workspace_id,omitempty"`
	Behavior        SettingsMutationBehavior `json:"behavior"`
	Applied         bool                     `json:"applied"`
	RestartRequired bool                     `json:"restart_required"`
	RestartScope    string                   `json:"restart_scope,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
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
