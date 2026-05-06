package settings

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/resources"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (s *service) GetSection(ctx context.Context, req SectionRequest) (SectionEnvelope, error) {
	scope, workspaceID, agentName, err := s.resolveSectionScope(req.Section, req.Scope, req.WorkspaceID, req.AgentName)
	if err != nil {
		return SectionEnvelope{}, fmt.Errorf("settings: get section %q: %w", req.Section, err)
	}

	cfg, resolved, err := s.loadConfig(ctx, scope, workspaceID)
	if err != nil {
		return SectionEnvelope{}, fmt.Errorf("settings: load section %q config: %w", req.Section, err)
	}

	envelope := newSectionEnvelope(req.Section, scope, workspaceID, agentName)
	if err := s.populateSectionEnvelope(ctx, &envelope, &cfg, resolved); err != nil {
		return SectionEnvelope{}, err
	}

	return envelope, nil
}

func (s *service) UpdateSection(ctx context.Context, req SectionUpdateRequest) (MutationResult, error) {
	if req.Section == SectionSkills {
		if req.Skills == nil {
			return MutationResult{}, validationError(errors.New("settings: skills section payload is required"))
		}
		result, err := s.updateSkillsSection(ctx, req.SectionRequest, *req.Skills)
		return s.finalizeSectionUpdate(ctx, result, err)
	}

	result, err := s.updateConfigBackedSection(ctx, req)
	return s.finalizeSectionUpdate(ctx, result, err)
}

func (s *service) resolveSectionScope(
	section SectionName,
	scope ScopeKind,
	workspaceID string,
	agentName string,
) (ScopeKind, string, string, error) {
	normalizedScope, normalizedWorkspaceID, err := s.normalizeReadScope(scope, workspaceID)
	if err != nil {
		return "", "", "", err
	}
	normalizedAgentName, err := normalizeAgentName(agentName)
	if err != nil {
		return "", "", "", err
	}
	if err := validateSectionScope(section, normalizedScope, normalizedAgentName); err != nil {
		return "", "", "", err
	}
	return normalizedScope, normalizedWorkspaceID, normalizedAgentName, nil
}

func validateSectionScope(section SectionName, scope ScopeKind, agentName string) error {
	if section != SectionSkills && scope != ScopeGlobal {
		return conflictError(
			fmt.Errorf("settings: section %q does not support %s scope", section, scope),
		)
	}
	if section != SectionSkills {
		return nil
	}
	if scope == ScopeWorkspace {
		return conflictError(
			fmt.Errorf("settings: section %q does not support %s scope", section, scope),
		)
	}
	if scope == ScopeAgent && agentName == "" {
		return validationError(errors.New("settings: agent scope requires agent_name"))
	}
	return nil
}

func newSectionEnvelope(
	section SectionName,
	scope ScopeKind,
	workspaceID string,
	agentName string,
) SectionEnvelope {
	return SectionEnvelope{
		Section:         section,
		Scope:           scope,
		WorkspaceID:     workspaceID,
		AgentName:       agentName,
		AvailableScopes: []ScopeKind{ScopeGlobal},
	}
}

