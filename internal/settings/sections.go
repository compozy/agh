package settings

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/resources"
)

func (s *service) GetSection(ctx context.Context, req SectionRequest) (SectionEnvelope, error) {
	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return SectionEnvelope{}, fmt.Errorf("settings: get section %q: %w", req.Section, err)
	}
	if scope != ScopeGlobal {
		return SectionEnvelope{}, conflictError(
			fmt.Errorf("settings: section %q does not support workspace scope", req.Section),
		)
	}

	cfg, _, err := s.loadConfig(ctx, scope, workspaceID)
	if err != nil {
		return SectionEnvelope{}, fmt.Errorf("settings: load section %q config: %w", req.Section, err)
	}

	envelope := SectionEnvelope{
		Section:         req.Section,
		Scope:           ScopeGlobal,
		AvailableScopes: []ScopeKind{ScopeGlobal},
	}

	switch req.Section {
	case SectionGeneral:
		section, sectionErr := s.buildGeneralSection(ctx, &cfg)
		if sectionErr != nil {
			return SectionEnvelope{}, sectionErr
		}
		envelope.General = &section
	case SectionMemory:
		section, sectionErr := s.buildMemorySection(ctx, &cfg)
		if sectionErr != nil {
			return SectionEnvelope{}, sectionErr
		}
		envelope.Memory = &section
	case SectionSkills:
		section := s.buildSkillsSection(&cfg)
		envelope.Skills = &section
	case SectionAutomation:
		section, sectionErr := s.buildAutomationSection(ctx, &cfg)
		if sectionErr != nil {
			return SectionEnvelope{}, sectionErr
		}
		envelope.Automation = &section
	case SectionNetwork:
		section, sectionErr := s.buildNetworkSection(ctx, &cfg)
		if sectionErr != nil {
			return SectionEnvelope{}, sectionErr
		}
		envelope.Network = &section
	case SectionObservability:
		section, sectionErr := s.buildObservabilitySection(ctx, &cfg)
		if sectionErr != nil {
			return SectionEnvelope{}, sectionErr
		}
		envelope.Observability = &section
	case SectionHooksExtensions:
		section, sectionErr := s.buildHooksExtensionsSection(ctx, &cfg)
		if sectionErr != nil {
			return SectionEnvelope{}, sectionErr
		}
		envelope.HooksExtensions = &section
	default:
		return SectionEnvelope{}, notFoundError(fmt.Errorf("settings: unknown section %q", req.Section))
	}

	return envelope, nil
}

func (s *service) UpdateSection(ctx context.Context, req SectionUpdateRequest) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}

	switch req.Section {
	case SectionGeneral:
		if req.General == nil {
			return MutationResult{}, validationError(errors.New("settings: general section payload is required"))
		}
		changed := diffGeneralSettings(&cfg, *req.General)
		return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
			return applyGeneralSettings(editor, *req.General)
		})
	case SectionMemory:
		if req.Memory == nil {
			return MutationResult{}, validationError(errors.New("settings: memory section payload is required"))
		}
		changed := diffMemorySettings(cfg.Memory, *req.Memory)
		return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
			return applyMemorySettings(editor, *req.Memory)
		})
	case SectionSkills:
		if req.Skills == nil {
			return MutationResult{}, validationError(errors.New("settings: skills section payload is required"))
		}
		changed := diffSkillsSettings(cfg.Skills, *req.Skills)
		return s.updateSkillsSection(ctx, cfg.Skills, *req.Skills, changed, target)
	case SectionAutomation:
		if req.Automation == nil {
			return MutationResult{}, validationError(errors.New("settings: automation section payload is required"))
		}
		changed := diffAutomationSettings(&cfg, *req.Automation)
		return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
			return applyAutomationSettings(editor, *req.Automation)
		})
	case SectionNetwork:
		if req.Network == nil {
			return MutationResult{}, validationError(errors.New("settings: network section payload is required"))
		}
		changed := diffNetworkSettings(cfg.Network, *req.Network)
		return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
			return applyNetworkSettings(editor, *req.Network)
		})
	case SectionObservability:
		if req.Observability == nil {
			return MutationResult{}, validationError(
				errors.New("settings: observability section payload is required"),
			)
		}
		changed := diffObservabilitySettings(cfg.Observability, *req.Observability)
		return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
			return applyObservabilitySettings(editor, *req.Observability)
		})
	case SectionHooksExtensions:
		if req.HooksExtensions == nil {
			return MutationResult{}, validationError(
				errors.New("settings: hooks-extensions section payload is required"),
			)
		}
		changed := diffExtensionsSettings(cfg.Extensions, *req.HooksExtensions)
		return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
			return applyExtensionsSettings(editor, *req.HooksExtensions)
		})
	default:
		return MutationResult{}, notFoundError(fmt.Errorf("settings: unknown section %q", req.Section))
	}
}

