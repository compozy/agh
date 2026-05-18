// Package settings provides the daemon-facing settings orchestration service.
package settings

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/resources"
)

// ScopeKind identifies the supported settings scope.
type ScopeKind string

const (
	// ScopeGlobal selects the global AGH home scope.
	ScopeGlobal ScopeKind = "global"
	// ScopeWorkspace selects one workspace-local overlay scope.
	ScopeWorkspace ScopeKind = "workspace"
	// ScopeAgent selects one effective agent-local overlay scope.
	ScopeAgent ScopeKind = "agent"
)

// Validate ensures the requested settings scope is supported.
func (s ScopeKind) Validate() error {
	switch s {
	case ScopeGlobal, ScopeWorkspace, ScopeAgent:
		return nil
	default:
		return fmt.Errorf("settings: invalid scope %q", s)
	}
}

func (s ScopeKind) configWriteScope() aghconfig.WriteScope {
	if s == ScopeWorkspace {
		return aghconfig.WriteScopeWorkspace
	}
	return aghconfig.WriteScopeGlobal
}

// WriteTargetKind identifies the semantic persistence target for one mutation.
type WriteTargetKind = aghconfig.WriteTargetKind

const (
	// WriteTargetGlobalConfig persists to `~/.agh/config.toml`.
	WriteTargetGlobalConfig = aghconfig.WriteTargetGlobalConfig
	// WriteTargetWorkspaceConfig persists to `<workspace>/.agh/config.toml`.
	WriteTargetWorkspaceConfig = aghconfig.WriteTargetWorkspaceConfig
	// WriteTargetGlobalMCPSidecar persists to `~/.agh/mcp.json`.
	WriteTargetGlobalMCPSidecar = aghconfig.WriteTargetGlobalMCPSidecar
	// WriteTargetWorkspaceMCPSidecar persists to `<workspace>/.agh/mcp.json`.
	WriteTargetWorkspaceMCPSidecar = aghconfig.WriteTargetWorkspaceMCPSidecar
	// WriteTargetGlobalAgentFile persists to `~/.agh/agents/<name>/AGENT.md`.
	WriteTargetGlobalAgentFile WriteTargetKind = "global-agent-file"
	// WriteTargetWorkspaceAgentFile persists to `<root>/.agh/agents/<name>/AGENT.md`.
	WriteTargetWorkspaceAgentFile WriteTargetKind = "workspace-agent-file"
)

// SectionName names one section-oriented settings resource.
type SectionName string

const (
	// SectionGeneral exposes daemon-wide runtime and config defaults.
	SectionGeneral SectionName = "general"
	// SectionMemory exposes memory and dream settings.
	SectionMemory SectionName = "memory"
	// SectionSkills exposes global skills-engine settings.
	SectionSkills SectionName = "skills"
	// SectionAutomation exposes automation engine settings.
	SectionAutomation SectionName = "automation"
	// SectionNetwork exposes embedded network settings.
	SectionNetwork SectionName = "network"
	// SectionObservability exposes observe and transcript settings.
	SectionObservability SectionName = "observability"
	// SectionHooksExtensions exposes hook declarations plus extension policy.
	SectionHooksExtensions SectionName = "hooks-extensions"
)

// CollectionName names one collection-oriented settings resource.
type CollectionName string

const (
	// CollectionProviders exposes the provider catalog.
	CollectionProviders CollectionName = "providers"
	// CollectionMCPServers exposes the scoped MCP server catalog.
	CollectionMCPServers CollectionName = "mcp-servers"
	// CollectionSandboxes exposes execution sandboxes.
	CollectionSandboxes CollectionName = "sandboxes"
	// CollectionHooks exposes config-defined hook declarations.
	CollectionHooks CollectionName = "hooks"
)

// TargetSelector selects one MCP persistence destination.
type TargetSelector string

const (
	// TargetAuto edits the highest-precedence definition in the selected scope.
	TargetAuto TargetSelector = "auto"
	// TargetConfig edits the TOML-backed source in the selected scope.
	TargetConfig TargetSelector = "config"
	// TargetSidecar edits the sidecar-backed source in the selected scope.
	TargetSidecar TargetSelector = "sidecar"
)

