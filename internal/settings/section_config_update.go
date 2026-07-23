package settings

import (
	"context"
	"errors"
	"fmt"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/config/lifecycle"
)

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
	case SectionWindowManager:
		return s.updateWindowManagerSection(ctx, req)
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
	desired := *req.General
	desired.Daemon.ReloadTimeouts = normalizeDaemonReloadTimeouts(desired.Daemon.ReloadTimeouts)
	changed := diffGeneralSettings(&cfg, desired)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyGeneralSettings(editor, desired)
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

func (s *service) updateWindowManagerSection(
	ctx context.Context,
	req SectionUpdateRequest,
) (MutationResult, error) {
	cfg, target, err := s.loadGlobalSectionUpdate(ctx, req.Section, req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	if req.WindowManager == nil {
		return MutationResult{}, validationError(
			errors.New("settings: window-manager section payload is required"),
		)
	}
	if err := req.WindowManager.Validate(); err != nil {
		return MutationResult{}, validationError(err)
	}
	desired := cloneWindowManagerConfig(*req.WindowManager)
	changed := diffWindowManagerSettings(cfg.WindowManager, desired)
	return s.updateConfigSection(req.Section, changed, target, func(editor *aghconfig.OverlayEditor) error {
		return applyWindowManagerSettings(editor, desired)
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
			Section:   section,
			Scope:     ScopeGlobal,
			Behavior:  MutationBehaviorAppliedNow,
			Applied:   true,
			Warnings:  []string{sectionsNoChangesValue},
			Lifecycle: lifecycle.Live,
			DiffClass: lifecycle.DiffClassForRoot(string(section)),
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
		Lifecycle:       classification.Lifecycle,
		DiffClass:       classification.DiffClass,
	}, nil
}
