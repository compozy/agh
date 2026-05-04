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
	Sandboxes     map[string]sandboxOverlay  `toml:"sandboxes"`
	Observability observabilityOverlay       `toml:"observability"`
	Log           logOverlay                 `toml:"log"`
	Memory        memoryOverlay              `toml:"memory"`
	Skills        skillsOverlay              `toml:"skills"`
	Extensions    extensionsOverlay          `toml:"extensions"`
	Tools         toolsOverlay               `toml:"tools"`
	Automation    automationOverlay          `toml:"automation"`
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
	MaxSessions         *int `toml:"max_sessions"`
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
	DefaultModel    *string                     `toml:"default_model"`
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
	Enabled   *bool        `toml:"enabled"`
	GlobalDir *string      `toml:"global_dir"`
	Dream     dreamOverlay `toml:"dream"`
}

type dreamOverlay struct {
	Enabled       *bool          `toml:"enabled"`
	Agent         *string        `toml:"agent"`
	MinHours      *float64       `toml:"min_hours"`
	MinSessions   *int           `toml:"min_sessions"`
	CheckInterval *time.Duration `toml:"check_interval"`
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
		return configOverlay{}, FileError{Op: "read", Path: path, Err: err}
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
		return overlay, fmt.Errorf("unknown config keys in %q: %s", source, joinTOMLKeys(undecoded))
	}

	return overlay, nil
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
	applySandboxOverlays(dst, o.Sandboxes)
	o.Observability.Apply(&dst.Observability)
	o.Log.Apply(&dst.Log)
	o.Memory.Apply(&dst.Memory)
	inheritDreamAgentFromDefaultAgent(dst, o)
	o.Skills.Apply(&dst.Skills)
	o.Extensions.Apply(&dst.Extensions)
	o.Tools.Apply(&dst.Tools)
	if err := o.Automation.Apply(&dst.Automation); err != nil {
		return err
	}
	o.Network.Apply(&dst.Network)
	o.Autonomy.Apply(&dst.Autonomy)
	return o.Hooks.Apply(&dst.Hooks)
}

func inheritDreamAgentFromDefaultAgent(dst *Config, overlay *configOverlay) {
	if dst == nil || overlay == nil || overlay.Defaults.Agent == nil || overlay.Memory.Dream.Agent != nil {
		return
	}
	defaultAgent := strings.TrimSpace(dst.Defaults.Agent)
	if defaultAgent == "" || defaultAgent == DefaultAgentName {
		return
	}
	currentDreamAgent := strings.TrimSpace(dst.Memory.Dream.Agent)
	if currentDreamAgent == "" || currentDreamAgent == DefaultAgentName {
		dst.Memory.Dream.Agent = defaultAgent
	}
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
	if o.MaxSessions != nil {
		dst.MaxSessions = *o.MaxSessions
	}
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
	if o.DefaultModel != nil {
		dst.DefaultModel = *o.DefaultModel
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
		dst.SessionMCP = boolRef(*o.SessionMCP)
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

func (o memoryOverlay) Apply(dst *MemoryConfig) {
	if o.Enabled != nil {
		dst.Enabled = *o.Enabled
	}
	if o.GlobalDir != nil && strings.TrimSpace(*o.GlobalDir) != "" {
		dst.GlobalDir = *o.GlobalDir
	}
	o.Dream.Apply(&dst.Dream)
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
	if o.CheckInterval != nil {
		dst.CheckInterval = *o.CheckInterval
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

	for idx, raw := range o.Declarations {
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
	index := make(map[string]int, len(merged))
	for i, server := range merged {
		if server.Name == "" {
			continue
		}
		index[server.Name] = i
	}

	for _, overlay := range overlays {
		name := ""
		if overlay.Name != nil {
			name = strings.TrimSpace(*overlay.Name)
		}

		if idx, ok := index[name]; ok && name != "" {
			server := merged[idx]
			overlay.Apply(&server)
			merged[idx] = server
			continue
		}

		var server MCPServer
		overlay.Apply(&server)
		merged = append(merged, server)
		if server.Name != "" {
			index[server.Name] = len(merged) - 1
		}
	}

	return merged
}

func joinTOMLKeys(keys []burnttoml.Key) string {
	if len(keys) == 0 {
		return ""
	}

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key.String())
	}
	sort.Strings(values)

	return strings.Join(values, ", ")
}