// Normalize returns the canonical selector, defaulting an omitted selector to auto.
func (s TargetSelector) Normalize() TargetSelector {
	trimmed := TargetSelector(strings.TrimSpace(string(s)))
	if trimmed == "" {
		return TargetAuto
	}
	return trimmed
}

// Validate reports malformed MCP target selectors instead of silently selecting auto.
func (s TargetSelector) Validate() error {
	normalized := s.Normalize()
	switch normalized {
	case TargetAuto, TargetConfig, TargetSidecar:
		return nil
	default:
		return validationError(fmt.Errorf("settings: unsupported MCP target selector %q", normalized))
	}
}

// MutationBehavior classifies how a mutation takes effect at runtime.
type MutationBehavior string

const (
	// MutationBehaviorAppliedNow reports a live-applied mutation.
	MutationBehaviorAppliedNow MutationBehavior = "applied_now"
	// MutationBehaviorRestartRequired reports a persisted mutation that needs restart.
	MutationBehaviorRestartRequired MutationBehavior = "restart_required"
	// MutationBehaviorActionTrigger reports a mutation that triggers an action.
	MutationBehaviorActionTrigger MutationBehavior = "action_trigger"
)

// SourceKind identifies one semantic resource source.
type SourceKind string

const (
	// SourceKindBuiltinProvider identifies the builtin provider registry.
	SourceKindBuiltinProvider SourceKind = "builtin-provider"
	// SourceKindGlobalConfig identifies the global TOML config.
	SourceKindGlobalConfig SourceKind = "global-config"
	// SourceKindWorkspaceConfig identifies the workspace TOML config.
	SourceKindWorkspaceConfig SourceKind = "workspace-config"
	// SourceKindGlobalMCPSidecar identifies the global MCP JSON sidecar.
	SourceKindGlobalMCPSidecar SourceKind = "global-mcp-sidecar"
	// SourceKindWorkspaceMCPSidecar identifies the workspace MCP JSON sidecar.
	SourceKindWorkspaceMCPSidecar SourceKind = "workspace-mcp-sidecar"
	// SourceKindGlobalAgentFile identifies a global AGENT.md frontmatter source.
	SourceKindGlobalAgentFile SourceKind = "global-agent-file"
	// SourceKindWorkspaceAgentFile identifies a workspace/additional AGENT.md frontmatter source.
	SourceKindWorkspaceAgentFile SourceKind = "workspace-agent-file"
)

// Service is the daemon-facing settings orchestration boundary.
type Service interface {
	GetSection(ctx context.Context, req SectionRequest) (SectionEnvelope, error)
	UpdateSection(ctx context.Context, req SectionUpdateRequest) (MutationResult, error)
	ListCollection(ctx context.Context, req CollectionRequest) (CollectionEnvelope, error)
	PutCollectionItem(ctx context.Context, req CollectionItemPutRequest) (MutationResult, error)
	DeleteCollectionItem(ctx context.Context, req CollectionItemDeleteRequest) (MutationResult, error)
}

// SectionRequest identifies one section read.
type SectionRequest struct {
	Section     SectionName
	Scope       ScopeKind
	WorkspaceID string
	AgentName   string
}

// SectionUpdateRequest identifies one section mutation.
type SectionUpdateRequest struct {
	SectionRequest
	General         *GeneralSettings
	Memory          *aghconfig.MemoryConfig
	Skills          *aghconfig.SkillsConfig
	Automation      *AutomationSettings
	Network         *aghconfig.NetworkConfig
	Observability   *aghconfig.ObservabilityConfig
	HooksExtensions *aghconfig.ExtensionsConfig
}

// CollectionRequest identifies one collection read.
type CollectionRequest struct {
	Collection  CollectionName
	Scope       ScopeKind
	WorkspaceID string
}