func (s *service) loadGlobalSectionUpdate(
	ctx context.Context,
	section SectionName,
	scope ScopeKind,
	workspaceID string,
) (aghconfig.Config, aghconfig.WriteTarget, error) {
	normalizedScope, normalizedWorkspaceID, err := s.normalizeReadScope(scope, workspaceID)
	if err != nil {
		return aghconfig.Config{}, aghconfig.WriteTarget{}, fmt.Errorf("settings: update section %q: %w", section, err)
	}
	if normalizedScope != ScopeGlobal {
		return aghconfig.Config{}, aghconfig.WriteTarget{}, conflictError(
			fmt.Errorf("settings: section %q does not support workspace scope", section),
		)
	}

	cfg, _, err := s.loadConfig(ctx, normalizedScope, normalizedWorkspaceID)
	if err != nil {
		return aghconfig.Config{}, aghconfig.WriteTarget{}, fmt.Errorf(
			"settings: load section %q config: %w",
			section,
			err,
		)
	}

	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return aghconfig.Config{}, aghconfig.WriteTarget{}, fmt.Errorf(
			"settings: resolve section %q write target: %w",
			section,
			err,
		)
	}

	return cfg, target, nil
}

func (s *service) updateConfigSection(
	section SectionName,
	changed []string,
	target aghconfig.WriteTarget,
	mutate func(*aghconfig.OverlayEditor) error,
) (MutationResult, error) {
	if len(changed) == 0 {
		return MutationResult{
			Section:  section,
			Scope:    ScopeGlobal,
			Behavior: MutationBehaviorAppliedNow,
			Applied:  true,
			Warnings: []string{"no changes"},
		}, nil
	}

	classification, err := ClassifyMutation(MutationDescriptor{Section: section, ChangedFields: changed})
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, mutate); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write section %q: %w", section, err)
	}

	return MutationResult{
		Section:         section,
		Scope:           ScopeGlobal,
		WriteTarget:     target.Kind(),
		Behavior:        classification.Behavior,
		Applied:         classification.Applied,
		RestartRequired: classification.RestartRequired,
		RestartScope:    classification.RestartScope,
	}, nil
}

func (s *service) updateSkillsSection(
	ctx context.Context,
	current aghconfig.SkillsConfig,
	next aghconfig.SkillsConfig,
	changed []string,
	target aghconfig.WriteTarget,
) (MutationResult, error) {
	if len(changed) == 0 {
		return MutationResult{
			Section:  SectionSkills,
			Scope:    ScopeGlobal,
			Behavior: MutationBehaviorAppliedNow,
			Applied:  true,
			Warnings: []string{"no changes"},
		}, nil
	}

	classification, err := ClassifyMutation(MutationDescriptor{Section: SectionSkills, ChangedFields: changed})
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		return applySkillsSettings(editor, next)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write section %q: %w", SectionSkills, err)
	}

	if classification.Behavior == MutationBehaviorAppliedNow {
		if err := s.applySkillsDisabledChanges(ctx, current.DisabledSkills, next.DisabledSkills); err != nil {
			return MutationResult{}, err
		}
	}

	return MutationResult{
		Section:         SectionSkills,
		Scope:           ScopeGlobal,
		WriteTarget:     target.Kind(),
		Behavior:        classification.Behavior,
		Applied:         classification.Applied,
		RestartRequired: classification.RestartRequired,
		RestartScope:    classification.RestartScope,
	}, nil
}

