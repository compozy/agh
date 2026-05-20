package settings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/config/lifecycle"
	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	"github.com/pedronauck/agh/internal/diagnostics"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

const (
	configApplyNoChangesReason = "no_changes_detected"
)

type activeConfigState struct {
	mu          sync.Mutex
	initialized bool
	hash        string
	generation  int64
	config      aghconfig.Config
}

// ApplySection persists a section mutation through the config apply lifecycle.
func (s *service) ApplySection(ctx context.Context, req SectionUpdateRequest) (ApplyResult, error) {
	result, err := s.UpdateSection(ctx, req)
	if err != nil {
		return s.recordFailedApply(ctx, req.Section, req.Scope, "", err)
	}
	return s.recordMutationApply(ctx, result)
}

// ApplyCollectionItem persists a collection upsert through the config apply lifecycle.
func (s *service) ApplyCollectionItem(ctx context.Context, req CollectionItemPutRequest) (ApplyResult, error) {
	before, err := s.collectionItemExistsBeforeMutation(ctx, req.CollectionRequest, req.Name)
	if err != nil {
		return ApplyResult{}, err
	}
	result, err := s.PutCollectionItem(ctx, req)
	if err != nil {
		return s.recordFailedApply(ctx, SectionName(req.Collection), req.Scope, req.WorkspaceID, err)
	}
	result = applyCollectionLifecycle(result, req.Collection, "put", before)
	return s.recordMutationApply(ctx, result)
}

// ApplyCollectionDelete persists a collection deletion through the config apply lifecycle.
func (s *service) ApplyCollectionDelete(
	ctx context.Context,
	req CollectionItemDeleteRequest,
) (ApplyResult, error) {
	result, err := s.DeleteCollectionItem(ctx, req)
	if err != nil {
		return s.recordFailedApply(ctx, SectionName(req.Collection), req.Scope, req.WorkspaceID, err)
	}
	result = applyCollectionLifecycle(result, req.Collection, "delete", true)
	return s.recordMutationApply(ctx, result)
}