func (s *service) populateSectionEnvelope(
	ctx context.Context,
	envelope *SectionEnvelope,
	cfg *aghconfig.Config,
	resolved *workspacepkg.ResolvedWorkspace,
) error {
	switch envelope.Section {
	case SectionGeneral:
		envelope.Scope = ScopeGlobal
		section, err := s.buildGeneralSection(ctx, cfg)
		if err != nil {
			return err
		}
		envelope.General = &section
	case SectionMemory:
		envelope.Scope = ScopeGlobal
		section, err := s.buildMemorySection(ctx, cfg)
		if err != nil {
			return err
		}
		envelope.Memory = &section
	case SectionSkills:
		envelope.AvailableScopes = []ScopeKind{ScopeGlobal, ScopeAgent}
		section, err := s.buildSkillsSection(
			ctx,
			cfg,
			resolved,
			envelope.Scope,
			envelope.WorkspaceID,
			envelope.AgentName,
		)
		if err != nil {
			return err
		}
		envelope.Skills = &section
	case SectionAutomation:
		envelope.Scope = ScopeGlobal
		section, err := s.buildAutomationSection(ctx, cfg)
		if err != nil {
			return err
		}
		envelope.Automation = &section
	case SectionNetwork:
		envelope.Scope = ScopeGlobal
		section, err := s.buildNetworkSection(ctx, cfg)
		if err != nil {
			return err
		}
		envelope.Network = &section
	case SectionObservability:
		envelope.Scope = ScopeGlobal
		section, err := s.buildObservabilitySection(ctx, cfg)
		if err != nil {
			return err
		}
		envelope.Observability = &section
	case SectionHooksExtensions:
		envelope.Scope = ScopeGlobal
		section, err := s.buildHooksExtensionsSection(ctx, cfg)
		if err != nil {
			return err
		}
		envelope.HooksExtensions = &section
	default:
		return notFoundError(fmt.Errorf("settings: unknown section %q", envelope.Section))
	}
	return nil
}

func (s *service) finalizeSectionUpdate(
	ctx context.Context,
	result MutationResult,
	err error,
) (MutationResult, error) {
	if err != nil {
		return MutationResult{}, err
	}
	if emitErr := s.emitSettingsChanged(ctx, result, "patch"); emitErr != nil {
		return MutationResult{}, emitErr
	}
	return result, nil
}

func (s *service) updateConfigBackedSection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	switch req.Section {
	case SectionGeneral:
		return s.updateGeneralSection(ctx, req)
	case SectionMemory:
		return s.updateMemorySection(ctx, req)
	case SectionAutomation:
		return s.updateAutomationSection(ctx, req)
	case SectionNetwork:
		return s.updateNetworkSection(ctx, req)
	case SectionObservability:
		return s.updateObservabilitySection(ctx, req)
	case SectionHooksExtensions:
		return s.updateHooksExtensionsSection(ctx, req)
	default:
		return MutationResult{}, notFoundError(fmt.Errorf("settings: unknown section %q", req.Section))
	}
}

func (s *service) updateGeneralSection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.General == nil {
		return MutationResult{}, validationError(errors.New("settings: general section payload is required"))
	}
	changed := diffGeneralSettings(&cfg, *req.General)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyGeneralSettings(editor, *req.General)
	})
}

func (s *service) updateMemorySection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.Memory == nil {
		return MutationResult{}, validationError(errors.New("settings: memory section payload is required"))
	}
	changed := diffMemorySettings(&cfg.Memory, req.Memory)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyMemorySettings(editor, req.Memory)
	})
}

func (s *service) updateAutomationSection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.Automation == nil {
		return MutationResult{}, validationError(errors.New("settings: automation section payload is required"))
	}
	changed := diffAutomationSettings(&cfg, *req.Automation)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyAutomationSettings(editor, *req.Automation)
	})
}

func (s *service) updateNetworkSection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.Network == nil {
		return MutationResult{}, validationError(errors.New("settings: network section payload is required"))
	}
	changed := diffNetworkSettings(cfg.Network, *req.Network)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyNetworkSettings(editor, *req.Network)
	})
}

func (s *service) updateObservabilitySection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.Observability == nil {
		return MutationResult{}, validationError(
			errors.New("settings: observability section payload is required"),
		)
	}
	changed := diffObservabilitySettings(cfg.Observability, *req.Observability)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyObservabilitySettings(editor, *req.Observability)
	})
}