func (s *service) buildGeneralSection(ctx context.Context, cfg *aghconfig.Config) (GeneralSection, error) {
	runtime := DaemonRuntimeStatus{}
	if s.generalRuntime != nil {
		status, err := s.generalRuntime.GeneralRuntimeStatus(ctx)
		if err != nil {
			return GeneralSection{}, fmt.Errorf("settings: general runtime status: %w", err)
		}
		runtime = status
	}

	return GeneralSection{
		Runtime: runtime,
		ConfigPaths: ConfigPaths{
			HomeDir:          s.homePaths.HomeDir,
			GlobalConfig:     s.homePaths.ConfigFile,
			GlobalMCPSidecar: globalMCPSidecarPath(s.homePaths),
			LogFile:          s.homePaths.LogFile,
			DaemonInfo:       s.homePaths.DaemonInfo,
		},
		Settings: GeneralSettings{
			Defaults:       cfg.Defaults,
			Limits:         cfg.Limits,
			Permissions:    cfg.Permissions,
			SessionTimeout: cfg.Session.Limits.Timeout,
			HTTP:           cfg.HTTP,
			Daemon:         cfg.Daemon,
		},
		Actions: GeneralActions{
			Restart: ActionMetadata{
				Name:      "restart",
				Available: s.restartActionAvailable,
				Behavior:  MutationBehaviorActionTrigger,
			},
		},
	}, nil
}

func (s *service) buildMemorySection(ctx context.Context, cfg *aghconfig.Config) (MemorySection, error) {
	health := MemoryHealthStatus{}
	if s.memoryRuntime != nil {
		status, err := s.memoryRuntime.MemoryHealthStatus(ctx)
		if err != nil {
			return MemorySection{}, fmt.Errorf("settings: memory health: %w", err)
		}
		health = status
	}

	return MemorySection{
		Config: cfg.Memory,
		Health: health,
		Actions: MemoryActions{
			Consolidate: ActionMetadata{
				Name:      "consolidate",
				Available: s.consolidateActionAvailable,
				Behavior:  MutationBehaviorActionTrigger,
			},
		},
	}, nil
}

func (s *service) buildSkillsSection(cfg *aghconfig.Config) SkillsSection {
	section := SkillsSection{
		Config: cfg.Skills,
		Links: []OperationalLink{
			{Label: "skills", Path: "/skills"},
		},
	}

	if s.skillsRuntime == nil {
		section.DisabledCount = len(cfg.Skills.DisabledSkills)
		return section
	}

	skills := s.skillsRuntime.List()
	section.RuntimeAvailable = true
	section.DiscoveredCount = len(skills)
	for _, skill := range skills {
		if skill != nil && !skill.Enabled {
			section.DisabledCount++
		}
	}

	return section
}

func (s *service) buildAutomationSection(
	ctx context.Context,
	cfg *aghconfig.Config,
) (AutomationSection, error) {
	runtime := AutomationRuntimeStatus{}
	if s.automationRuntime != nil {
		status, err := s.automationRuntime.AutomationRuntimeStatus(ctx)
		if err != nil {
			return AutomationSection{}, fmt.Errorf("settings: automation runtime: %w", err)
		}
		runtime = status
	}

	return AutomationSection{
		Config: AutomationSettings{
			Enabled:           cfg.Automation.Enabled,
			Timezone:          cfg.Automation.Timezone,
			MaxConcurrentJobs: cfg.Automation.MaxConcurrentJobs,
			DefaultFireLimit:  cfg.Automation.DefaultFireLimit,
		},
		Runtime: runtime,
		Links: []OperationalLink{
			{Label: "automation", Path: "/automation"},
		},
	}, nil
}

func (s *service) buildNetworkSection(ctx context.Context, cfg *aghconfig.Config) (NetworkSection, error) {
	runtime := NetworkRuntimeStatus{}
	if s.networkRuntime != nil {
		status, err := s.networkRuntime.NetworkRuntimeStatus(ctx)
		if err != nil {
			return NetworkSection{}, fmt.Errorf("settings: network runtime: %w", err)
		}
		runtime = status
	}

	return NetworkSection{
		Config:  cfg.Network,
		Runtime: runtime,
		Links: []OperationalLink{
			{Label: "network", Path: "/network"},
		},
	}, nil
}

func (s *service) buildObservabilitySection(
	ctx context.Context,
	cfg *aghconfig.Config,
) (ObservabilitySection, error) {
	runtime := ObservabilityRuntimeStatus{}
	if s.observabilityRuntime != nil {
		status, err := s.observabilityRuntime.ObservabilityRuntimeStatus(ctx)
		if err != nil {
			return ObservabilitySection{}, fmt.Errorf("settings: observability runtime: %w", err)
		}
		runtime = status
	}

	return ObservabilitySection{
		Config:         cfg.Observability,
		Runtime:        runtime,
		LogTailSupport: CapabilityStatus{Available: s.logTailAvailable},
	}, nil
}

