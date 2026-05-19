package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	burnttoml "github.com/BurntSushi/toml"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
)

const (
	mergeReadKey = "read"
)

const providersConfigKey = "providers"

type configOverlay struct {
	Daemon        daemonOverlay              `toml:"daemon"`
	HTTP          httpOverlay                `toml:"http"`
	Defaults      defaultsOverlay            `toml:"defaults"`
	Agents        agentsOverlay              `toml:"agents"`
	Limits        limitsOverlay              `toml:"limits"`
	Session       sessionOverlay             `toml:"session"`
	Permissions   permissionsOverlay         `toml:"permissions"`
	MCPServers    []mcpServerOverlay         `toml:"mcp_servers"`
	Providers     map[string]providerOverlay `toml:"providers"`
	ModelCatalog  modelCatalogOverlay        `toml:"model_catalog"`
	Sandboxes     map[string]sandboxOverlay  `toml:"sandboxes"`
	Observability observabilityOverlay       `toml:"observability"`
	Log           logOverlay                 `toml:"log"`
	Memory        memoryOverlay              `toml:"memory"`
	Skills        skillsOverlay              `toml:"skills"`
	Extensions    extensionsOverlay          `toml:"extensions"`
	Tools         toolsOverlay               `toml:"tools"`
	Automation    automationOverlay          `toml:"automation"`
	Task          taskOverlay                `toml:"task"`
	Hooks         hooksOverlay               `toml:"hooks"`
	Network       networkOverlay             `toml:"network"`
	Autonomy      autonomyOverlay            `toml:"autonomy"`
}

// FileError preserves the source file for configuration read/decode failures.
type FileError struct {
	Op   string
	Path string
	Err  error
}

func (e FileError) Error() string {
	return fmt.Sprintf("%s config file %q: %v", e.Op, e.Path, e.Err)
}

func (e FileError) Unwrap() error {
	return e.Err
}

type daemonOverlay struct {
	Socket *string `toml:"socket"`
}

type httpOverlay struct {
	Host *string `toml:"host"`
	Port *int    `toml:"port"`
}

type defaultsOverlay struct {
	Agent    *string `toml:"agent"`
	Provider *string `toml:"provider"`
	Sandbox  *string `toml:"sandbox"`
}

type agentsOverlay struct {
	Soul      soulOverlay      `toml:"soul"`
	Heartbeat heartbeatOverlay `toml:"heartbeat"`
}

type soulOverlay struct {
	Enabled                *bool  `toml:"enabled"`
	MaxBodyBytes           *int64 `toml:"max_body_bytes"`
	ContextProjectionBytes *int64 `toml:"context_projection_bytes"`
}

type heartbeatOverlay struct {
	Enabled                      *bool          `toml:"enabled"`
	MaxBodyBytes                 *int64         `toml:"max_body_bytes"`
	ContextProjectionBytes       *int64         `toml:"context_projection_bytes"`
	MinInterval                  *time.Duration `toml:"min_interval"`
	DefaultInterval              *time.Duration `toml:"default_interval"`
	WakeCooldown                 *time.Duration `toml:"wake_cooldown"`
	MaxWakesPerCycle             *int           `toml:"max_wakes_per_cycle"`
	ActiveSessionOnly            *bool          `toml:"active_session_only"`
	AllowActiveHoursPreferences  *bool          `toml:"allow_active_hours_preferences"`
	WakeEventRetention           *time.Duration `toml:"wake_event_retention"`
	SessionHealthStaleAfter      *time.Duration `toml:"session_health_stale_after"`
	SessionHealthHookMinInterval *time.Duration `toml:"session_health_hook_min_interval"`
}

type limitsOverlay struct {
	MaxConcurrentAgents *int `toml:"max_concurrent_agents"`
}

type sessionOverlay struct {
	Limits      sessionLimitsOverlay      `toml:"limits"`
	Supervision sessionSupervisionOverlay `toml:"supervision"`
}

type sessionLimitsOverlay struct {
	Timeout *time.Duration `toml:"timeout"`
}

type sessionSupervisionOverlay struct {
	ActivityHeartbeatInterval *time.Duration `toml:"activity_heartbeat_interval"`
	ProgressNotifyInterval    *time.Duration `toml:"progress_notify_interval"`
	PromptDeadline            *time.Duration `toml:"prompt_deadline"`
	InactivityWarningAfter    *time.Duration `toml:"inactivity_warning_after"`
	InactivityTimeout         *time.Duration `toml:"inactivity_timeout"`
	TimeoutCancelGrace        *time.Duration `toml:"timeout_cancel_grace"`
}

type permissionsOverlay struct {
	Mode *PermissionMode `toml:"mode"`
}

type providerOverlay struct {
	Command         *string                     `toml:"command"`
	DisplayName     *string                     `toml:"display_name"`
	Models          *providerModelsOverlay      `toml:"models"`
	Harness         *ProviderHarness            `toml:"harness"`
	RuntimeProvider *string                     `toml:"runtime_provider"`
	Transport       *string                     `toml:"transport"`
	BaseURL         *string                     `toml:"base_url"`
	AuthMode        *ProviderAuthMode           `toml:"auth_mode"`
	EnvPolicy       *ProviderEnvPolicy          `toml:"env_policy"`
	HomePolicy      *ProviderHomePolicy         `toml:"home_policy"`
	AuthStatusCmd   *string                     `toml:"auth_status_command"`
	AuthLoginCmd    *string                     `toml:"auth_login_command"`
	SessionMCP      *bool                       `toml:"session_mcp"`
	Aliases         *[]string                   `toml:"aliases"`
	CredentialSlots []providerCredentialOverlay `toml:"credential_slots"`
	MCPServers      []mcpServerOverlay          `toml:"mcp_servers"`
}

type providerModelsOverlay struct {
	Default   *string                        `toml:"default"`
	Curated   []ProviderModelConfig          `toml:"curated"`
	Discovery providerModelsDiscoveryOverlay `toml:"discovery"`
}

type providerModelsDiscoveryOverlay struct {
	Enabled  *bool   `toml:"enabled"`
	Command  *string `toml:"command"`
	Endpoint *string `toml:"endpoint"`
	Timeout  *string `toml:"timeout"`
}

type modelCatalogOverlay struct {
	Sources modelCatalogSourcesOverlay `toml:"sources"`
}

type modelCatalogSourcesOverlay struct {
	ModelsDev modelsDevSourceOverlay `toml:"models_dev"`
}

type modelsDevSourceOverlay struct {
	Enabled  *bool   `toml:"enabled"`
	Endpoint *string `toml:"endpoint"`
	TTL      *string `toml:"ttl"`
	Timeout  *string `toml:"timeout"`
}

type providerCredentialOverlay struct {
	Name      *string `toml:"name"`
	TargetEnv *string `toml:"target_env"`
	SecretRef *string `toml:"secret_ref"`
	Kind      *string `toml:"kind"`
	Required  *bool   `toml:"required"`
}

type sandboxOverlay struct {
	Backend     *string               `toml:"backend"`
	SyncMode    *string               `toml:"sync_mode"`
	Persistence *string               `toml:"persistence"`
	RuntimeRoot *string               `toml:"runtime_root"`
	Env         *map[string]string    `toml:"env"`
	SecretEnv   *map[string]string    `toml:"secret_env"`
	Network     networkProfileOverlay `toml:"network"`
	Daytona     daytonaProfileOverlay `toml:"daytona"`
}

type networkProfileOverlay struct {
	AllowPublicIngress *bool     `toml:"allow_public_ingress"`
	AllowOutbound      *bool     `toml:"allow_outbound"`
	AllowList          *[]string `toml:"allow_list"`
	DenyList           *[]string `toml:"deny_list"`
	Required           *bool     `toml:"required"`
}