func (s *service) updateHooksExtensionsSection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.HooksExtensions == nil {
		return MutationResult{}, validationError(
			errors.New("settings: hooks-extensions section payload is required"),
		)
	}
	changed := diffExtensionsSettings(cfg.Extensions, *req.HooksExtensions)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyExtensionsSettings(editor, *req.HooksExtensions)
	})
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
	req SectionRequest,
	next aghconfig.SkillsConfig,
) (MutationResult, error) {
	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: update section %q: %w", SectionSkills, err)
	}
	if scope == ScopeWorkspace {
		return MutationResult{}, conflictError(
			errors.New("settings: section \"skills\" does not support workspace scope"),
		)
	}
	agentName, err := normalizeAgentName(req.AgentName)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: update section %q: %w", SectionSkills, err)
	}

	cfg, resolved, err := s.loadConfig(ctx, scope, workspaceID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: load section %q config: %w", SectionSkills, err)
	}

	if scope == ScopeAgent {
		if agentName == "" {
			return MutationResult{}, validationError(errors.New("settings: agent scope requires agent_name"))
		}
		return s.updateAgentSkillsSection(cfg.Skills, resolved, workspaceID, agentName, next)
	}

	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: resolve section %q write target: %w", SectionSkills, err)
	}

	current := cfg.Skills
	changed := diffSkillsSettings(current, next)
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
		if err := s.applySkillsDisabledChanges(current.DisabledSkills, next.DisabledSkills); err != nil {
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

func (s *service) updateAgentSkillsSection(
	base aghconfig.SkillsConfig,
	resolved *workspacepkg.ResolvedWorkspace,
	workspaceID string,
	agentName string,
	next aghconfig.SkillsConfig,
) (MutationResult, error) {
	agent, targetKind, err := s.resolveEffectiveAgent(resolved, agentName)
	if err != nil {
		return MutationResult{}, err
	}
	if immutable := diffAgentImmutableSkillsSettings(base, next); len(immutable) > 0 {
		return MutationResult{}, validationError(
			fmt.Errorf(
				"settings: agent scope only supports skills.disabled_skills, got %s",
				strings.Join(immutable, ", "),
			),
		)
	}

	current := base
	current.DisabledSkills = append([]string(nil), agent.Skills.Disabled...)
	changed := diffSkillsSettings(current, next)
	if len(changed) == 0 {
		return MutationResult{
			Section:     SectionSkills,
			Scope:       ScopeAgent,
			WriteTarget: targetKind,
			WorkspaceID: workspaceID,
			AgentName:   agentName,
			Behavior:    MutationBehaviorAppliedNow,
			Applied:     true,
			Warnings:    []string{"no changes"},
		}, nil
	}

	classification, err := ClassifyMutation(MutationDescriptor{Section: SectionSkills, ChangedFields: changed})
	if err != nil {
		return MutationResult{}, err
	}
	if classification.Behavior == MutationBehaviorAppliedNow {
		if err := s.applyAgentSkillsDisabledChanges(
			resolved,
			agentName,
			current.DisabledSkills,
			next.DisabledSkills,
		); err != nil {
			return MutationResult{}, err
		}
	}

	return MutationResult{
		Section:         SectionSkills,
		Scope:           ScopeAgent,
		WriteTarget:     targetKind,
		WorkspaceID:     workspaceID,
		AgentName:       agentName,
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

func (s *service) buildSkillsSection(
	ctx context.Context,
	cfg *aghconfig.Config,
	resolved *workspacepkg.ResolvedWorkspace,
	scope ScopeKind,
	workspaceID string,
	agentName string,
) (SkillsSection, error) {
	section := SkillsSection{
		Config: cfg.Skills,
	}
	section.Links = buildSkillsOperationalLinks(scope, workspaceID, agentName)

	if scope == ScopeAgent {
		agent, _, err := s.resolveEffectiveAgent(resolved, agentName)
		if err != nil {
			return SkillsSection{}, err
		}
		section.Config.DisabledSkills = append([]string(nil), agent.Skills.Disabled...)
	}

	if s.skillsRuntime == nil {
		section.DisabledCount = len(section.Config.DisabledSkills)
		return section, nil
	}

	var (
		skills []*skillspkg.Skill
		err    error
	)
	if scope == ScopeAgent {
		skills, err = s.skillsRuntime.ForAgent(ctx, resolved, agentName)
	} else {
		skills = s.skillsRuntime.List()
	}
	if err != nil {
		return SkillsSection{}, mapSkillsSettingsError(err)
	}
	section.RuntimeAvailable = true
	section.DiscoveredCount = len(skills)
	for _, skill := range skills {
		if skill != nil && !skill.Enabled {
			section.DisabledCount++
		}
	}

	return section, nil
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
	if cfg.Defaults.Sandbox != desired.Defaults.Sandbox {
		changed = append(changed, "defaults.sandbox")
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

func diffMemorySettings(current *aghconfig.MemoryConfig, desired *aghconfig.MemoryConfig) []string {
	var changed []string
	currentValues := memorySettingsUpdates(current)
	desiredValues := memorySettingsUpdates(desired)
	for i, currentValue := range currentValues {
		if i >= len(desiredValues) {
			break
		}
		desiredValue := desiredValues[i]
		if reflect.DeepEqual(currentValue.value, desiredValue.value) {
			continue
		}
		changed = append(changed, strings.Join(currentValue.path, "."))
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

func diffAgentImmutableSkillsSettings(current aghconfig.SkillsConfig, desired aghconfig.SkillsConfig) []string {
	baseCurrent := current
	baseCurrent.DisabledSkills = nil
	baseDesired := desired
	baseDesired.DisabledSkills = nil
	return diffSkillsSettings(baseCurrent, baseDesired)
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
		{path: []string{"defaults", "sandbox"}, value: settings.Defaults.Sandbox},
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

func applyMemorySettings(editor *aghconfig.OverlayEditor, settings *aghconfig.MemoryConfig) error {
	return applyValueUpdates(editor, memorySettingsUpdates(settings))
}

func memorySettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	if settings == nil {
		return nil
	}
	updates := []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "enabled"}, value: settings.Enabled},
		{path: []string{"memory", "global_dir"}, value: settings.GlobalDir},
	}
	updates = append(updates, memoryControllerSettingsUpdates(settings)...)
	updates = append(updates, memoryRecallSettingsUpdates(settings)...)
	updates = append(updates, memoryExtractorSettingsUpdates(settings)...)
	updates = append(updates, memoryDreamSettingsUpdates(settings)...)
	updates = append(updates, memoryRetentionSettingsUpdates(settings)...)
	return append(updates, memoryProviderSettingsUpdates(settings)...)
}

func memoryControllerSettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	return []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "controller", "mode"}, value: settings.Controller.Mode},
		{path: []string{"memory", "controller", "max_latency"}, value: settings.Controller.MaxLatency.String()},
		{path: []string{"memory", "controller", "default_op_on_fail"}, value: settings.Controller.DefaultOpOnFail},
		{path: []string{"memory", "controller", "llm", "enabled"}, value: settings.Controller.LLM.Enabled},
		{path: []string{"memory", "controller", "llm", "model"}, value: settings.Controller.LLM.Model},
		{path: []string{"memory", "controller", "llm", "top_k"}, value: settings.Controller.LLM.TopK},
		{path: []string{"memory", "controller", "llm", "prompt_version"}, value: settings.Controller.LLM.PromptVersion},
		{path: []string{"memory", "controller", "llm", "timeout"}, value: settings.Controller.LLM.Timeout.String()},
		{path: []string{"memory", "controller", "llm", "max_tokens_out"}, value: settings.Controller.LLM.MaxTokensOut},
		{
			path:  []string{"memory", "controller", "policy", "max_content_chars"},
			value: settings.Controller.Policy.MaxContentChars,
		},
		{
			path:  []string{"memory", "controller", "policy", "max_writes_per_min"},
			value: settings.Controller.Policy.MaxWritesPerMin,
		},
		{
			path:  []string{"memory", "controller", "policy", "allow_origins"},
			value: append([]string(nil), settings.Controller.Policy.AllowOrigins...),
		},
	}
}

func memoryRecallSettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	return []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "recall", "top_k"}, value: settings.Recall.TopK},
		{path: []string{"memory", "recall", "raw_candidates"}, value: settings.Recall.RawCandidates},
		{path: []string{"memory", "recall", "fusion"}, value: settings.Recall.Fusion},
		{path: []string{"memory", "recall", "include_already_surfaced"}, value: settings.Recall.IncludeAlreadySurfaced},
		{path: []string{"memory", "recall", "include_system"}, value: settings.Recall.IncludeSystem},
		{path: []string{"memory", "recall", "weights", "bm25_unicode"}, value: settings.Recall.Weights.BM25Unicode},
		{path: []string{"memory", "recall", "weights", "bm25_trigram"}, value: settings.Recall.Weights.BM25Trigram},
		{path: []string{"memory", "recall", "weights", "recency"}, value: settings.Recall.Weights.Recency},
		{path: []string{"memory", "recall", "weights", "recall_signal"}, value: settings.Recall.Weights.RecallSignal},
		{
			path:  []string{"memory", "recall", "freshness", "banner_after_days"},
			value: settings.Recall.Freshness.BannerAfterDays,
		},
		{path: []string{"memory", "recall", "signals", "queue_capacity"}, value: settings.Recall.Signals.QueueCapacity},
		{
			path:  []string{"memory", "recall", "signals", "worker_retry_max"},
			value: settings.Recall.Signals.WorkerRetryMax,
		},
		{
			path:  []string{"memory", "recall", "signals", "metrics_enabled"},
			value: settings.Recall.Signals.MetricsEnabled,
		},
		{
			path:  []string{"memory", "decisions", "prune_after_applied_days"},
			value: settings.Decisions.PruneAfterAppliedDays,
		},
		{path: []string{"memory", "decisions", "keep_audit_summary"}, value: settings.Decisions.KeepAuditSummary},
		{
			path:  []string{"memory", "decisions", "max_post_content_bytes"},
			value: settings.Decisions.MaxPostContentBytes,
		},
	}
}

func memoryExtractorSettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	return []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "extractor", "enabled"}, value: settings.Extractor.Enabled},
		{path: []string{"memory", "extractor", "mode"}, value: settings.Extractor.Mode},
		{path: []string{"memory", "extractor", "throttle_turns"}, value: settings.Extractor.ThrottleTurns},
		{path: []string{"memory", "extractor", "deadline"}, value: settings.Extractor.Deadline.String()},
		{path: []string{"memory", "extractor", "sandbox_inbox_only"}, value: settings.Extractor.SandboxInboxOnly},
		{path: []string{"memory", "extractor", "inbox_path"}, value: settings.Extractor.InboxPath},
		{path: []string{"memory", "extractor", "dlq_path"}, value: settings.Extractor.DLQPath},
		{path: []string{"memory", "extractor", "model"}, value: settings.Extractor.Model},
		{path: []string{"memory", "extractor", "queue", "capacity"}, value: settings.Extractor.Queue.Capacity},
		{path: []string{"memory", "extractor", "queue", "coalesce_max"}, value: settings.Extractor.Queue.CoalesceMax},
	}
}

func memoryDreamSettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	return []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "dream", "enabled"}, value: settings.Dream.Enabled},
		{path: []string{"memory", "dream", "agent"}, value: settings.Dream.Agent},
		{path: []string{"memory", "dream", "min_hours"}, value: settings.Dream.MinHours},
		{path: []string{"memory", "dream", "min_sessions"}, value: settings.Dream.MinSessions},
		{path: []string{"memory", "dream", "debounce"}, value: settings.Dream.Debounce.String()},
		{path: []string{"memory", "dream", "prompt_version"}, value: settings.Dream.PromptVersion},
		{path: []string{"memory", "dream", "check_interval"}, value: settings.Dream.CheckInterval.String()},
		{path: []string{"memory", "dream", "gates", "min_unpromoted"}, value: settings.Dream.Gates.MinUnpromoted},
		{path: []string{"memory", "dream", "gates", "min_recall_count"}, value: settings.Dream.Gates.MinRecallCount},
		{path: []string{"memory", "dream", "gates", "min_score"}, value: settings.Dream.Gates.MinScore},
		{
			path:  []string{"memory", "dream", "scoring", "recency_half_life_days"},
			value: settings.Dream.Scoring.RecencyHalfLifeDays,
		},
		{
			path:  []string{"memory", "dream", "scoring", "weights", "frequency"},
			value: settings.Dream.Scoring.Weights.Frequency,
		},
		{
			path:  []string{"memory", "dream", "scoring", "weights", "relevance"},
			value: settings.Dream.Scoring.Weights.Relevance,
		},
		{
			path:  []string{"memory", "dream", "scoring", "weights", "recency"},
			value: settings.Dream.Scoring.Weights.Recency,
		},
		{
			path:  []string{"memory", "dream", "scoring", "weights", "freshness"},
			value: settings.Dream.Scoring.Weights.Freshness,
		},
	}
}

func memoryRetentionSettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	return []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "session", "ledger_format"}, value: settings.Session.LedgerFormat},
		{path: []string{"memory", "session", "ledger_root"}, value: settings.Session.LedgerRoot},
		{path: []string{"memory", "session", "events_purge_grace"}, value: settings.Session.EventsPurgeGrace.String()},
		{path: []string{"memory", "session", "cold_archive_days"}, value: settings.Session.ColdArchiveDays},
		{path: []string{"memory", "session", "hard_delete_days"}, value: settings.Session.HardDeleteDays},
		{path: []string{"memory", "session", "max_archive_bytes"}, value: settings.Session.MaxArchiveBytes},
		{path: []string{"memory", "session", "unbound_partition"}, value: settings.Session.UnboundPartition},
		{path: []string{"memory", "daily", "max_bytes"}, value: settings.Daily.MaxBytes},
		{path: []string{"memory", "daily", "max_lines"}, value: settings.Daily.MaxLines},
		{path: []string{"memory", "daily", "rotate_format"}, value: settings.Daily.RotateFormat},
		{path: []string{"memory", "daily", "dreaming_window"}, value: settings.Daily.DreamingWindow},
		{path: []string{"memory", "daily", "cold_archive_days"}, value: settings.Daily.ColdArchiveDays},
		{path: []string{"memory", "daily", "hard_delete_days"}, value: settings.Daily.HardDeleteDays},
		{path: []string{"memory", "daily", "max_archive_bytes"}, value: settings.Daily.MaxArchiveBytes},
		{path: []string{"memory", "daily", "sweep_hour"}, value: settings.Daily.SweepHour},
		{path: []string{"memory", "daily", "archive_path"}, value: settings.Daily.ArchivePath},
		{path: []string{"memory", "file", "max_lines"}, value: settings.File.MaxLines},
		{path: []string{"memory", "file", "max_bytes"}, value: settings.File.MaxBytes},
	}
}