func (s *service) buildHooksExtensionsSection(
	ctx context.Context,
	cfg *aghconfig.Config,
) (HooksExtensionsSection, error) {
	hooks := buildHookItems(cfg.Hooks.Declarations)

	installed := []InstalledExtension{}
	if s.extensions != nil {
		values, err := s.extensions.InstalledExtensions(ctx)
		if err != nil {
			return HooksExtensionsSection{}, fmt.Errorf("settings: installed extensions: %w", err)
		}
		installed = append(installed, values...)
		sort.Slice(installed, func(i, j int) bool {
			return installed[i].Name < installed[j].Name
		})
	}

	parity := TransportParityStatus{}
	if s.transportParity != nil {
		status, err := s.transportParity.TransportParityStatus(ctx)
		if err != nil {
			return HooksExtensionsSection{}, fmt.Errorf("settings: transport parity status: %w", err)
		}
		parity = status
	}

	return HooksExtensionsSection{
		Hooks:           hooks,
		Extensions:      cloneExtensionsConfig(cfg.Extensions),
		Installed:       installed,
		TransportParity: parity,
	}, nil
}

func diffGeneralSettings(cfg *aghconfig.Config, desired GeneralSettings) []string {
	var changed []string
	if cfg.Defaults.Agent != desired.Defaults.Agent {
		changed = append(changed, "defaults.agent")
	}
	if cfg.Defaults.Provider != desired.Defaults.Provider {
		changed = append(changed, "defaults.provider")
	}
	if cfg.Defaults.Environment != desired.Defaults.Environment {
		changed = append(changed, "defaults.environment")
	}
	if cfg.Limits.MaxSessions != desired.Limits.MaxSessions {
		changed = append(changed, "limits.max_sessions")
	}
	if cfg.Limits.MaxConcurrentAgents != desired.Limits.MaxConcurrentAgents {
		changed = append(changed, "limits.max_concurrent_agents")
	}
	if cfg.Session.Limits.Timeout != desired.SessionTimeout {
		changed = append(changed, "session.limits.timeout")
	}
	if cfg.Permissions.Mode != desired.Permissions.Mode {
		changed = append(changed, "permissions.mode")
	}
	if cfg.HTTP.Host != desired.HTTP.Host {
		changed = append(changed, "http.host")
	}
	if cfg.HTTP.Port != desired.HTTP.Port {
		changed = append(changed, "http.port")
	}
	if cfg.Daemon.Socket != desired.Daemon.Socket {
		changed = append(changed, "daemon.socket")
	}
	return changed
}

func diffMemorySettings(current aghconfig.MemoryConfig, desired aghconfig.MemoryConfig) []string {
	var changed []string
	if current.Enabled != desired.Enabled {
		changed = append(changed, "memory.enabled")
	}
	if current.GlobalDir != desired.GlobalDir {
		changed = append(changed, "memory.global_dir")
	}
	if current.Dream.Enabled != desired.Dream.Enabled {
		changed = append(changed, "memory.dream.enabled")
	}
	if current.Dream.Agent != desired.Dream.Agent {
		changed = append(changed, "memory.dream.agent")
	}
	if current.Dream.MinHours != desired.Dream.MinHours {
		changed = append(changed, "memory.dream.min_hours")
	}
	if current.Dream.MinSessions != desired.Dream.MinSessions {
		changed = append(changed, "memory.dream.min_sessions")
	}
	if current.Dream.CheckInterval != desired.Dream.CheckInterval {
		changed = append(changed, "memory.dream.check_interval")
	}
	return changed
}

func diffSkillsSettings(current aghconfig.SkillsConfig, desired aghconfig.SkillsConfig) []string {
	var changed []string
	if current.Enabled != desired.Enabled {
		changed = append(changed, "skills.enabled")
	}
	if !reflect.DeepEqual(current.DisabledSkills, desired.DisabledSkills) {
		changed = append(changed, "skills.disabled_skills")
	}
	if current.PollInterval != desired.PollInterval {
		changed = append(changed, "skills.poll_interval")
	}
	if !reflect.DeepEqual(current.AllowedMarketplaceMCP, desired.AllowedMarketplaceMCP) {
		changed = append(changed, "skills.allowed_marketplace_mcp")
	}
	if !reflect.DeepEqual(current.AllowedMarketplaceHooks, desired.AllowedMarketplaceHooks) {
		changed = append(changed, "skills.allowed_marketplace_hooks")
	}
	if current.Marketplace.Registry != desired.Marketplace.Registry {
		changed = append(changed, "skills.marketplace.registry")
	}
	if current.Marketplace.BaseURL != desired.Marketplace.BaseURL {
		changed = append(changed, "skills.marketplace.base_url")
	}
	return changed
}