type daytonaProfileOverlay struct {
	APIURL      *string `toml:"api_url"`
	Target      *string `toml:"target"`
	Image       *string `toml:"image"`
	Snapshot    *string `toml:"snapshot"`
	Class       *string `toml:"class"`
	AutoStop    *string `toml:"auto_stop"`
	AutoArchive *string `toml:"auto_archive"`
}

type observabilityOverlay struct {
	Enabled           *bool                           `toml:"enabled"`
	RetentionDays     *int                            `toml:"retention_days"`
	MaxGlobalBytes    *int64                          `toml:"max_global_bytes"`
	AgentProbeTimeout *time.Duration                  `toml:"agent_probe_timeout"`
	Transcripts       observabilityTranscriptsOverlay `toml:"transcripts"`
}

type observabilityTranscriptsOverlay struct {
	Enabled            *bool  `toml:"enabled"`
	SegmentBytes       *int   `toml:"segment_bytes"`
	MaxBytesPerSession *int64 `toml:"max_bytes_per_session"`
}

type logOverlay struct {
	Level *string `toml:"level"`
}

type memoryOverlay struct {
	Enabled    *bool                   `toml:"enabled"`
	GlobalDir  *string                 `toml:"global_dir"`
	Controller memoryControllerOverlay `toml:"controller"`
	Recall     memoryRecallOverlay     `toml:"recall"`
	Decisions  memoryDecisionsOverlay  `toml:"decisions"`
	Extractor  memoryExtractorOverlay  `toml:"extractor"`
	Dream      dreamOverlay            `toml:"dream"`
	Session    memorySessionOverlay    `toml:"session"`
	Daily      memoryDailyOverlay      `toml:"daily"`
	File       memoryFileOverlay       `toml:"file"`
	Provider   memoryProviderOverlay   `toml:"provider"`
	Workspace  memoryWorkspaceOverlay  `toml:"workspace"`
}

type memoryControllerOverlay struct {
	Mode            *string                       `toml:"mode"`
	MaxLatency      *time.Duration                `toml:"max_latency"`
	DefaultOpOnFail *string                       `toml:"default_op_on_fail"`
	LLM             memoryControllerLLMOverlay    `toml:"llm"`
	Policy          memoryControllerPolicyOverlay `toml:"policy"`
}

type memoryControllerLLMOverlay struct {
	Enabled       *bool          `toml:"enabled"`
	Model         *string        `toml:"model"`
	TopK          *int           `toml:"top_k"`
	PromptVersion *string        `toml:"prompt_version"`
	Timeout       *time.Duration `toml:"timeout"`
	MaxTokensOut  *int           `toml:"max_tokens_out"`
}

type memoryControllerPolicyOverlay struct {
	MaxContentChars *int      `toml:"max_content_chars"`
	MaxWritesPerMin *int      `toml:"max_writes_per_min"`
	AllowOrigins    *[]string `toml:"allow_origins"`
}

type memoryRecallOverlay struct {
	TopK                   *int                         `toml:"top_k"`
	RawCandidates          *int                         `toml:"raw_candidates"`
	Fusion                 *string                      `toml:"fusion"`
	IncludeAlreadySurfaced *bool                        `toml:"include_already_surfaced"`
	IncludeSystem          *bool                        `toml:"include_system"`
	Weights                memoryRecallWeightsOverlay   `toml:"weights"`
	Freshness              memoryRecallFreshnessOverlay `toml:"freshness"`
	Signals                memoryRecallSignalsOverlay   `toml:"signals"`
}

type memoryRecallWeightsOverlay struct {
	BM25Unicode  *float64 `toml:"bm25_unicode"`
	BM25Trigram  *float64 `toml:"bm25_trigram"`
	Recency      *float64 `toml:"recency"`
	RecallSignal *float64 `toml:"recall_signal"`
}

type memoryRecallFreshnessOverlay struct {
	BannerAfterDays *int `toml:"banner_after_days"`
}

type memoryRecallSignalsOverlay struct {
	QueueCapacity  *int  `toml:"queue_capacity"`
	WorkerRetryMax *int  `toml:"worker_retry_max"`
	MetricsEnabled *bool `toml:"metrics_enabled"`
}

type memoryDecisionsOverlay struct {
	PruneAfterAppliedDays *int   `toml:"prune_after_applied_days"`
	KeepAuditSummary      *bool  `toml:"keep_audit_summary"`
	MaxPostContentBytes   *int64 `toml:"max_post_content_bytes"`
}

type memoryExtractorOverlay struct {
	Enabled          *bool                       `toml:"enabled"`
	Mode             *string                     `toml:"mode"`
	ThrottleTurns    *int                        `toml:"throttle_turns"`
	Deadline         *time.Duration              `toml:"deadline"`
	SandboxInboxOnly *bool                       `toml:"sandbox_inbox_only"`
	InboxPath        *string                     `toml:"inbox_path"`
	DLQPath          *string                     `toml:"dlq_path"`
	Model            *string                     `toml:"model"`
	Queue            memoryExtractorQueueOverlay `toml:"queue"`
}

type memoryExtractorQueueOverlay struct {
	Capacity    *int `toml:"capacity"`
	CoalesceMax *int `toml:"coalesce_max"`
}

type dreamOverlay struct {
	Enabled       *bool                     `toml:"enabled"`
	Agent         *string                   `toml:"agent"`
	MinHours      *float64                  `toml:"min_hours"`
	MinSessions   *int                      `toml:"min_sessions"`
	Debounce      *time.Duration            `toml:"debounce"`
	PromptVersion *string                   `toml:"prompt_version"`
	CheckInterval *time.Duration            `toml:"check_interval"`
	Gates         memoryDreamGatesOverlay   `toml:"gates"`
	Scoring       memoryDreamScoringOverlay `toml:"scoring"`
}

type memoryDreamGatesOverlay struct {
	MinUnpromoted  *int     `toml:"min_unpromoted"`
	MinRecallCount *int     `toml:"min_recall_count"`
	MinScore       *float64 `toml:"min_score"`
}

type memoryDreamScoringOverlay struct {
	RecencyHalfLifeDays *int                             `toml:"recency_half_life_days"`
	Weights             memoryDreamScoringWeightsOverlay `toml:"weights"`
}

type memoryDreamScoringWeightsOverlay struct {
	Frequency *float64 `toml:"frequency"`
	Relevance *float64 `toml:"relevance"`
	Recency   *float64 `toml:"recency"`
	Freshness *float64 `toml:"freshness"`
}

type memorySessionOverlay struct {
	LedgerFormat     *string        `toml:"ledger_format"`
	LedgerRoot       *string        `toml:"ledger_root"`
	EventsPurgeGrace *time.Duration `toml:"events_purge_grace"`
	ColdArchiveDays  *int           `toml:"cold_archive_days"`
	HardDeleteDays   *int           `toml:"hard_delete_days"`
	MaxArchiveBytes  *int64         `toml:"max_archive_bytes"`
	UnboundPartition *string        `toml:"unbound_partition"`
}

type memoryDailyOverlay struct {
	MaxBytes        *int64  `toml:"max_bytes"`
	MaxLines        *int    `toml:"max_lines"`
	RotateFormat    *string `toml:"rotate_format"`
	DreamingWindow  *int    `toml:"dreaming_window"`
	ColdArchiveDays *int    `toml:"cold_archive_days"`
	HardDeleteDays  *int    `toml:"hard_delete_days"`
	MaxArchiveBytes *int64  `toml:"max_archive_bytes"`
	SweepHour       *int    `toml:"sweep_hour"`
	ArchivePath     *string `toml:"archive_path"`
}