func memoryProviderSettingsUpdates(settings *aghconfig.MemoryConfig) []struct {
	path  []string
	value any
} {
	return []struct {
		path  []string
		value any
	}{
		{path: []string{"memory", "provider", "name"}, value: settings.Provider.Name},
		{path: []string{"memory", "provider", "timeout"}, value: settings.Provider.Timeout.String()},
		{path: []string{"memory", "provider", "failure_threshold"}, value: settings.Provider.FailureThreshold},
		{path: []string{"memory", "provider", "cooldown"}, value: settings.Provider.Cooldown.String()},
		{path: []string{"memory", "workspace", "auto_create"}, value: settings.Workspace.AutoCreate},
	}
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

func (s *service) applySkillsDisabledChanges(current []string, next []string) error {
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

func (s *service) applyAgentSkillsDisabledChanges(
	resolved *workspacepkg.ResolvedWorkspace,
	agentName string,
	current []string,
	next []string,
) error {
	if s.skillsRuntime == nil {
		return errors.New("settings: skills runtime is required to apply agent skills.disabled_skills")
	}

	currentSet := sliceToSet(current)
	nextSet := sliceToSet(next)
	names := make([]string, 0, len(currentSet)+len(nextSet))
	seen := make(map[string]struct{}, len(currentSet)+len(nextSet))
	for name := range currentSet {
		names = append(names, name)
		seen[name] = struct{}{}
	}
	for name := range nextSet {
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		_, wasDisabled := currentSet[name]
		_, nowDisabled := nextSet[name]
		if wasDisabled == nowDisabled {
			continue
		}
		if err := s.skillsRuntime.SetEnabledForAgent(name, resolved, agentName, !nowDisabled); err != nil {
			return mapSkillsSettingsError(
				fmt.Errorf("settings: apply agent skills.disabled_skills for %q on %q: %w", name, agentName, err),
			)
		}
	}

	return nil
}

func buildSkillsOperationalLinks(scope ScopeKind, workspaceID string, agentName string) []OperationalLink {
	values := url.Values{}
	if trimmed := strings.TrimSpace(workspaceID); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	if scope == ScopeAgent && strings.TrimSpace(agentName) != "" {
		values.Set("for_agent", agentName)
	}

	path := "/skills"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return []OperationalLink{{Label: "skills", Path: path}}
}

func (s *service) resolveEffectiveAgent(
	resolved *workspacepkg.ResolvedWorkspace,
	agentName string,
) (aghconfig.AgentDef, WriteTargetKind, error) {
	target := aghconfig.NormalizeAgentName(agentName)
	if target == "" {
		return aghconfig.AgentDef{}, "", validationError(errors.New("settings: agent_name is required"))
	}

	if resolved != nil {
		for _, diagnostic := range resolved.AgentDiagnostics {
			if aghconfig.NormalizeAgentName(diagnostic.Name) != target {
				continue
			}
			return aghconfig.AgentDef{}, "", unprocessableError(
				fmt.Errorf(
					"settings: agent %q at %q: %s",
					target,
					strings.TrimSpace(diagnostic.Path),
					strings.TrimSpace(diagnostic.Message),
				),
			)
		}
		for _, agent := range resolved.Agents {
			if aghconfig.NormalizeAgentName(agent.Name) != target {
				continue
			}
			return aghconfig.CloneAgentDef(agent), agentWriteTargetKind(s.homePaths, agent.SourcePath), nil
		}
		return aghconfig.AgentDef{}, "", notFoundError(fmt.Errorf("settings: agent %q not found", target))
	}

	path := filepath.Join(s.homePaths.AgentsDir, target, "AGENT.md")
	agent, err := aghconfig.LoadAgentDefFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return aghconfig.AgentDef{}, "", notFoundError(fmt.Errorf("settings: agent %q not found", target))
		}
		return aghconfig.AgentDef{}, "", unprocessableError(
			fmt.Errorf("settings: load agent %q: %w", target, err),
		)
	}
	if aghconfig.NormalizeAgentName(agent.Name) != target {
		return aghconfig.AgentDef{}, "", unprocessableError(
			fmt.Errorf("settings: agent file %q defines %q, expected %q", path, agent.Name, target),
		)
	}
	return agent, WriteTargetGlobalAgentFile, nil
}

func agentWriteTargetKind(homePaths aghconfig.HomePaths, sourcePath string) WriteTargetKind {
	if withinRoot(sourcePath, homePaths.AgentsDir) {
		return WriteTargetGlobalAgentFile
	}
	return WriteTargetWorkspaceAgentFile
}

func withinRoot(path string, root string) bool {
	trimmedPath := strings.TrimSpace(path)
	trimmedRoot := strings.TrimSpace(root)
	if trimmedPath == "" || trimmedRoot == "" {
		return false
	}
	rel, err := filepath.Rel(trimmedRoot, trimmedPath)
	if err != nil {
		return false
	}
	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func mapSkillsSettingsError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, skillspkg.ErrAgentNotFound):
		return notFoundError(err)
	case errors.Is(err, skillspkg.ErrAgentLocalInvalid):
		return unprocessableError(err)
	default:
		return err
	}
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