func diffAutomationSettings(cfg *aghconfig.Config, desired AutomationSettings) []string {
	var changed []string
	if cfg.Automation.Enabled != desired.Enabled {
		changed = append(changed, "automation.enabled")
	}
	if cfg.Automation.Timezone != desired.Timezone {
		changed = append(changed, "automation.timezone")
	}
	if cfg.Automation.MaxConcurrentJobs != desired.MaxConcurrentJobs {
		changed = append(changed, "automation.max_concurrent_jobs")
	}
	if cfg.Automation.DefaultFireLimit != desired.DefaultFireLimit {
		changed = append(changed, "automation.default_fire_limit")
	}
	return changed
}

func diffNetworkSettings(current aghconfig.NetworkConfig, desired aghconfig.NetworkConfig) []string {
	var changed []string
	if current.Enabled != desired.Enabled {
		changed = append(changed, "network.enabled")
	}
	if current.DefaultChannel != desired.DefaultChannel {
		changed = append(changed, "network.default_channel")
	}
	if current.Port != desired.Port {
		changed = append(changed, "network.port")
	}
	if current.MaxPayload != desired.MaxPayload {
		changed = append(changed, "network.max_payload")
	}
	if current.GreetInterval != desired.GreetInterval {
		changed = append(changed, "network.greet_interval")
	}
	if current.MaxReplayAge != desired.MaxReplayAge {
		changed = append(changed, "network.max_replay_age")
	}
	if current.MaxQueueDepth != desired.MaxQueueDepth {
		changed = append(changed, "network.max_queue_depth")
	}
	return changed
}

func diffObservabilitySettings(current aghconfig.ObservabilityConfig, desired aghconfig.ObservabilityConfig) []string {
	var changed []string
	if current.Enabled != desired.Enabled {
		changed = append(changed, "observability.enabled")
	}
	if current.RetentionDays != desired.RetentionDays {
		changed = append(changed, "observability.retention_days")
	}
	if current.MaxGlobalBytes != desired.MaxGlobalBytes {
		changed = append(changed, "observability.max_global_bytes")
	}
	if current.Transcripts.Enabled != desired.Transcripts.Enabled {
		changed = append(changed, "observability.transcripts.enabled")
	}
	if current.Transcripts.SegmentBytes != desired.Transcripts.SegmentBytes {
		changed = append(changed, "observability.transcripts.segment_bytes")
	}
	if current.Transcripts.MaxBytesPerSession != desired.Transcripts.MaxBytesPerSession {
		changed = append(changed, "observability.transcripts.max_bytes_per_session")
	}
	return changed
}

func diffExtensionsSettings(current aghconfig.ExtensionsConfig, desired aghconfig.ExtensionsConfig) []string {
	var changed []string
	if current.Marketplace.Registry != desired.Marketplace.Registry {
		changed = append(changed, "extensions.marketplace.registry")
	}
	if current.Marketplace.BaseURL != desired.Marketplace.BaseURL {
		changed = append(changed, "extensions.marketplace.base_url")
	}
	if !reflect.DeepEqual(current.Resources.AllowedKinds, desired.Resources.AllowedKinds) {
		changed = append(changed, "extensions.resources.allowed_kinds")
	}
	if current.Resources.MaxScope != desired.Resources.MaxScope {
		changed = append(changed, "extensions.resources.max_scope")
	}
	if current.Resources.SnapshotRateLimit != desired.Resources.SnapshotRateLimit {
		changed = append(changed, "extensions.resources.snapshot_rate_limit")
	}
	if current.Resources.OperatorWriteRateLimit != desired.Resources.OperatorWriteRateLimit {
		changed = append(changed, "extensions.resources.operator_write_rate_limit")
	}
	return changed
}