type memoryFileOverlay struct {
	MaxLines *int   `toml:"max_lines"`
	MaxBytes *int64 `toml:"max_bytes"`
}

type memoryProviderOverlay struct {
	Name             *string        `toml:"name"`
	Timeout          *time.Duration `toml:"timeout"`
	FailureThreshold *int           `toml:"failure_threshold"`
	Cooldown         *time.Duration `toml:"cooldown"`
}

type memoryWorkspaceOverlay struct {
	TOMLPath   *string `toml:"toml_path"`
	AutoCreate *bool   `toml:"auto_create"`
}

type skillsOverlay struct {
	Enabled                 *bool              `toml:"enabled"`
	DisabledSkills          *[]string          `toml:"disabled_skills"`
	PollInterval            *time.Duration     `toml:"poll_interval"`
	AllowedMarketplaceMCP   *[]string          `toml:"allowed_marketplace_mcp"`
	AllowedMarketplaceHooks *[]string          `toml:"allowed_marketplace_hooks"`
	Marketplace             marketplaceOverlay `toml:"marketplace"`
}

type extensionsOverlay struct {
	Marketplace extensionsMarketplaceOverlay `toml:"marketplace"`
	Resources   extensionsResourcesOverlay   `toml:"resources"`
}

type extensionsResourcesOverlay struct {
	AllowedKinds           *[]resources.ResourceKind    `toml:"allowed_kinds"`
	MaxScope               *resources.ResourceScopeKind `toml:"max_scope"`
	SnapshotRateLimit      extensionsRateLimitOverlay   `toml:"snapshot_rate_limit"`
	OperatorWriteRateLimit extensionsRateLimitOverlay   `toml:"operator_write_rate_limit"`
}

type extensionsRateLimitOverlay struct {
	Requests *int           `toml:"requests"`
	Window   *time.Duration `toml:"window"`
	Queue    *int           `toml:"queue"`
}

type toolsOverlay struct {
	Enabled               *bool                 `toml:"enabled"`
	HostedMCPEnabled      *bool                 `toml:"hosted_mcp_enabled"`
	DefaultMaxResultBytes *int64                `toml:"default_max_result_bytes"`
	HostedMCP             toolsHostedMCPOverlay `toml:"hosted_mcp"`
	Policy                toolsPolicyOverlay    `toml:"policy"`
}

type toolsHostedMCPOverlay struct {
	BindNonceTTLSeconds *int `toml:"bind_nonce_ttl_seconds"`
}

type toolsPolicyOverlay struct {
	ExternalDefault        *ToolsExternalDefault `toml:"external_default"`
	ApprovalTimeoutSeconds *int                  `toml:"approval_timeout_seconds"`
	TrustedSources         *[]string             `toml:"trusted_sources"`
}

type taskOverlay struct {
	Orchestration taskOrchestrationOverlay `toml:"orchestration"`
}

type taskOrchestrationOverlay struct {
	SummaryMaxBytes           *int                            `toml:"summary_max_bytes"`
	ContextBodyMaxBytes       *int                            `toml:"context_body_max_bytes"`
	ContextPriorAttempts      *int                            `toml:"context_prior_attempts"`
	ContextRecentEvents       *int                            `toml:"context_recent_events"`
	SpawnFailureLimit         *int                            `toml:"spawn_failure_limit"`
	SchedulerBadTickThreshold *int                            `toml:"scheduler_bad_tick_threshold"`
	SchedulerBadTickCooldown  *time.Duration                  `toml:"scheduler_bad_tick_cooldown"`
	DefaultMaxRuntime         *time.Duration                  `toml:"default_max_runtime"`
	BridgeNotificationTimeout *time.Duration                  `toml:"bridge_notification_timeout"`
	Profile                   taskOrchestrationProfileOverlay `toml:"profile"`
	Review                    taskOrchestrationReviewOverlay  `toml:"review"`
}

type taskOrchestrationProfileOverlay struct {
	DefaultCoordinatorMode    *string `toml:"default_coordinator_mode"`
	DefaultWorkerMode         *string `toml:"default_worker_mode"`
	DefaultSandboxMode        *string `toml:"default_sandbox_mode"`
	AllowTaskProviderOverride *bool   `toml:"allow_task_provider_override"`
	AllowTaskSandboxNone      *bool   `toml:"allow_task_sandbox_none"`
}

type taskOrchestrationReviewOverlay struct {
	DefaultPolicy             *string        `toml:"default_policy"`
	MaxRounds                 *int           `toml:"max_rounds"`
	MaxReviewAttempts         *int           `toml:"max_review_attempts"`
	Timeout                   *time.Duration `toml:"timeout"`
	RapidTerminalWindow       *time.Duration `toml:"rapid_terminal_window"`
	RapidTerminalLimit        *int           `toml:"rapid_terminal_limit"`
	MissingWorkMaxItems       *int           `toml:"missing_work_max_items"`
	MissingWorkItemMaxBytes   *int           `toml:"missing_work_item_max_bytes"`
	ReasonMaxBytes            *int           `toml:"reason_max_bytes"`
	ReviewTextMaxBytes        *int           `toml:"review_text_max_bytes"`
	NextRoundGuidanceMaxBytes *int           `toml:"next_round_guidance_max_bytes"`
	FailurePolicy             *string        `toml:"failure_policy"`
}

type networkOverlay struct {
	Enabled        *bool   `toml:"enabled"`
	DefaultChannel *string `toml:"default_channel"`
	Port           *int    `toml:"port"`
	MaxPayload     *int    `toml:"max_payload"`
	GreetInterval  *int    `toml:"greet_interval"`
	MaxReplayAge   *int    `toml:"max_replay_age"`
	MaxQueueDepth  *int    `toml:"max_queue_depth"`
}

type autonomyOverlay struct {
	Coordinator coordinatorOverlay `toml:"coordinator"`
}

type coordinatorOverlay struct {
	Enabled               *bool          `toml:"enabled"`
	AgentName             *string        `toml:"agent_name"`
	Provider              *string        `toml:"provider"`
	Model                 *string        `toml:"model"`
	DefaultTTL            *time.Duration `toml:"default_ttl"`
	MaxChildren           *int           `toml:"max_children"`
	MaxActivePerWorkspace *int           `toml:"max_active_per_workspace"`
}

type marketplaceOverlay struct {
	Registry *string `toml:"registry"`
	BaseURL  *string `toml:"base_url"`
}

type extensionsMarketplaceOverlay struct {
	Registry *string `toml:"registry"`
	BaseURL  *string `toml:"base_url"`
}

type hooksOverlay struct {
	Declarations []parsedHookDeclaration `toml:"declarations"`
}

type mcpServerOverlay struct {
	Name      *string             `toml:"name"`
	Transport *MCPServerTransport `toml:"transport"`
	Command   *string             `toml:"command"`
	Args      *[]string           `toml:"args"`
	Env       *map[string]string  `toml:"env"`
	SecretEnv *map[string]string  `toml:"secret_env"`
	URL       *string             `toml:"url"`
	Auth      mcpAuthOverlay      `toml:"auth"`
}

type mcpAuthOverlay struct {
	Type             *MCPAuthType `toml:"type"`
	IssuerURL        *string      `toml:"issuer_url"`
	MetadataURL      *string      `toml:"metadata_url"`
	AuthorizationURL *string      `toml:"authorization_url"`
	TokenURL         *string      `toml:"token_url"`
	RevocationURL    *string      `toml:"revocation_url"`
	ClientID         *string      `toml:"client_id"`
	ClientSecretRef  *string      `toml:"client_secret_ref"`
	Scopes           *[]string    `toml:"scopes"`
}