// CollectionItemPutRequest identifies one collection upsert.
type CollectionItemPutRequest struct {
	CollectionRequest
	Name            string
	Target          TargetSelector
	Provider        *ProviderSettings
	ProviderSecrets []ProviderSecretWrite
	MCPServer       *aghconfig.MCPServer
	MCPSecrets      MCPSecretValues
	Sandbox         *aghconfig.SandboxProfile
	Hook            *hookspkg.HookDecl
}

// CollectionItemDeleteRequest identifies one collection delete.
type CollectionItemDeleteRequest struct {
	CollectionRequest
	Name   string
	Target TargetSelector
}

// SectionEnvelope returns one typed section payload.
type SectionEnvelope struct {
	Section         SectionName
	Scope           ScopeKind
	WorkspaceID     string
	AgentName       string
	AvailableScopes []ScopeKind
	General         *GeneralSection
	Memory          *MemorySection
	Skills          *SkillsSection
	Automation      *AutomationSection
	Network         *NetworkSection
	Observability   *ObservabilitySection
	HooksExtensions *HooksExtensionsSection
}

// CollectionEnvelope returns one typed collection payload.
type CollectionEnvelope struct {
	Collection      CollectionName
	Scope           ScopeKind
	WorkspaceID     string
	AvailableScopes []ScopeKind
	Providers       []ProviderItem
	MCPServers      []MCPServerItem
	Sandboxes       []SandboxItem
	Hooks           []HookItem
}