// Reload reconciles desired config.toml with the daemon active generation.
func (s *service) Reload(ctx context.Context) (ApplyResult, error) {
	state, err := s.ensureActiveConfigState(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	desiredHash, desiredConfig, err := s.currentDesiredConfigHash()
	if err != nil {
		return s.recordFailedApply(ctx, "", ScopeGlobal, "", err)
	}
	if desiredHash == state.hash {
		record, createErr := s.createTerminalApplyRecord(ctx, applyRecordInput{
			desiredHash: desiredHash,
			activeHash:  state.hash,
			generation:  state.generation,
			lifecycle:   lifecycle.Live,
			status:      lifecycle.StatusApplied,
		})
		if createErr != nil {
			return ApplyResult{}, createErr
		}
		return ApplyResult{
			Record:        record,
			Applied:       true,
			NextAction:    lifecycle.NextActionNone,
			Skipped:       true,
			SkippedReason: configApplyNoChangesReason,
		}, nil
	}

	configLifecycle, err := classifyReloadLifecycle(&state.config, &desiredConfig)
	if err != nil {
		return s.recordFailedApply(ctx, "", ScopeGlobal, "", err)
	}
	status := lifecycle.StatusApplied
	activeHash := desiredHash
	generation := state.generation + 1
	applied := true
	if configLifecycle == lifecycle.RestartRequired {
		status = lifecycle.StatusBlocked
		activeHash = state.hash
		generation = state.generation
		applied = false
	}
	record, err := s.createTerminalApplyRecord(ctx, applyRecordInput{
		desiredHash:  desiredHash,
		activeHash:   activeHash,
		generation:   generation,
		lifecycle:    configLifecycle,
		status:       status,
		diagnostics:  restartRequiredDiagnostics(configLifecycle, status),
		appliedAtNow: applied,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if applied {
		s.advanceActiveConfig(&desiredConfig, desiredHash, generation)
	}
	return ApplyResult{
		Record:          record,
		Applied:         applied,
		NextAction:      lifecycle.NextActionForLifecycle(configLifecycle, status),
		RestartRequired: configLifecycle == lifecycle.RestartRequired,
		RestartScope:    restartScopeForLifecycle(configLifecycle),
	}, nil
}

// ListApplyRecords returns apply history rows.
func (s *service) ListApplyRecords(
	ctx context.Context,
	filter ApplyRecordFilter,
) ([]ApplyRecord, error) {
	if s.applyRecords == nil {
		return nil, errors.New("settings: config apply records are not configured")
	}
	return s.applyRecords.ListApplyRecords(ctx, filter)
}

type applyRecordInput struct {
	desiredHash  string
	activeHash   string
	generation   int64
	lifecycle    lifecycle.Lifecycle
	status       lifecycle.Status
	diagnostics  []diagnosticcontract.DiagnosticItem
	appliedAtNow bool
}

func (s *service) recordMutationApply(ctx context.Context, result MutationResult) (ApplyResult, error) {
	state, err := s.ensureActiveConfigState(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	desiredHash, desiredConfig, err := s.currentDesiredConfigHash()
	if err != nil {
		return ApplyResult{}, err
	}
	configLifecycle := result.Lifecycle
	if configLifecycle == "" {
		configLifecycle = lifecycle.Lifecycle(result.DiffClass)
	}
	if configLifecycle == "" {
		configLifecycle = lifecycle.RestartRequired
	}
	noChanges := mutationResultHasNoChanges(result)
	status := lifecycle.StatusApplied
	activeHash := desiredHash
	generation := state.generation
	applied := true
	if !noChanges {
		generation = state.generation + 1
	}
	if configLifecycle == lifecycle.RestartRequired {
		status = lifecycle.StatusBlocked
		activeHash = state.hash
		generation = state.generation
		applied = false
	}
	record, err := s.createTerminalApplyRecord(ctx, applyRecordInput{
		desiredHash:  desiredHash,
		activeHash:   activeHash,
		generation:   generation,
		lifecycle:    configLifecycle,
		status:       status,
		diagnostics:  restartRequiredDiagnostics(configLifecycle, status),
		appliedAtNow: applied && !noChanges,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if applied && !noChanges {
		s.advanceActiveConfig(&desiredConfig, desiredHash, generation)
	}
	return ApplyResult{
		Record:          record,
		Section:         result.Section,
		Scope:           result.Scope,
		WriteTarget:     result.WriteTarget,
		WorkspaceID:     result.WorkspaceID,
		AgentName:       result.AgentName,
		Applied:         applied,
		NextAction:      lifecycle.NextActionForLifecycle(configLifecycle, status),
		RestartRequired: configLifecycle == lifecycle.RestartRequired,
		RestartScope:    restartScopeForLifecycle(configLifecycle),
		Warnings:        append([]string(nil), result.Warnings...),
		Skipped:         noChanges,
		SkippedReason:   skippedReason(noChanges),
	}, nil
}

func (s *service) recordFailedApply(
	ctx context.Context,
	section SectionName,
	scope ScopeKind,
	workspaceID string,
	cause error,
) (ApplyResult, error) {
	state, stateErr := s.ensureActiveConfigState(ctx)
	if stateErr != nil {
		return ApplyResult{}, stateErr
	}
	desiredHash, _, hashErr := s.currentDesiredConfigHash()
	if hashErr != nil {
		desiredHash = state.hash
	}
	diagnostic := diagnostics.NewItem(
		"config.apply.failed",
		diagnosticcontract.CodeConfigInvalid,
		diagnosticcontract.CategoryConfig,
		"Config apply failed",
		cause.Error(),
		diagnosticcontract.SeverityError,
		diagnosticcontract.FreshnessLive,
		diagnostics.WithSuggestedCommand("agh config validate"),
	)
	record, err := s.createTerminalApplyRecord(ctx, applyRecordInput{
		desiredHash: desiredHash,
		activeHash:  state.hash,
		generation:  state.generation,
		lifecycle:   lifecycle.RestartRequired,
		status:      lifecycle.StatusFailed,
		diagnostics: []diagnosticcontract.DiagnosticItem{diagnostic},
	})
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{
		Record:          record,
		Section:         section,
		Scope:           scope,
		WorkspaceID:     workspaceID,
		Applied:         false,
		NextAction:      lifecycle.NextActionRetry,
		RestartRequired: false,
	}, cause
}

func (s *service) createTerminalApplyRecord(
	ctx context.Context,
	input applyRecordInput,
) (ApplyRecord, error) {
	if s.applyRecords == nil {
		return ApplyRecord{}, errors.New("settings: config apply records are not configured")
	}
	now := time.Now().UTC()
	pending, err := s.applyRecords.CreateApplyRecord(ctx, ApplyRecord{
		DesiredHash: input.desiredHash,
		ActiveHash:  input.activeHash,
		Generation:  input.generation,
		Actor:       mutationSourceFromContext(ctx),
		DiffClass:   lifecycle.DiffClass(input.lifecycle),
		Status:      lifecycle.StatusPendingApply,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return ApplyRecord{}, err
	}
	pending.Status = input.status
	pending.Lifecycle = input.lifecycle
	pending.DiffClass = lifecycle.DiffClass(input.lifecycle)
	pending.NextAction = lifecycle.NextActionForLifecycle(input.lifecycle, input.status)
	pending.Diagnostics = input.diagnostics
	pending.UpdatedAt = time.Now().UTC()
	if input.appliedAtNow {
		appliedAt := pending.UpdatedAt
		pending.AppliedAt = &appliedAt
	}
	return s.applyRecords.UpdateApplyRecord(ctx, pending)
}

func (s *service) ensureActiveConfigState(ctx context.Context) (activeSnapshot, error) {
	s.activeConfig.mu.Lock()
	defer s.activeConfig.mu.Unlock()
	if s.activeConfig.initialized {
		return activeSnapshot{
			hash:       s.activeConfig.hash,
			generation: s.activeConfig.generation,
			config:     cloneActiveConfig(&s.activeConfig.config),
		}, nil
	}

	hash, cfg, err := s.currentDesiredConfigHash()
	if err != nil {
		return activeSnapshot{}, err
	}
	generation := int64(0)
	if s.applyRecords != nil {
		latest, latestErr := s.applyRecords.LatestAppliedRecord(ctx)
		if latestErr != nil {
			return activeSnapshot{}, latestErr
		}
		if latest != nil {
			hash = latest.ActiveHash
			generation = latest.Generation
		}
	}
	s.activeConfig.initialized = true
	s.activeConfig.hash = hash
	s.activeConfig.generation = generation
	s.activeConfig.config = cfg
	return activeSnapshot{hash: hash, generation: generation, config: cloneActiveConfig(&cfg)}, nil
}

type activeSnapshot struct {
	hash       string
	generation int64
	config     aghconfig.Config
}

func (s *service) advanceActiveConfig(cfg *aghconfig.Config, hash string, generation int64) {
	s.activeConfig.mu.Lock()
	defer s.activeConfig.mu.Unlock()
	s.activeConfig.initialized = true
	s.activeConfig.config = cloneActiveConfig(cfg)
	s.activeConfig.hash = hash
	s.activeConfig.generation = generation
}

func (s *service) currentDesiredConfigHash() (string, aghconfig.Config, error) {
	cfg, err := aghconfig.LoadForHome(s.homePaths)
	if err != nil {
		return "", aghconfig.Config{}, fmt.Errorf("settings: load desired config: %w", err)
	}
	hash, err := hashConfigSnapshot(s.homePaths.ConfigFile, &cfg)
	if err != nil {
		return "", aghconfig.Config{}, err
	}
	return hash, cfg, nil
}

func hashConfigSnapshot(path string, cfg *aghconfig.Config) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("settings: read config snapshot %q: %w", path, err)
		}
		bytes, err = json.Marshal(cfg)
		if err != nil {
			return "", fmt.Errorf("settings: marshal default config snapshot: %w", err)
		}
	}
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func mutationResultHasNoChanges(result MutationResult) bool {
	return slices.Contains(result.Warnings, sectionsNoChangesValue)
}

func skippedReason(skipped bool) string {
	if !skipped {
		return ""
	}
	return configApplyNoChangesReason
}

func restartScopeForLifecycle(configLifecycle lifecycle.Lifecycle) string {
	if configLifecycle == lifecycle.RestartRequired {
		return restartScopeDaemon
	}
	return ""
}

func restartRequiredDiagnostics(
	configLifecycle lifecycle.Lifecycle,
	status lifecycle.Status,
) []diagnosticcontract.DiagnosticItem {
	if configLifecycle != lifecycle.RestartRequired || status != lifecycle.StatusBlocked {
		return nil
	}
	return []diagnosticcontract.DiagnosticItem{
		diagnostics.NewItem(
			"config.apply.restart_required",
			diagnosticcontract.CodeConfigRestartRequired,
			diagnosticcontract.CategoryConfig,
			"Daemon restart required",
			"Desired config was written, but the active generation cannot advance until the daemon restarts.",
			diagnosticcontract.SeverityWarn,
			diagnosticcontract.FreshnessLive,
			diagnostics.WithSuggestedCommand("agh daemon restart"),
		),
	}
}

func applyCollectionLifecycle(
	result MutationResult,
	collection CollectionName,
	operation string,
	existedBefore bool,
) MutationResult {
	configLifecycle := lifecycle.RestartRequired
	switch collection {
	case CollectionProviders, CollectionMCPServers:
		if operation == "put" && !existedBefore {
			configLifecycle = lifecycle.LiveAdd
		}
		if operation == "delete" {
			configLifecycle = lifecycle.LiveRemoveIfUnused
		}
	case CollectionSandboxes:
		configLifecycle = lifecycle.SessionRebind
	case CollectionHooks:
		configLifecycle = lifecycle.RestartRequired
	}
	result.Lifecycle = configLifecycle
	result.DiffClass = lifecycle.DiffClass(configLifecycle)
	classification := classificationFromLifecycle(configLifecycle, lifecycle.DiffClass(configLifecycle))
	result.Behavior = classification.Behavior
	result.Applied = classification.Applied
	result.RestartRequired = classification.RestartRequired
	result.RestartScope = classification.RestartScope
	return result
}

func (s *service) collectionItemExistsBeforeMutation(
	ctx context.Context,
	req CollectionRequest,
	name string,
) (bool, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return false, nil
	}
	envelope, err := s.ListCollection(ctx, req)
	if err != nil {
		return false, err
	}
	switch req.Collection {
	case CollectionProviders:
		for i := range envelope.Providers {
			item := &envelope.Providers[i]
			if item.Name == trimmedName {
				return item.SourceMetadata.EffectiveSource.Kind != SourceKindBuiltinProvider, nil
			}
		}
	case CollectionMCPServers:
		for _, item := range envelope.MCPServers {
			if item.Name == trimmedName {
				return true, nil
			}
		}
	case CollectionSandboxes:
		for _, item := range envelope.Sandboxes {
			if item.Name == trimmedName {
				return true, nil
			}
		}
	case CollectionHooks:
		for i := range envelope.Hooks {
			item := &envelope.Hooks[i]
			if item.Name == trimmedName {
				return true, nil
			}
		}
	}
	return false, nil
}

func classifyReloadLifecycle(current *aghconfig.Config, desired *aghconfig.Config) (lifecycle.Lifecycle, error) {
	changed := reloadChangedPaths(current, desired)
	if len(changed) == 0 {
		return lifecycle.Live, nil
	}
	configLifecycle, _, err := lifecycle.ClassifyPaths(changed)
	return configLifecycle, err
}

func reloadChangedPaths(current *aghconfig.Config, desired *aghconfig.Config) []string {
	var changed []string
	changed = append(changed, diffGeneralSettings(current, generalSettingsFromConfig(desired))...)
	changed = append(changed, diffSkillsSettings(current.Skills, desired.Skills)...)
	changed = append(changed, diffMemorySettings(&current.Memory, &desired.Memory)...)
	changed = append(changed, diffAutomationSettings(current, automationSettingsFromConfig(desired))...)
	changed = append(changed, diffNetworkSettings(current.Network, desired.Network)...)
	changed = append(changed, diffObservabilitySettings(current.Observability, desired.Observability)...)
	changed = append(changed, diffExtensionsSettings(current.Extensions, desired.Extensions)...)
	if !reflect.DeepEqual(current.Providers, desired.Providers) {
		changed = append(changed, "providers.*")
	}
	if !reflect.DeepEqual(current.MCPServers, desired.MCPServers) {
		changed = append(changed, "mcp-servers.*")
	}
	if !reflect.DeepEqual(current.Sandboxes, desired.Sandboxes) {
		changed = append(changed, "sandboxes.*")
	}
	if !reflect.DeepEqual(current.Hooks.Declarations, desired.Hooks.Declarations) {
		changed = append(changed, "hooks.*")
	}
	return changed
}

func generalSettingsFromConfig(cfg *aghconfig.Config) GeneralSettings {
	return GeneralSettings{
		Defaults:       cfg.Defaults,
		Limits:         cfg.Limits,
		Permissions:    cfg.Permissions,
		SessionTimeout: cfg.Session.Limits.Timeout,
		HTTP:           cfg.HTTP,
		Daemon:         cfg.Daemon,
	}
}

func automationSettingsFromConfig(cfg *aghconfig.Config) AutomationSettings {
	return AutomationSettings{
		Enabled:           cfg.Automation.Enabled,
		Timezone:          cfg.Automation.Timezone,
		MaxConcurrentJobs: cfg.Automation.MaxConcurrentJobs,
		DefaultFireLimit:  cfg.Automation.DefaultFireLimit,
	}
}

func cloneActiveConfig(cfg *aghconfig.Config) aghconfig.Config {
	cloned := *cfg
	cloned.Providers = mapsClone(cfg.Providers)
	cloned.Sandboxes = mapsClone(cfg.Sandboxes)
	cloned.MCPServers = append([]aghconfig.MCPServer(nil), cfg.MCPServers...)
	cloned.Hooks.Declarations = append([]hookspkg.HookDecl(nil), cfg.Hooks.Declarations...)
	return cloned
}

func mapsClone[K comparable, V any](source map[K]V) map[K]V {
	return maps.Clone(source)
}