// ApplyConfigOverlayFile deep-merges an optional TOML config file into dst.
func ApplyConfigOverlayFile(path string, dst *Config) error {
	if dst == nil {
		return errors.New("config: destination config is required")
	}

	overlay, err := loadConfigOverlayFile(path)
	if err != nil {
		return err
	}

	return overlay.Apply(dst)
}

func loadConfigOverlayFile(path string) (configOverlay, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configOverlay{}, nil
		}
		return configOverlay{}, FileError{Op: mergeReadKey, Path: path, Err: err}
	}

	return loadConfigOverlayBytes(contents, path)
}

func loadConfigOverlayBytes(contents []byte, source string) (configOverlay, error) {
	var overlay configOverlay

	meta, err := burnttoml.Decode(string(contents), &overlay)
	if err != nil {
		return overlay, FileError{Op: "decode", Path: source, Err: err}
	}

	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		if err := rejectRemovedProviderModelKeys(source, undecoded); err != nil {
			return overlay, err
		}
		return overlay, fmt.Errorf("unknown config keys in %q: %s", source, joinTOMLKeys(undecoded))
	}

	return overlay, nil
}

func rejectRemovedProviderModelKeys(source string, keys []burnttoml.Key) error {
	for _, key := range sortedTOMLKeys(keys) {
		if len(key) != 3 || key[0] != providersConfigKey {
			continue
		}
		replacement := ""
		switch key[2] {
		case "default_model":
			replacement = fmt.Sprintf("providers.%s.models.default", key[1])
		case "supported_models":
			replacement = fmt.Sprintf("providers.%s.models.curated", key[1])
		case "supports_reasoning_effort":
			replacement = fmt.Sprintf("providers.%s.models.curated[].reasoning_efforts", key[1])
		}
		if replacement != "" {
			return fmt.Errorf(
				"removed config key %q in %q: use %q",
				key.String(),
				source,
				replacement,
			)
		}
	}
	return nil
}

func (o *configOverlay) Apply(dst *Config) error {
	o.Daemon.Apply(&dst.Daemon)
	o.HTTP.Apply(&dst.HTTP)
	o.Defaults.Apply(&dst.Defaults)
	o.Agents.Apply(&dst.Agents)
	o.Limits.Apply(&dst.Limits)
	o.Session.Apply(&dst.Session)
	o.Permissions.Apply(&dst.Permissions)
	if len(o.MCPServers) > 0 {
		dst.MCPServers = applyMCPServerOverlays(dst.MCPServers, o.MCPServers)
	}
	applyProviderOverlays(dst, o.Providers)
	o.ModelCatalog.Apply(&dst.ModelCatalog)
	applySandboxOverlays(dst, o.Sandboxes)
	o.Observability.Apply(&dst.Observability)
	o.Log.Apply(&dst.Log)
	o.Memory.Apply(&dst.Memory)
	o.Skills.Apply(&dst.Skills)
	o.Extensions.Apply(&dst.Extensions)
	o.Tools.Apply(&dst.Tools)
	if err := o.Automation.Apply(&dst.Automation); err != nil {
		return err
	}
	o.Task.Apply(&dst.Task)
	o.Network.Apply(&dst.Network)
	o.Autonomy.Apply(&dst.Autonomy)
	return o.Hooks.Apply(&dst.Hooks)
}

func (o daemonOverlay) Apply(dst *DaemonConfig) {
	if o.Socket != nil {
		dst.Socket = *o.Socket
	}
}

func (o httpOverlay) Apply(dst *HTTPConfig) {
	if o.Host != nil {
		dst.Host = *o.Host
	}
	if o.Port != nil {
		dst.Port = *o.Port
	}
}

func (o defaultsOverlay) Apply(dst *DefaultsConfig) {
	if o.Agent != nil {
		dst.Agent = *o.Agent
	}
	if o.Provider != nil {
		dst.Provider = *o.Provider
	}
	if o.Sandbox != nil {
		dst.Sandbox = *o.Sandbox
	}
}

func (o agentsOverlay) Apply(dst *AgentsConfig) {
	o.Soul.Apply(&dst.Soul)
	o.Heartbeat.Apply(&dst.Heartbeat)
}

func (o soulOverlay) Apply(dst *SoulConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.MaxBodyBytes != nil {
		dst.MaxBodyBytes = *o.MaxBodyBytes
	}
	if o.ContextProjectionBytes != nil {
		dst.ContextProjectionBytes = *o.ContextProjectionBytes
	}
}

func (o heartbeatOverlay) Apply(dst *HeartbeatConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.MaxBodyBytes != nil {
		dst.MaxBodyBytes = *o.MaxBodyBytes
	}
	if o.ContextProjectionBytes != nil {
		dst.ContextProjectionBytes = *o.ContextProjectionBytes
	}
	if o.MinInterval != nil {
		dst.MinInterval = *o.MinInterval
	}
	if o.DefaultInterval != nil {
		dst.DefaultInterval = *o.DefaultInterval
	}
	if o.WakeCooldown != nil {
		dst.WakeCooldown = *o.WakeCooldown
	}
	if o.MaxWakesPerCycle != nil {
		dst.MaxWakesPerCycle = *o.MaxWakesPerCycle
	}
	if o.ActiveSessionOnly != nil {
		dst.ActiveSessionOnly = *o.ActiveSessionOnly
	}
	if o.AllowActiveHoursPreferences != nil {
		dst.AllowActiveHoursPreferences = *o.AllowActiveHoursPreferences
	}
	if o.WakeEventRetention != nil {
		dst.WakeEventRetention = *o.WakeEventRetention
	}
	if o.SessionHealthStaleAfter != nil {
		dst.SessionHealthStaleAfter = *o.SessionHealthStaleAfter
	}
	if o.SessionHealthHookMinInterval != nil {
		dst.SessionHealthHookMinInterval = *o.SessionHealthHookMinInterval
	}
}

func (o limitsOverlay) Apply(dst *LimitsConfig) {
	if o.MaxConcurrentAgents != nil {
		dst.MaxConcurrentAgents = *o.MaxConcurrentAgents
	}
}

func (o sessionOverlay) Apply(dst *SessionConfig) {
	o.Limits.Apply(&dst.Limits)
	o.Supervision.Apply(&dst.Supervision)
}

func (o sessionLimitsOverlay) Apply(dst *SessionLimitsConfig) {
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
}

func (o sessionSupervisionOverlay) Apply(dst *SessionSupervisionConfig) {
	if o.ActivityHeartbeatInterval != nil {
		dst.ActivityHeartbeatInterval = *o.ActivityHeartbeatInterval
	}
	if o.ProgressNotifyInterval != nil {
		dst.ProgressNotifyInterval = *o.ProgressNotifyInterval
	}
	if o.PromptDeadline != nil {
		dst.PromptDeadline = *o.PromptDeadline
	}
	if o.InactivityWarningAfter != nil {
		dst.InactivityWarningAfter = *o.InactivityWarningAfter
	}
	if o.InactivityTimeout != nil {
		dst.InactivityTimeout = *o.InactivityTimeout
	}
	if o.TimeoutCancelGrace != nil {
		dst.TimeoutCancelGrace = *o.TimeoutCancelGrace
	}
}

func (o permissionsOverlay) Apply(dst *PermissionsConfig) {
	if o.Mode != nil {
		dst.Mode = *o.Mode
	}
}

