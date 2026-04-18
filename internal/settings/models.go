// Package settings provides the daemon-facing settings orchestration service.
package settings

import (
	"context"
	"maps"
	"time"

	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
)

// ScopeKind identifies the supported settings scope.
type ScopeKind = aghconfig.WriteScope

const (
	// ScopeGlobal selects the global AGH home scope.
	ScopeGlobal = aghconfig.WriteScopeGlobal
	// ScopeWorkspace selects one workspace-local overlay scope.
	ScopeWorkspace = aghconfig.WriteScopeWorkspace
)

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
	// CollectionEnvironments exposes execution environments.
	CollectionEnvironments CollectionName = "environments"
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
	Name        string
	Target      TargetSelector
	Provider    *ProviderSettings
	MCPServer   *aghconfig.MCPServer
	Environment *aghconfig.EnvironmentProfile
	Hook        *hookspkg.HookDecl
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
	Environments    []EnvironmentItem
	Hooks           []HookItem
}

// MutationResult reports the semantic outcome of one settings mutation.
type MutationResult struct {
	Section         SectionName      `json:"section"`
	Scope           ScopeKind        `json:"scope"`
	WriteTarget     WriteTargetKind  `json:"write_target,omitempty"`
	WorkspaceID     string           `json:"workspace_id,omitempty"`
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
}

// SourceRef identifies one semantic source for a resolved resource.
type SourceRef struct {
	Kind        SourceKind
	Scope       ScopeKind
	WorkspaceID string
}

// SourceMetadata reports precedence and target information for one resource.
type SourceMetadata struct {
	EffectiveSource  SourceRef
	ShadowedSources  []SourceRef
	AvailableTargets []WriteTargetKind
}

// ProviderSettings is the editable provider overlay payload.
type ProviderSettings struct {
	Command      string
	DefaultModel string
	APIKeyEnv    string
}

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
	APIKeyEnvPresent bool
	SourceMetadata   SourceMetadata
	Fallback         *ProviderFallback
}

// MCPServerItem is one MCP server collection row.
type MCPServerItem struct {
	Name           string
	Command        string
	Args           []string
	Env            map[string]string
	Scope          ScopeKind
	WorkspaceID    string
	SourceMetadata SourceMetadata
}

// EnvironmentItem is one environment collection row.
type EnvironmentItem struct {
	Name                string
	Profile             aghconfig.EnvironmentProfile
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

func sourceRefForWriteTarget(kind WriteTargetKind, workspaceID string) SourceRef {
	switch kind {
	case WriteTargetGlobalConfig:
		return SourceRef{Kind: SourceKindGlobalConfig, Scope: ScopeGlobal}
	case WriteTargetWorkspaceConfig:
		return SourceRef{Kind: SourceKindWorkspaceConfig, Scope: ScopeWorkspace, WorkspaceID: workspaceID}
	case WriteTargetGlobalMCPSidecar:
		return SourceRef{Kind: SourceKindGlobalMCPSidecar, Scope: ScopeGlobal}
	case WriteTargetWorkspaceMCPSidecar:
		return SourceRef{Kind: SourceKindWorkspaceMCPSidecar, Scope: ScopeWorkspace, WorkspaceID: workspaceID}
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
		EffectiveSource:  sourceRefForWriteTarget(kind, workspaceID),
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
	return value
}

func cloneProviderItem(value ProviderItem) ProviderItem {
	value.SourceMetadata = cloneSourceMetadata(value.SourceMetadata)
	if value.Fallback != nil {
		fallback := *value.Fallback
		fallback.Settings = cloneProviderSettings(fallback.Settings)
		value.Fallback = &fallback
	}
	return value
}

func cloneMCPServerItem(value MCPServerItem) MCPServerItem {
	value.Args = append([]string(nil), value.Args...)
	if len(value.Env) > 0 {
		cloned := make(map[string]string, len(value.Env))
		maps.Copy(cloned, value.Env)
		value.Env = cloned
	}
	value.SourceMetadata = cloneSourceMetadata(value.SourceMetadata)
	return value
}

func cloneEnvironmentItem(value EnvironmentItem) EnvironmentItem {
	value.Profile = aghconfig.EnvironmentProfile{
		Backend:     value.Profile.Backend,
		SyncMode:    value.Profile.SyncMode,
		Persistence: value.Profile.Persistence,
		RuntimeRoot: value.Profile.RuntimeRoot,
		Env:         cloneStringMap(value.Profile.Env),
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