func applyGeneralSettings(editor *aghconfig.OverlayEditor, settings GeneralSettings) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"defaults", "agent"}, value: settings.Defaults.Agent},
		{path: []string{"defaults", "provider"}, value: settings.Defaults.Provider},
		{path: []string{"defaults", "environment"}, value: settings.Defaults.Environment},
		{path: []string{"limits", "max_sessions"}, value: settings.Limits.MaxSessions},
		{path: []string{"limits", "max_concurrent_agents"}, value: settings.Limits.MaxConcurrentAgents},
		{path: []string{"session", "limits", "timeout"}, value: settings.SessionTimeout.String()},
		{path: []string{"permissions", "mode"}, value: string(settings.Permissions.Mode)},
		{path: []string{"http", "host"}, value: settings.HTTP.Host},
		{path: []string{"http", "port"}, value: settings.HTTP.Port},
		{path: []string{"daemon", "socket"}, value: settings.Daemon.Socket},
	}
	return applyValueUpdates(editor, updates)
}

func applyMemorySettings(editor *aghconfig.OverlayEditor, settings aghconfig.MemoryConfig) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "enabled"}, value: settings.Enabled},
		{path: []string{"memory", "global_dir"}, value: settings.GlobalDir},
		{path: []string{"memory", "dream", "enabled"}, value: settings.Dream.Enabled},
		{path: []string{"memory", "dream", "agent"}, value: settings.Dream.Agent},
		{path: []string{"memory", "dream", "min_hours"}, value: settings.Dream.MinHours},
		{path: []string{"memory", "dream", "min_sessions"}, value: settings.Dream.MinSessions},
		{path: []string{"memory", "dream", "check_interval"}, value: settings.Dream.CheckInterval.String()},
	}
	return applyValueUpdates(editor, updates)
}

func applySkillsSettings(editor *aghconfig.OverlayEditor, settings aghconfig.SkillsConfig) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"skills", "enabled"}, value: settings.Enabled},
		{path: []string{"skills", "disabled_skills"}, value: append([]string(nil), settings.DisabledSkills...)},
		{path: []string{"skills", "poll_interval"}, value: settings.PollInterval.String()},
		{
			path:  []string{"skills", "allowed_marketplace_mcp"},
			value: append([]string(nil), settings.AllowedMarketplaceMCP...),
		},
		{
			path:  []string{"skills", "allowed_marketplace_hooks"},
			value: append([]string(nil), settings.AllowedMarketplaceHooks...),
		},
		{path: []string{"skills", "marketplace", "registry"}, value: settings.Marketplace.Registry},
		{path: []string{"skills", "marketplace", "base_url"}, value: settings.Marketplace.BaseURL},
	}
	return applyValueUpdates(editor, updates)
}

func applyAutomationSettings(editor *aghconfig.OverlayEditor, settings AutomationSettings) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"automation", "enabled"}, value: settings.Enabled},
		{path: []string{"automation", "timezone"}, value: settings.Timezone},
		{path: []string{"automation", "max_concurrent_jobs"}, value: settings.MaxConcurrentJobs},
		{path: []string{"automation", "default_fire_limit", "max"}, value: settings.DefaultFireLimit.Max},
		{path: []string{"automation", "default_fire_limit", "window"}, value: settings.DefaultFireLimit.Window},
	}
	return applyValueUpdates(editor, updates)
}

func applyNetworkSettings(editor *aghconfig.OverlayEditor, settings aghconfig.NetworkConfig) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"network", "enabled"}, value: settings.Enabled},
		{path: []string{"network", "default_channel"}, value: settings.DefaultChannel},
		{path: []string{"network", "port"}, value: settings.Port},
		{path: []string{"network", "max_payload"}, value: settings.MaxPayload},
		{path: []string{"network", "greet_interval"}, value: settings.GreetInterval},
		{path: []string{"network", "max_replay_age"}, value: settings.MaxReplayAge},
		{path: []string{"network", "max_queue_depth"}, value: settings.MaxQueueDepth},
	}
	return applyValueUpdates(editor, updates)
}

func applyObservabilitySettings(editor *aghconfig.OverlayEditor, settings aghconfig.ObservabilityConfig) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"observability", "enabled"}, value: settings.Enabled},
		{path: []string{"observability", "retention_days"}, value: settings.RetentionDays},
		{path: []string{"observability", "max_global_bytes"}, value: settings.MaxGlobalBytes},
		{path: []string{"observability", "transcripts", "enabled"}, value: settings.Transcripts.Enabled},
		{path: []string{"observability", "transcripts", "segment_bytes"}, value: settings.Transcripts.SegmentBytes},
		{
			path:  []string{"observability", "transcripts", "max_bytes_per_session"},
			value: settings.Transcripts.MaxBytesPerSession,
		},
	}
	return applyValueUpdates(editor, updates)
}