func (o providerOverlay) Apply(dst *ProviderConfig) {
	if o.Command != nil {
		dst.Command = *o.Command
	}
	if o.DisplayName != nil {
		dst.DisplayName = *o.DisplayName
	}
	if o.Models != nil {
		o.Models.Apply(&dst.Models)
	}
	if o.Harness != nil {
		dst.Harness = *o.Harness
	}
	if o.RuntimeProvider != nil {
		dst.RuntimeProvider = *o.RuntimeProvider
	}
	if o.Transport != nil {
		dst.Transport = *o.Transport
	}
	if o.BaseURL != nil {
		dst.BaseURL = *o.BaseURL
	}
	if o.AuthMode != nil {
		dst.AuthMode = *o.AuthMode
	}
	if o.EnvPolicy != nil {
		dst.EnvPolicy = *o.EnvPolicy
	}
	if o.HomePolicy != nil {
		dst.HomePolicy = *o.HomePolicy
	}
	if o.AuthStatusCmd != nil {
		dst.AuthStatusCmd = *o.AuthStatusCmd
	}
	if o.AuthLoginCmd != nil {
		dst.AuthLoginCmd = *o.AuthLoginCmd
	}
	if o.SessionMCP != nil {
		dst.SessionMCP = new(*o.SessionMCP)
	}
	if o.Aliases != nil {
		dst.Aliases = append([]string(nil), (*o.Aliases)...)
	}
	if len(o.CredentialSlots) > 0 {
		dst.CredentialSlots = applyProviderCredentialOverlays(o.CredentialSlots)
	}
	if len(o.MCPServers) > 0 {
		dst.MCPServers = applyMCPServerOverlays(dst.MCPServers, o.MCPServers)
	}
}

func (o providerModelsOverlay) Apply(dst *ProviderModelsConfig) {
	if o.Default != nil {
		dst.Default = *o.Default
	}
	if o.Curated != nil {
		dst.Curated = cloneProviderModelConfigs(o.Curated)
	}
	o.Discovery.Apply(&dst.Discovery)
}

func (o providerModelsDiscoveryOverlay) Apply(dst *ProviderModelsDiscoveryConfig) {
	if o.Enabled != nil {
		dst.Enabled = new(*o.Enabled)
	}
	if o.Command != nil {
		dst.Command = *o.Command
	}
	if o.Endpoint != nil {
		dst.Endpoint = *o.Endpoint
	}
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
}

func (o modelCatalogOverlay) Apply(dst *ModelCatalogConfig) {
	o.Sources.Apply(&dst.Sources)
}

func (o modelCatalogSourcesOverlay) Apply(dst *ModelCatalogSourcesConfig) {
	o.ModelsDev.Apply(&dst.ModelsDev)
}

func (o modelsDevSourceOverlay) Apply(dst *ModelsDevSourceConfig) {
	if o.Enabled != nil {
		dst.Enabled = new(*o.Enabled)
	}
	if o.Endpoint != nil {
		dst.Endpoint = *o.Endpoint
	}
	if o.TTL != nil {
		dst.TTL = *o.TTL
	}
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
}

func applyProviderCredentialOverlays(overlays []providerCredentialOverlay) []ProviderCredentialSlot {
	slots := make([]ProviderCredentialSlot, 0, len(overlays))
	for _, overlay := range overlays {
		var slot ProviderCredentialSlot
		if overlay.Name != nil {
			slot.Name = *overlay.Name
		}
		if overlay.TargetEnv != nil {
			slot.TargetEnv = *overlay.TargetEnv
		}
		if overlay.SecretRef != nil {
			slot.SecretRef = *overlay.SecretRef
		}
		if overlay.Kind != nil {
			slot.Kind = *overlay.Kind
		}
		if overlay.Required != nil {
			slot.Required = *overlay.Required
		}
		slots = append(slots, slot)
	}
	return slots
}

func (o sandboxOverlay) Apply(dst *SandboxProfile) {
	if o.Backend != nil {
		dst.Backend = *o.Backend
	}
	if o.SyncMode != nil {
		dst.SyncMode = *o.SyncMode
	}
	if o.Persistence != nil {
		dst.Persistence = *o.Persistence
	}
	if o.RuntimeRoot != nil {
		dst.RuntimeRoot = *o.RuntimeRoot
	}
	if o.Env != nil {
		dst.Env = mergeStringMaps(dst.Env, *o.Env)
	}
	if o.SecretEnv != nil {
		dst.SecretEnv = mergeStringMaps(dst.SecretEnv, *o.SecretEnv)
	}
	o.Network.Apply(&dst.Network)
	o.Daytona.Apply(&dst.Daytona)
}

func (o networkProfileOverlay) Apply(dst *NetworkProfile) {
	if o.AllowPublicIngress != nil {
		dst.AllowPublicIngress = *o.AllowPublicIngress
	}
	if o.AllowOutbound != nil {
		dst.AllowOutbound = *o.AllowOutbound
	}
	if o.AllowList != nil {
		dst.AllowList = append([]string(nil), (*o.AllowList)...)
	}
	if o.DenyList != nil {
		dst.DenyList = append([]string(nil), (*o.DenyList)...)
	}
	if o.Required != nil {
		dst.Required = *o.Required
	}
}

func (o daytonaProfileOverlay) Apply(dst *DaytonaProfile) {
	if o.APIURL != nil {
		dst.APIURL = *o.APIURL
	}
	if o.Target != nil {
		dst.Target = *o.Target
	}
	if o.Image != nil {
		dst.Image = *o.Image
	}
	if o.Snapshot != nil {
		dst.Snapshot = *o.Snapshot
	}
	if o.Class != nil {
		dst.Class = *o.Class
	}
	if o.AutoStop != nil {
		dst.AutoStop = *o.AutoStop
	}
	if o.AutoArchive != nil {
		dst.AutoArchive = *o.AutoArchive
	}
}

func (o observabilityOverlay) Apply(dst *ObservabilityConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.RetentionDays != nil {
		dst.RetentionDays = *o.RetentionDays
	}
	if o.MaxGlobalBytes != nil {
		dst.MaxGlobalBytes = *o.MaxGlobalBytes
	}
	if o.AgentProbeTimeout != nil {
		dst.AgentProbeTimeout = *o.AgentProbeTimeout
	}
	o.Transcripts.Apply(&dst.Transcripts)
}

func (o observabilityTranscriptsOverlay) Apply(dst *ObservabilityTranscriptConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.SegmentBytes != nil {
		dst.SegmentBytes = *o.SegmentBytes
	}
	if o.MaxBytesPerSession != nil {
		dst.MaxBytesPerSession = *o.MaxBytesPerSession
	}
}

func (o logOverlay) Apply(dst *LogConfig) {
	if o.Level != nil {
		dst.Level = *o.Level
	}
}

func (o *memoryOverlay) Apply(dst *MemoryConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.GlobalDir != nil && strings.TrimSpace(*o.GlobalDir) != "" {
		dst.GlobalDir = *o.GlobalDir
	}
	o.Controller.Apply(&dst.Controller)
	o.Recall.Apply(&dst.Recall)
	o.Decisions.Apply(&dst.Decisions)
	o.Extractor.Apply(&dst.Extractor)
	o.Dream.Apply(&dst.Dream)
	o.Session.Apply(&dst.Session)
	o.Daily.Apply(&dst.Daily)
	o.File.Apply(&dst.File)
	o.Provider.Apply(&dst.Provider)
	o.Workspace.Apply(&dst.Workspace)
}

func (o memoryControllerOverlay) Apply(dst *MemoryControllerConfig) {
	if o.Mode != nil {
		dst.Mode = *o.Mode
	}
	if o.MaxLatency != nil {
		dst.MaxLatency = *o.MaxLatency
	}
	if o.DefaultOpOnFail != nil {
		dst.DefaultOpOnFail = *o.DefaultOpOnFail
	}
	o.LLM.Apply(&dst.LLM)
	o.Policy.Apply(&dst.Policy)
}