// MutationResult reports the semantic outcome of one settings mutation.
type MutationResult struct {
	Section         SectionName      `json:"section"`
	Scope           ScopeKind        `json:"scope"`
	WriteTarget     WriteTargetKind  `json:"write_target,omitempty"`
	WorkspaceID     string           `json:"workspace_id,omitempty"`
	AgentName       string           `json:"agent_name,omitempty"`
	Behavior        MutationBehavior `json:"behavior"`
	Applied         bool             `json:"applied"`
	RestartRequired bool             `json:"restart_required"`
	RestartScope    string           `json:"restart_scope,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
}

// MutationDescriptor identifies the changed fields or action behind one mutation.
type MutationDescriptor struct {
	Section       SectionName
	ChangedFields []string
	Action        string
}

// MutationClassification reports the classified runtime behavior for one mutation.
type MutationClassification struct {
	Behavior        MutationBehavior
	Applied         bool
	RestartRequired bool
	RestartScope    string
}

// GeneralSettings groups the editable general section config.
type GeneralSettings struct {
	Defaults       aghconfig.DefaultsConfig
	Limits         aghconfig.LimitsConfig
	Permissions    aghconfig.PermissionsConfig
	SessionTimeout time.Duration
	HTTP           aghconfig.HTTPConfig
	Daemon         aghconfig.DaemonConfig
}

// AutomationSettings groups the editable automation-engine settings.
type AutomationSettings struct {
	Enabled           bool
	Timezone          string
	MaxConcurrentJobs int
	DefaultFireLimit  automationmodel.FireLimitConfig
}

// GeneralSection is the general section read model.
type GeneralSection struct {
	Runtime     DaemonRuntimeStatus
	ConfigPaths ConfigPaths
	Settings    GeneralSettings
	Actions     GeneralActions
}

// MemorySection is the memory section read model.
type MemorySection struct {
	Config  aghconfig.MemoryConfig
	Health  MemoryHealthStatus
	Actions MemoryActions
}

// SkillsSection is the skills section read model.
type SkillsSection struct {
	Config           aghconfig.SkillsConfig
	DiscoveredCount  int
	DisabledCount    int
	RuntimeAvailable bool
	Links            []OperationalLink
}

// AutomationSection is the automation section read model.
type AutomationSection struct {
	Config  AutomationSettings
	Runtime AutomationRuntimeStatus
	Links   []OperationalLink
}

// NetworkSection is the network section read model.
type NetworkSection struct {
	Config  aghconfig.NetworkConfig
	Runtime NetworkRuntimeStatus
	Links   []OperationalLink
}

// ObservabilitySection is the observability section read model.
type ObservabilitySection struct {
	Config         aghconfig.ObservabilityConfig
	Runtime        ObservabilityRuntimeStatus
	LogTailSupport CapabilityStatus
}

// HooksExtensionsSection is the hooks and extensions section read model.
type HooksExtensionsSection struct {
	Hooks           []HookItem
	Extensions      aghconfig.ExtensionsConfig
	Installed       []InstalledExtension
	TransportParity TransportParityStatus
}

// ConfigPaths exposes resolved daemon config file locations.
type ConfigPaths struct {
	HomeDir          string
	GlobalConfig     string
	GlobalMCPSidecar string
	LogFile          string
	DaemonInfo       string
}

// DaemonRuntimeStatus summarizes daemon-local runtime state.
type DaemonRuntimeStatus struct {
	Available      bool
	Status         string
	PID            int
	StartedAt      time.Time
	UptimeSeconds  int64
	Socket         string
	HTTPHost       string
	HTTPPort       int
	ActiveSessions int
	ActiveAgents   int
	TotalSessions  int
	Version        string
}

// MemoryHealthStatus summarizes memory runtime state.
type MemoryHealthStatus struct {
	Available          bool
	FileCount          int
	DreamEnabled       bool
	LastConsolidatedAt *time.Time
}

// AutomationRuntimeStatus summarizes automation runtime state.
type AutomationRuntimeStatus struct {
	Available        bool
	Running          bool
	SchedulerRunning bool
	JobTotal         int
	JobEnabled       int
	TriggerTotal     int
	TriggerEnabled   int
	NextFire         *time.Time
	LastSyncedAt     *time.Time
}

// NetworkRuntimeStatus summarizes network runtime state.
type NetworkRuntimeStatus struct {
	Available       bool
	Enabled         bool
	Status          string
	ListenerHost    string
	ListenerPort    int
	LocalPeers      int
	RemotePeers     int
	Channels        int
	QueuedMessages  int
	QueuedSessions  int
	DeliveryWorkers int
}

// ObservabilityRuntimeStatus summarizes observability runtime state.
type ObservabilityRuntimeStatus struct {
	Available          bool
	Status             string
	GlobalDBSizeBytes  int64
	SessionDBSizeBytes int64
	ActiveSessions     int
	ActiveAgents       int
	UptimeSeconds      int64
}

// CapabilityStatus reports whether one auxiliary feature is available.
type CapabilityStatus struct {
	Available bool
}

// GeneralActions reports general-section action metadata.
type GeneralActions struct {
	Restart ActionMetadata
}

// MemoryActions reports memory-section action metadata.
type MemoryActions struct {
	Consolidate ActionMetadata
}

// ActionMetadata reports one action trigger's semantic behavior.
type ActionMetadata struct {
	Name      string
	Available bool
	Behavior  MutationBehavior
}

// OperationalLink reports one related operational destination.
type OperationalLink struct {
	Label string
	Path  string
}

// TransportParityStatus reports whether required settings transports are present.
type TransportParityStatus struct {
	Known          bool
	SettingsHTTP   bool
	SettingsUDS    bool
	ExtensionsHTTP bool
	ExtensionsUDS  bool
}

// InstalledExtension summarizes one installed extension for settings surfaces.
type InstalledExtension struct {
	Name          string
	Version       string
	Enabled       bool
	State         string
	Health        string
	HealthMessage string
	LastError     string
	RequiresEnv   []string
	MissingEnv    []string
}

// SourceRef identifies one semantic source for a resolved resource.
type SourceRef struct {
	Kind        SourceKind
	Scope       ScopeKind
	WorkspaceID string
	AgentName   string
}

// SourceMetadata reports precedence and target information for one resource.
type SourceMetadata struct {
	EffectiveSource  SourceRef
	ShadowedSources  []SourceRef
	AvailableTargets []WriteTargetKind
}

// ProviderSettings is the editable provider overlay payload.
type ProviderSettings struct {
	Command         string
	DisplayName     string
	Models          aghconfig.ProviderModelsConfig
	ModelsSet       bool
	Harness         aghconfig.ProviderHarness
	RuntimeProvider string
	Transport       string
	BaseURL         string
	AuthMode        aghconfig.ProviderAuthMode
	EnvPolicy       aghconfig.ProviderEnvPolicy
	HomePolicy      aghconfig.ProviderHomePolicy
	AuthStatusCmd   string
	AuthLoginCmd    string
	CredentialSlots []aghconfig.ProviderCredentialSlot
}

// ProviderCredentialStatus is a redacted launch credential status.
type ProviderCredentialStatus struct {
	Name      string
	TargetEnv string
	SecretRef string
	Kind      string
	Required  bool
	Present   bool
	Source    string
}

// ProviderAuthStatus is a redacted provider authentication readiness summary.
type ProviderAuthStatus struct {
	Mode       aghconfig.ProviderAuthMode
	EnvPolicy  aghconfig.ProviderEnvPolicy
	HomePolicy aghconfig.ProviderHomePolicy
	State      string
	Message    string
	StatusCmd  string
	LoginCmd   string
}

// ProviderSecretWrite is one write-only provider secret mutation.
type ProviderSecretWrite struct {
	Name      string
	SecretRef string
	Kind      string
	Value     string
}

// MCPSecretValues is the write-only secret material submitted with an MCP server mutation.
type MCPSecretValues struct {
	SecretEnv         map[string]string
	OAuthClientSecret *string
}

func (v MCPSecretValues) Empty() bool {
	return len(v.SecretEnv) == 0 && v.OAuthClientSecret == nil
}

// MCPAuthStatus is a redacted remote MCP authentication status.
type MCPAuthStatus = mcpauth.Status

// ProviderFallback reports the builtin provider revealed when an overlay is removed.
type ProviderFallback struct {
	Source   SourceRef
	Settings ProviderSettings
}

// ProviderItem is one provider collection row.
type ProviderItem struct {
	Name             string
	Settings         ProviderSettings
	Default          bool
	CommandAvailable bool
	Credentials      []ProviderCredentialStatus
	AuthStatus       ProviderAuthStatus
	SourceMetadata   SourceMetadata
	Fallback         *ProviderFallback
}

// MCPServerItem is one MCP server collection row.
type MCPServerItem struct {
	Name           string
	Transport      aghconfig.MCPServerTransport
	Command        string
	Args           []string
	Env            map[string]string
	SecretEnv      map[string]string
	URL            string
	Auth           aghconfig.MCPAuthConfig
	AuthStatus     *mcpauth.Status
	Scope          ScopeKind
	WorkspaceID    string
	SourceMetadata SourceMetadata
}

// SandboxItem is one sandbox collection row.
type SandboxItem struct {
	Name                string
	Profile             aghconfig.SandboxProfile
	WorkspaceUsageCount int
	SourceMetadata      SourceMetadata
}

// HookItem is one config-defined hook collection row.
type HookItem struct {
	Name           string
	Declaration    hookspkg.HookDecl
	SourceMetadata SourceMetadata
}

func builtinProviderSource() SourceRef {
	return SourceRef{Kind: SourceKindBuiltinProvider, Scope: ScopeGlobal}
}

func sourceRefForWriteTarget(kind WriteTargetKind, workspaceID string, agentName string) SourceRef {
	switch kind {
	case WriteTargetGlobalConfig:
		return SourceRef{Kind: SourceKindGlobalConfig, Scope: ScopeGlobal}
	case WriteTargetWorkspaceConfig:
		return SourceRef{Kind: SourceKindWorkspaceConfig, Scope: ScopeWorkspace, WorkspaceID: workspaceID}
	case WriteTargetGlobalMCPSidecar:
		return SourceRef{Kind: SourceKindGlobalMCPSidecar, Scope: ScopeGlobal}
	case WriteTargetWorkspaceMCPSidecar:
		return SourceRef{Kind: SourceKindWorkspaceMCPSidecar, Scope: ScopeWorkspace, WorkspaceID: workspaceID}
	case WriteTargetGlobalAgentFile:
		return SourceRef{Kind: SourceKindGlobalAgentFile, Scope: ScopeAgent, AgentName: agentName}
	case WriteTargetWorkspaceAgentFile:
		return SourceRef{
			Kind:        SourceKindWorkspaceAgentFile,
			Scope:       ScopeAgent,
			WorkspaceID: workspaceID,
			AgentName:   agentName,
		}
	default:
		return SourceRef{}
	}
}

func availableTargetsForScope(scope ScopeKind) []WriteTargetKind {
	switch scope {
	case ScopeWorkspace:
		return []WriteTargetKind{WriteTargetWorkspaceConfig, WriteTargetWorkspaceMCPSidecar}
	default:
		return []WriteTargetKind{WriteTargetGlobalConfig, WriteTargetGlobalMCPSidecar}
	}
}

func singleTargetSourceMetadata(kind WriteTargetKind, workspaceID string) SourceMetadata {
	return SourceMetadata{
		EffectiveSource:  sourceRefForWriteTarget(kind, workspaceID, ""),
		AvailableTargets: []WriteTargetKind{kind},
	}
}

func globalConfigSourceMetadata() SourceMetadata {
	return singleTargetSourceMetadata(WriteTargetGlobalConfig, "")
}

func cloneSourceRefs(values []SourceRef) []SourceRef {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]SourceRef, len(values))
	copy(cloned, values)
	return cloned
}

func cloneSourceMetadata(value SourceMetadata) SourceMetadata {
	value.ShadowedSources = cloneSourceRefs(value.ShadowedSources)
	value.AvailableTargets = append([]WriteTargetKind(nil), value.AvailableTargets...)
	return value
}

func cloneProviderSettings(value ProviderSettings) ProviderSettings {
	value.Models = cloneProviderModelsConfig(value.Models)
	value.CredentialSlots = append([]aghconfig.ProviderCredentialSlot(nil), value.CredentialSlots...)
	return value
}

func cloneProviderModelsConfig(value aghconfig.ProviderModelsConfig) aghconfig.ProviderModelsConfig {
	return aghconfig.ProviderModelsConfig{
		Default:   value.Default,
		Curated:   cloneProviderModelConfigs(value.Curated),
		Discovery: cloneProviderModelsDiscoveryConfig(value.Discovery),
	}
}

func cloneProviderModelsDiscoveryConfig(
	value aghconfig.ProviderModelsDiscoveryConfig,
) aghconfig.ProviderModelsDiscoveryConfig {
	return aghconfig.ProviderModelsDiscoveryConfig{
		Enabled:  cloneBoolPtr(value.Enabled),
		Command:  value.Command,
		Endpoint: value.Endpoint,
		Timeout:  value.Timeout,
	}
}

func cloneProviderModelConfigs(values []aghconfig.ProviderModelConfig) []aghconfig.ProviderModelConfig {
	if values == nil {
		return nil
	}
	cloned := make([]aghconfig.ProviderModelConfig, len(values))
	for idx, value := range values {
		cloned[idx] = aghconfig.ProviderModelConfig{
			ID:                     value.ID,
			DisplayName:            value.DisplayName,
			ContextWindow:          cloneInt64Ptr(value.ContextWindow),
			MaxInputTokens:         cloneInt64Ptr(value.MaxInputTokens),
			MaxOutputTokens:        cloneInt64Ptr(value.MaxOutputTokens),
			SupportsTools:          cloneBoolPtr(value.SupportsTools),
			SupportsReasoning:      cloneBoolPtr(value.SupportsReasoning),
			ReasoningEfforts:       cloneStringSlicePreserveNil(value.ReasoningEfforts),
			DefaultReasoningEffort: value.DefaultReasoningEffort,
			CostInputPerMillion:    cloneFloat64Ptr(value.CostInputPerMillion),
			CostOutputPerMillion:   cloneFloat64Ptr(value.CostOutputPerMillion),
		}
	}
	return cloned
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneStringSlicePreserveNil(value []string) []string {
	if value == nil {
		return nil
	}
	cloned := make([]string, len(value))
	copy(cloned, value)
	return cloned
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneProviderItem(value *ProviderItem) ProviderItem {
	if value == nil {
		return ProviderItem{}
	}
	cloned := *value
	cloned.Settings = cloneProviderSettings(value.Settings)
	cloned.Credentials = append([]ProviderCredentialStatus(nil), value.Credentials...)
	cloned.SourceMetadata = cloneSourceMetadata(value.SourceMetadata)
	if value.Fallback != nil {
		fallback := *value.Fallback
		fallback.Settings = cloneProviderSettings(fallback.Settings)
		cloned.Fallback = &fallback
	}
	return cloned
}

func cloneMCPServerItem(value MCPServerItem) MCPServerItem {
	value.Args = append([]string(nil), value.Args...)
	if len(value.Env) > 0 {
		cloned := make(map[string]string, len(value.Env))
		maps.Copy(cloned, value.Env)
		value.Env = cloned
	}
	if len(value.SecretEnv) > 0 {
		cloned := make(map[string]string, len(value.SecretEnv))
		maps.Copy(cloned, value.SecretEnv)
		value.SecretEnv = cloned
	}
	value.Auth.Scopes = append([]string(nil), value.Auth.Scopes...)
	if value.AuthStatus != nil {
		status := *value.AuthStatus
		status.Scopes = append([]string(nil), value.AuthStatus.Scopes...)
		if value.AuthStatus.ExpiresAt != nil {
			expiresAt := *value.AuthStatus.ExpiresAt
			status.ExpiresAt = &expiresAt
		}
		if value.AuthStatus.UpdatedAt != nil {
			updatedAt := *value.AuthStatus.UpdatedAt
			status.UpdatedAt = &updatedAt
		}
		value.AuthStatus = &status
	}
	value.SourceMetadata = cloneSourceMetadata(value.SourceMetadata)
	return value
}

func cloneSandboxItem(value SandboxItem) SandboxItem {
	value.Profile = aghconfig.SandboxProfile{
		Backend:     value.Profile.Backend,
		SyncMode:    value.Profile.SyncMode,
		Persistence: value.Profile.Persistence,
		RuntimeRoot: value.Profile.RuntimeRoot,
		Env:         cloneStringMap(value.Profile.Env),
		SecretEnv:   cloneStringMap(value.Profile.SecretEnv),
		Network: aghconfig.NetworkProfile{
			AllowPublicIngress: value.Profile.Network.AllowPublicIngress,
			AllowOutbound:      value.Profile.Network.AllowOutbound,
			AllowList:          append([]string(nil), value.Profile.Network.AllowList...),
			DenyList:           append([]string(nil), value.Profile.Network.DenyList...),
			Required:           value.Profile.Network.Required,
		},
		Daytona: aghconfig.DaytonaProfile{
			APIURL:      value.Profile.Daytona.APIURL,
			Target:      value.Profile.Daytona.Target,
			Image:       value.Profile.Daytona.Image,
			Snapshot:    value.Profile.Daytona.Snapshot,
			Class:       value.Profile.Daytona.Class,
			AutoStop:    value.Profile.Daytona.AutoStop,
			AutoArchive: value.Profile.Daytona.AutoArchive,
		},
	}
	value.SourceMetadata = cloneSourceMetadata(value.SourceMetadata)
	return value
}

func cloneHookItem(value *HookItem) HookItem {
	cloned := *value
	cloned.SourceMetadata = cloneSourceMetadata(cloned.SourceMetadata)
	cloned.Declaration = cloneHookDecl(cloned.Declaration)
	return cloned
}

func cloneHookDecl(value hookspkg.HookDecl) hookspkg.HookDecl {
	cloned := value
	cloned.Args = append([]string(nil), value.Args...)
	cloned.Env = cloneStringMap(value.Env)
	cloned.SecretEnv = cloneStringMap(value.SecretEnv)
	cloned.Metadata = cloneStringMap(value.Metadata)
	if value.Matcher.ToolReadOnly != nil {
		toolReadOnly := *value.Matcher.ToolReadOnly
		cloned.Matcher.ToolReadOnly = &toolReadOnly
	}
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	maps.Copy(cloned, values)
	return cloned
}

func cloneAllowedKinds(values []resources.ResourceKind) []resources.ResourceKind {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]resources.ResourceKind, len(values))
	copy(cloned, values)
	return cloned
}