func applyExtensionsSettings(editor *aghconfig.OverlayEditor, settings aghconfig.ExtensionsConfig) error {
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"extensions", "marketplace", "registry"}, value: settings.Marketplace.Registry},
		{path: []string{"extensions", "marketplace", "base_url"}, value: settings.Marketplace.BaseURL},
		{
			path:  []string{"extensions", "resources", "allowed_kinds"},
			value: resourceKindsToStrings(settings.Resources.AllowedKinds),
		},
		{path: []string{"extensions", "resources", "max_scope"}, value: string(settings.Resources.MaxScope)},
		{
			path:  []string{"extensions", "resources", "snapshot_rate_limit", "requests"},
			value: settings.Resources.SnapshotRateLimit.Requests,
		},
		{
			path:  []string{"extensions", "resources", "snapshot_rate_limit", "window"},
			value: settings.Resources.SnapshotRateLimit.Window.String(),
		},
		{
			path:  []string{"extensions", "resources", "snapshot_rate_limit", "queue"},
			value: settings.Resources.SnapshotRateLimit.Queue,
		},
		{
			path:  []string{"extensions", "resources", "operator_write_rate_limit", "requests"},
			value: settings.Resources.OperatorWriteRateLimit.Requests,
		},
		{
			path:  []string{"extensions", "resources", "operator_write_rate_limit", "window"},
			value: settings.Resources.OperatorWriteRateLimit.Window.String(),
		},
		{
			path:  []string{"extensions", "resources", "operator_write_rate_limit", "queue"},
			value: settings.Resources.OperatorWriteRateLimit.Queue,
		},
	}
	return applyValueUpdates(editor, updates)
}

func applyValueUpdates(editor *aghconfig.OverlayEditor, updates []struct {
	path  []string
	value any
}) error {
	for _, update := range updates {
		if err := editor.SetValue(update.path, update.value); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) applySkillsDisabledChanges(_ context.Context, current []string, next []string) error {
	if s.skillsRuntime == nil {
		return errors.New("settings: skills runtime is required to apply skills.disabled_skills")
	}

	currentSet := sliceToSet(current)
	nextSet := sliceToSet(next)
	for _, skill := range s.skillsRuntime.List() {
		if skill == nil {
			continue
		}
		name := strings.TrimSpace(skill.Meta.Name)
		if name == "" {
			continue
		}
		_, wasDisabled := currentSet[name]
		_, nowDisabled := nextSet[name]
		if wasDisabled == nowDisabled {
			continue
		}
		if err := s.skillsRuntime.SetEnabled(name, nil, !nowDisabled); err != nil {
			return fmt.Errorf("settings: apply skills.disabled_skills for %q: %w", name, err)
		}
	}
	return nil
}

func sliceToSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	return set
}

func cloneExtensionsConfig(value aghconfig.ExtensionsConfig) aghconfig.ExtensionsConfig {
	return aghconfig.ExtensionsConfig{
		Marketplace: aghconfig.ExtensionsMarketplaceConfig{
			Registry: value.Marketplace.Registry,
			BaseURL:  value.Marketplace.BaseURL,
		},
		Resources: aghconfig.ExtensionsResourcesConfig{
			AllowedKinds: cloneAllowedKinds(value.Resources.AllowedKinds),
			MaxScope:     value.Resources.MaxScope,
			SnapshotRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
				Requests: value.Resources.SnapshotRateLimit.Requests,
				Window:   value.Resources.SnapshotRateLimit.Window,
				Queue:    value.Resources.SnapshotRateLimit.Queue,
			},
			OperatorWriteRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
				Requests: value.Resources.OperatorWriteRateLimit.Requests,
				Window:   value.Resources.OperatorWriteRateLimit.Window,
				Queue:    value.Resources.OperatorWriteRateLimit.Queue,
			},
		},
	}
}

func resourceKindsToStrings(values []resources.ResourceKind) []string {
	if len(values) == 0 {
		return nil
	}
	converted := make([]string, 0, len(values))
	for _, value := range values {
		converted = append(converted, string(value))
	}
	return converted
}

func globalMCPSidecarPath(homePaths aghconfig.HomePaths) string {
	return filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName)
}