func (o memoryControllerLLMOverlay) Apply(dst *MemoryControllerLLMConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.Model != nil {
		dst.Model = *o.Model
	}
	if o.TopK != nil {
		dst.TopK = *o.TopK
	}
	if o.PromptVersion != nil {
		dst.PromptVersion = *o.PromptVersion
	}
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
	if o.MaxTokensOut != nil {
		dst.MaxTokensOut = *o.MaxTokensOut
	}
}

func (o memoryControllerPolicyOverlay) Apply(dst *MemoryControllerPolicyConfig) {
	if o.MaxContentChars != nil {
		dst.MaxContentChars = *o.MaxContentChars
	}
	if o.MaxWritesPerMin != nil {
		dst.MaxWritesPerMin = *o.MaxWritesPerMin
	}
	if o.AllowOrigins != nil {
		dst.AllowOrigins = append([]string(nil), (*o.AllowOrigins)...)
	}
}

func (o memoryRecallOverlay) Apply(dst *MemoryRecallConfig) {
	if o.TopK != nil {
		dst.TopK = *o.TopK
	}
	if o.RawCandidates != nil {
		dst.RawCandidates = *o.RawCandidates
	}
	if o.Fusion != nil {
		dst.Fusion = *o.Fusion
	}
	if o.IncludeAlreadySurfaced != nil {
		dst.IncludeAlreadySurfaced = *o.IncludeAlreadySurfaced
	}
	if o.IncludeSystem != nil {
		dst.IncludeSystem = *o.IncludeSystem
	}
	o.Weights.Apply(&dst.Weights)
	o.Freshness.Apply(&dst.Freshness)
	o.Signals.Apply(&dst.Signals)
}

func (o memoryRecallWeightsOverlay) Apply(dst *MemoryRecallWeightsConfig) {
	if o.BM25Unicode != nil {
		dst.BM25Unicode = *o.BM25Unicode
	}
	if o.BM25Trigram != nil {
		dst.BM25Trigram = *o.BM25Trigram
	}
	if o.Recency != nil {
		dst.Recency = *o.Recency
	}
	if o.RecallSignal != nil {
		dst.RecallSignal = *o.RecallSignal
	}
}

func (o memoryRecallFreshnessOverlay) Apply(dst *MemoryRecallFreshnessConfig) {
	if o.BannerAfterDays != nil {
		dst.BannerAfterDays = *o.BannerAfterDays
	}
}

func (o memoryRecallSignalsOverlay) Apply(dst *MemoryRecallSignalsConfig) {
	if o.QueueCapacity != nil {
		dst.QueueCapacity = *o.QueueCapacity
	}
	if o.WorkerRetryMax != nil {
		dst.WorkerRetryMax = *o.WorkerRetryMax
	}
	if o.MetricsEnabled != nil {
		dst.MetricsEnabled = *o.MetricsEnabled
	}
}

func (o memoryDecisionsOverlay) Apply(dst *MemoryDecisionsConfig) {
	if o.PruneAfterAppliedDays != nil {
		dst.PruneAfterAppliedDays = *o.PruneAfterAppliedDays
	}
	if o.KeepAuditSummary != nil {
		dst.KeepAuditSummary = *o.KeepAuditSummary
	}
	if o.MaxPostContentBytes != nil {
		dst.MaxPostContentBytes = *o.MaxPostContentBytes
	}
}

func (o memoryExtractorOverlay) Apply(dst *MemoryExtractorConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.Mode != nil {
		dst.Mode = *o.Mode
	}
	if o.ThrottleTurns != nil {
		dst.ThrottleTurns = *o.ThrottleTurns
	}
	if o.Deadline != nil {
		dst.Deadline = *o.Deadline
	}
	if o.SandboxInboxOnly != nil {
		dst.SandboxInboxOnly = *o.SandboxInboxOnly
	}
	if o.InboxPath != nil {
		dst.InboxPath = *o.InboxPath
	}
	if o.DLQPath != nil {
		dst.DLQPath = *o.DLQPath
	}
	if o.Model != nil {
		dst.Model = *o.Model
	}
	o.Queue.Apply(&dst.Queue)
}

func (o memoryExtractorQueueOverlay) Apply(dst *MemoryExtractorQueueConfig) {
	if o.Capacity != nil {
		dst.Capacity = *o.Capacity
	}
	if o.CoalesceMax != nil {
		dst.CoalesceMax = *o.CoalesceMax
	}
}

func (o dreamOverlay) Apply(dst *DreamConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.Agent != nil {
		dst.Agent = *o.Agent
	}
	if o.MinHours != nil {
		dst.MinHours = *o.MinHours
	}
	if o.MinSessions != nil {
		dst.MinSessions = *o.MinSessions
	}
	if o.Debounce != nil {
		dst.Debounce = *o.Debounce
	}
	if o.PromptVersion != nil {
		dst.PromptVersion = *o.PromptVersion
	}
	if o.CheckInterval != nil {
		dst.CheckInterval = *o.CheckInterval
	}
	o.Gates.Apply(&dst.Gates)
	o.Scoring.Apply(&dst.Scoring)
}

func (o memoryDreamGatesOverlay) Apply(dst *MemoryDreamGatesConfig) {
	if o.MinUnpromoted != nil {
		dst.MinUnpromoted = *o.MinUnpromoted
	}
	if o.MinRecallCount != nil {
		dst.MinRecallCount = *o.MinRecallCount
	}
	if o.MinScore != nil {
		dst.MinScore = *o.MinScore
	}
}

func (o memoryDreamScoringOverlay) Apply(dst *MemoryDreamScoringConfig) {
	if o.RecencyHalfLifeDays != nil {
		dst.RecencyHalfLifeDays = *o.RecencyHalfLifeDays
	}
	o.Weights.Apply(&dst.Weights)
}

func (o memoryDreamScoringWeightsOverlay) Apply(dst *MemoryDreamScoringWeightsConfig) {
	if o.Frequency != nil {
		dst.Frequency = *o.Frequency
	}
	if o.Relevance != nil {
		dst.Relevance = *o.Relevance
	}
	if o.Recency != nil {
		dst.Recency = *o.Recency
	}
	if o.Freshness != nil {
		dst.Freshness = *o.Freshness
	}
}

func (o memorySessionOverlay) Apply(dst *MemorySessionConfig) {
	if o.LedgerFormat != nil {
		dst.LedgerFormat = *o.LedgerFormat
	}
	if o.LedgerRoot != nil {
		dst.LedgerRoot = *o.LedgerRoot
	}
	if o.EventsPurgeGrace != nil {
		dst.EventsPurgeGrace = *o.EventsPurgeGrace
	}
	if o.ColdArchiveDays != nil {
		dst.ColdArchiveDays = *o.ColdArchiveDays
	}
	if o.HardDeleteDays != nil {
		dst.HardDeleteDays = *o.HardDeleteDays
	}
	if o.MaxArchiveBytes != nil {
		dst.MaxArchiveBytes = *o.MaxArchiveBytes
	}
	if o.UnboundPartition != nil {
		dst.UnboundPartition = *o.UnboundPartition
	}
}

func (o memoryDailyOverlay) Apply(dst *MemoryDailyConfig) {
	if o.MaxBytes != nil {
		dst.MaxBytes = *o.MaxBytes
	}
	if o.MaxLines != nil {
		dst.MaxLines = *o.MaxLines
	}
	if o.RotateFormat != nil {
		dst.RotateFormat = *o.RotateFormat
	}
	if o.DreamingWindow != nil {
		dst.DreamingWindow = *o.DreamingWindow
	}
	if o.ColdArchiveDays != nil {
		dst.ColdArchiveDays = *o.ColdArchiveDays
	}
	if o.HardDeleteDays != nil {
		dst.HardDeleteDays = *o.HardDeleteDays
	}
	if o.MaxArchiveBytes != nil {
		dst.MaxArchiveBytes = *o.MaxArchiveBytes
	}
	if o.SweepHour != nil {
		dst.SweepHour = *o.SweepHour
	}
	if o.ArchivePath != nil {
		dst.ArchivePath = *o.ArchivePath
	}
}

func (o memoryFileOverlay) Apply(dst *MemoryFileConfig) {
	if o.MaxLines != nil {
		dst.MaxLines = *o.MaxLines
	}
	if o.MaxBytes != nil {
		dst.MaxBytes = *o.MaxBytes
	}
}

func (o memoryProviderOverlay) Apply(dst *MemoryProviderConfig) {
	if o.Name != nil {
		dst.Name = *o.Name
	}
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
	if o.FailureThreshold != nil {
		dst.FailureThreshold = *o.FailureThreshold
	}
	if o.Cooldown != nil {
		dst.Cooldown = *o.Cooldown
	}
}

func (o memoryWorkspaceOverlay) Apply(dst *MemoryWorkspaceConfig) {
	if o.TOMLPath != nil {
		dst.TOMLPath = *o.TOMLPath
	}
	if o.AutoCreate != nil {
		dst.AutoCreate = *o.AutoCreate
	}
}

func (o skillsOverlay) Apply(dst *SkillsConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.DisabledSkills != nil {
		dst.DisabledSkills = append([]string(nil), (*o.DisabledSkills)...)
	}
	if o.PollInterval != nil {
		dst.PollInterval = *o.PollInterval
	}
	if o.AllowedMarketplaceMCP != nil {
		dst.AllowedMarketplaceMCP = append([]string(nil), (*o.AllowedMarketplaceMCP)...)
	}
	if o.AllowedMarketplaceHooks != nil {
		dst.AllowedMarketplaceHooks = append([]string(nil), (*o.AllowedMarketplaceHooks)...)
	}
	o.Marketplace.Apply(&dst.Marketplace)
}

func (o extensionsOverlay) Apply(dst *ExtensionsConfig) {
	o.Marketplace.Apply(&dst.Marketplace)
	o.Resources.Apply(&dst.Resources)
}

func (o extensionsResourcesOverlay) Apply(dst *ExtensionsResourcesConfig) {
	if o.AllowedKinds != nil {
		dst.AllowedKinds = append([]resources.ResourceKind(nil), (*o.AllowedKinds)...)
	}
	if o.MaxScope != nil {
		dst.MaxScope = *o.MaxScope
	}
	o.SnapshotRateLimit.Apply(&dst.SnapshotRateLimit)
	o.OperatorWriteRateLimit.Apply(&dst.OperatorWriteRateLimit)
}

func (o extensionsRateLimitOverlay) Apply(dst *ExtensionsResourceRateLimitConfig) {
	if o.Requests != nil {
		dst.Requests = *o.Requests
	}
	if o.Window != nil {
		dst.Window = *o.Window
	}
	if o.Queue != nil {
		dst.Queue = *o.Queue
	}
}

func (o toolsOverlay) Apply(dst *ToolsConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.HostedMCPEnabled != nil {
		dst.HostedMCPEnabled = *o.HostedMCPEnabled
	}
	if o.DefaultMaxResultBytes != nil {
		dst.DefaultMaxResultBytes = *o.DefaultMaxResultBytes
	}
	o.HostedMCP.Apply(&dst.HostedMCP)
	o.Policy.Apply(&dst.Policy)
}

func (o toolsHostedMCPOverlay) Apply(dst *ToolsHostedMCPConfig) {
	if o.BindNonceTTLSeconds != nil {
		dst.BindNonceTTLSeconds = *o.BindNonceTTLSeconds
	}
}

func (o toolsPolicyOverlay) Apply(dst *ToolsPolicyConfig) {
	if o.ExternalDefault != nil {
		dst.ExternalDefault = *o.ExternalDefault
	}
	if o.ApprovalTimeoutSeconds != nil {
		dst.ApprovalTimeoutSeconds = *o.ApprovalTimeoutSeconds
	}
	if o.TrustedSources != nil {
		dst.TrustedSources = append([]string(nil), (*o.TrustedSources)...)
	}
}

func (o taskOverlay) Apply(dst *TaskConfig) {
	o.Orchestration.Apply(&dst.Orchestration)
}

func (o taskOrchestrationOverlay) Apply(dst *TaskOrchestrationConfig) {
	if o.SummaryMaxBytes != nil {
		dst.SummaryMaxBytes = *o.SummaryMaxBytes
	}
	if o.ContextBodyMaxBytes != nil {
		dst.ContextBodyMaxBytes = *o.ContextBodyMaxBytes
	}
	if o.ContextPriorAttempts != nil {
		dst.ContextPriorAttempts = *o.ContextPriorAttempts
	}
	if o.ContextRecentEvents != nil {
		dst.ContextRecentEvents = *o.ContextRecentEvents
	}
	if o.SpawnFailureLimit != nil {
		dst.SpawnFailureLimit = *o.SpawnFailureLimit
	}
	if o.SchedulerBadTickThreshold != nil {
		dst.SchedulerBadTickThreshold = *o.SchedulerBadTickThreshold
	}
	if o.SchedulerBadTickCooldown != nil {
		dst.SchedulerBadTickCooldown = *o.SchedulerBadTickCooldown
	}
	if o.DefaultMaxRuntime != nil {
		dst.DefaultMaxRuntime = *o.DefaultMaxRuntime
	}
	if o.BridgeNotificationTimeout != nil {
		dst.BridgeNotificationTimeout = *o.BridgeNotificationTimeout
	}
	o.Profile.Apply(&dst.Profile)
	o.Review.Apply(&dst.Review)
}

func (o taskOrchestrationProfileOverlay) Apply(dst *TaskOrchestrationProfileConfig) {
	if o.DefaultCoordinatorMode != nil {
		dst.DefaultCoordinatorMode = *o.DefaultCoordinatorMode
	}
	if o.DefaultWorkerMode != nil {
		dst.DefaultWorkerMode = *o.DefaultWorkerMode
	}
	if o.DefaultSandboxMode != nil {
		dst.DefaultSandboxMode = *o.DefaultSandboxMode
	}
	if o.AllowTaskProviderOverride != nil {
		dst.AllowTaskProviderOverride = *o.AllowTaskProviderOverride
	}
	if o.AllowTaskSandboxNone != nil {
		dst.AllowTaskSandboxNone = *o.AllowTaskSandboxNone
	}
}

func (o taskOrchestrationReviewOverlay) Apply(dst *TaskOrchestrationReviewConfig) {
	if o.DefaultPolicy != nil {
		dst.DefaultPolicy = *o.DefaultPolicy
	}
	if o.MaxRounds != nil {
		dst.MaxRounds = *o.MaxRounds
	}
	if o.MaxReviewAttempts != nil {
		dst.MaxReviewAttempts = *o.MaxReviewAttempts
	}
	if o.Timeout != nil {
		dst.Timeout = *o.Timeout
	}
	if o.RapidTerminalWindow != nil {
		dst.RapidTerminalWindow = *o.RapidTerminalWindow
	}
	if o.RapidTerminalLimit != nil {
		dst.RapidTerminalLimit = *o.RapidTerminalLimit
	}
	if o.MissingWorkMaxItems != nil {
		dst.MissingWorkMaxItems = *o.MissingWorkMaxItems
	}
	if o.MissingWorkItemMaxBytes != nil {
		dst.MissingWorkItemMaxBytes = *o.MissingWorkItemMaxBytes
	}
	if o.ReasonMaxBytes != nil {
		dst.ReasonMaxBytes = *o.ReasonMaxBytes
	}
	if o.ReviewTextMaxBytes != nil {
		dst.ReviewTextMaxBytes = *o.ReviewTextMaxBytes
	}
	if o.NextRoundGuidanceMaxBytes != nil {
		dst.NextRoundGuidanceMaxBytes = *o.NextRoundGuidanceMaxBytes
	}
	if o.FailurePolicy != nil {
		dst.FailurePolicy = *o.FailurePolicy
	}
}

func (o networkOverlay) Apply(dst *NetworkConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.DefaultChannel != nil {
		dst.DefaultChannel = *o.DefaultChannel
	}
	if o.Port != nil {
		dst.Port = *o.Port
	}
	if o.MaxPayload != nil {
		dst.MaxPayload = *o.MaxPayload
	}
	if o.GreetInterval != nil {
		dst.GreetInterval = *o.GreetInterval
	}
	if o.MaxReplayAge != nil {
		dst.MaxReplayAge = *o.MaxReplayAge
	}
	if o.MaxQueueDepth != nil {
		dst.MaxQueueDepth = *o.MaxQueueDepth
	}
}

func (o autonomyOverlay) Apply(dst *AutonomyConfig) {
	o.Coordinator.Apply(&dst.Coordinator)
}

func (o coordinatorOverlay) Apply(dst *CoordinatorConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.AgentName != nil {
		dst.AgentName = *o.AgentName
	}
	if o.Provider != nil {
		dst.Provider = *o.Provider
	}
	if o.Model != nil {
		dst.Model = *o.Model
	}
	if o.DefaultTTL != nil {
		dst.DefaultTTL = *o.DefaultTTL
	}
	if o.MaxChildren != nil {
		dst.MaxChildren = *o.MaxChildren
	}
	if o.MaxActivePerWorkspace != nil {
		dst.MaxActivePerWorkspace = *o.MaxActivePerWorkspace
	}
}

func (o marketplaceOverlay) Apply(dst *MarketplaceConfig) {
	if o.Registry != nil {
		dst.Registry = *o.Registry
	}
	if o.BaseURL != nil {
		dst.BaseURL = *o.BaseURL
	}
}

func (o extensionsMarketplaceOverlay) Apply(dst *ExtensionsMarketplaceConfig) {
	if o.Registry != nil {
		dst.Registry = *o.Registry
	}
	if o.BaseURL != nil {
		dst.BaseURL = *o.BaseURL
	}
}

func (o hooksOverlay) Apply(dst *HooksConfig) error {
	if len(o.Declarations) == 0 {
		return nil
	}

	merged := cloneHookDecls(dst.Declarations)
	index := make(map[string]int, len(merged))
	for i, decl := range merged {
		if name := strings.TrimSpace(decl.Name); name != "" {
			index[name] = i
		}
	}

	for idx := range o.Declarations {
		raw := &o.Declarations[idx]
		decl, err := raw.toHookDecl(hookspkg.HookSourceConfig, "")
		if err != nil {
			return fmt.Errorf("hooks.declarations[%d]: %w", idx, err)
		}

		name := strings.TrimSpace(decl.Name)
		if existingIdx, ok := index[name]; ok && name != "" {
			merged[existingIdx] = decl
			continue
		}

		merged = append(merged, decl)
		if name != "" {
			index[name] = len(merged) - 1
		}
	}

	dst.Declarations = merged
	return nil
}

func (o mcpServerOverlay) Apply(dst *MCPServer) {
	if o.Name != nil {
		dst.Name = *o.Name
	}
	if o.Transport != nil {
		dst.Transport = *o.Transport
	}
	if o.Command != nil {
		dst.Command = *o.Command
	}
	if o.Args != nil {
		dst.Args = append([]string(nil), (*o.Args)...)
	}
	if o.Env != nil {
		dst.Env = mergeStringMaps(dst.Env, *o.Env)
	}
	if o.SecretEnv != nil {
		dst.SecretEnv = mergeStringMaps(dst.SecretEnv, *o.SecretEnv)
	}
	if o.URL != nil {
		dst.URL = *o.URL
	}
	o.Auth.Apply(&dst.Auth)
}

func (o mcpAuthOverlay) Apply(dst *MCPAuthConfig) {
	if o.Type != nil {
		dst.Type = *o.Type
	}
	if o.IssuerURL != nil {
		dst.IssuerURL = *o.IssuerURL
	}
	if o.MetadataURL != nil {
		dst.MetadataURL = *o.MetadataURL
	}
	if o.AuthorizationURL != nil {
		dst.AuthorizationURL = *o.AuthorizationURL
	}
	if o.TokenURL != nil {
		dst.TokenURL = *o.TokenURL
	}
	if o.RevocationURL != nil {
		dst.RevocationURL = *o.RevocationURL
	}
	if o.ClientID != nil {
		dst.ClientID = *o.ClientID
	}
	if o.ClientSecretRef != nil {
		dst.ClientSecretRef = *o.ClientSecretRef
	}
	if o.Scopes != nil {
		dst.Scopes = append([]string(nil), (*o.Scopes)...)
	}
}

func applyProviderOverlays(dst *Config, overlays map[string]providerOverlay) {
	if len(overlays) == 0 {
		return
	}
	if dst.Providers == nil {
		dst.Providers = make(map[string]ProviderConfig, len(overlays))
	}

	for name, overlay := range overlays {
		providerName := CanonicalProviderName(name)
		if providerName == "" {
			continue
		}
		provider := dst.Providers[providerName]
		overlay.Apply(&provider)
		dst.Providers[providerName] = provider
	}
}

func applySandboxOverlays(dst *Config, overlays map[string]sandboxOverlay) {
	if len(overlays) == 0 {
		return
	}
	if dst.Sandboxes == nil {
		dst.Sandboxes = make(map[string]SandboxProfile, len(overlays))
	}

	for name, overlay := range overlays {
		profile := dst.Sandboxes[name]
		overlay.Apply(&profile)
		dst.Sandboxes[name] = profile
	}
}

func applyMCPServerOverlays(base []MCPServer, overlays []mcpServerOverlay) []MCPServer {
	merged := cloneMCPServers(base)
	index := indexMCPServersByName(merged)

	for _, overlay := range overlays {
		name := ""
		if overlay.Name != nil {
			name = normalizeMCPServerName(*overlay.Name)
		}

		if idx, ok := index[name]; ok && name != "" {
			server := merged[idx]
			overlay.Apply(&server)
			merged[idx] = server
			if normalized := normalizeMCPServerName(server.Name); normalized != name {
				delete(index, name)
				if normalized != "" {
					index[normalized] = idx
				}
			}
			continue
		}

		var server MCPServer
		overlay.Apply(&server)
		merged = append(merged, server)
		if normalized := normalizeMCPServerName(server.Name); normalized != "" {
			index[normalized] = len(merged) - 1
		}
	}

	return merged
}

func joinTOMLKeys(keys []burnttoml.Key) string {
	if len(keys) == 0 {
		return ""
	}

	sorted := sortedTOMLKeys(keys)
	values := make([]string, 0, len(sorted))
	for _, key := range sorted {
		values = append(values, key.String())
	}

	return strings.Join(values, ", ")
}

func sortedTOMLKeys(keys []burnttoml.Key) []burnttoml.Key {
	sorted := append([]burnttoml.Key(nil), keys...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].String() < sorted[j].String()
	})
	return sorted
}
