package settings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/config/lifecycle"
	diagnosticcontract "github.com/compozy/agh/internal/diagnosticcontract"
	"github.com/compozy/agh/internal/diagnostics"
	hookspkg "github.com/compozy/agh/internal/hooks"
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
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	configLifecycle := s.classifySectionApplyRequest(ctx, req)
	result, err := s.UpdateSection(ctx, req)
	if err != nil {
		return s.recordFailedApply(ctx, req.Section, req.Scope, "", configLifecycle, err)
	}
	return s.recordMutationApply(ctx, result)
}

// ApplyCollectionItem persists a collection upsert through the config apply lifecycle.
func (s *service) ApplyCollectionItem(ctx context.Context, req CollectionItemPutRequest) (ApplyResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	before, err := s.collectionItemExistsBeforeMutation(ctx, req.CollectionRequest, req.Name)
	if err != nil {
		return ApplyResult{}, err
	}
	expected := applyCollectionLifecycle(MutationResult{}, req.Collection, collectionMutationPut, before)
	result, err := s.PutCollectionItem(ctx, req)
	if err != nil {
		return s.recordFailedApply(
			ctx,
			SectionName(req.Collection),
			req.Scope,
			req.WorkspaceID,
			expected.Lifecycle,
			err,
		)
	}
	result = applyCollectionLifecycle(result, req.Collection, collectionMutationPut, before)
	if req.Collection == CollectionProviders &&
		result.Lifecycle == lifecycle.Live &&
		!mutationResultHasNoChanges(result) {
		return s.recordProviderModelsMutationApply(ctx, result, req.Name)
	}
	return s.recordMutationApply(ctx, result)
}

// ApplyCollectionDelete persists a collection deletion through the config apply lifecycle.
func (s *service) ApplyCollectionDelete(
	ctx context.Context,
	req CollectionItemDeleteRequest,
) (ApplyResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	expected := applyCollectionLifecycle(MutationResult{}, req.Collection, collectionMutationDelete, true)
	result, err := s.DeleteCollectionItem(ctx, req)
	if err != nil {
		return s.recordFailedApply(
			ctx,
			SectionName(req.Collection),
			req.Scope,
			req.WorkspaceID,
			expected.Lifecycle,
			err,
		)
	}
	result = applyCollectionLifecycle(result, req.Collection, collectionMutationDelete, true)
	return s.recordMutationApply(ctx, result)
}

// Reload reconciles desired config.toml with the daemon active generation.
func (s *service) Reload(ctx context.Context) (ApplyResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	state, err := s.ensureActiveConfigState(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	desiredHash, desiredConfig, err := s.currentDesiredConfigHash()
	if err != nil {
		return s.recordFailedApply(ctx, "", ScopeGlobal, "", lifecycle.RestartRequired, err)
	}
	if desiredHash == state.hash {
		return skippedReloadResult(&state, desiredHash), nil
	}

	configLifecycle := classifyReloadLifecycle(&state.config, &desiredConfig)
	record, plan, err := s.persistRuntimeApply(
		ctx,
		&state,
		desiredHash,
		desiredHash,
		&desiredConfig,
		configLifecycle,
		false,
	)
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{
		Record:          record,
		Applied:         plan.applied,
		NextAction:      lifecycle.NextActionForLifecycle(configLifecycle, plan.status),
		RestartRequired: configLifecycle == lifecycle.RestartRequired,
		RestartScope:    restartScopeForLifecycle(configLifecycle),
		PartialFailures: plan.partialFailures,
	}, nil
}

// ActiveConfig returns the daemon's last successfully applied config generation.
func (s *service) ActiveConfig(ctx context.Context) (aghconfig.Config, error) {
	state, err := s.ensureActiveConfigState(ctx)
	if err != nil {
		return aghconfig.Config{}, err
	}
	return state.config, nil
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

type runtimeApplyPlan struct {
	status          lifecycle.Status
	activeHash      string
	generation      int64
	applied         bool
	partialFailures []ApplyFailure
	diagnostics     []diagnosticcontract.DiagnosticItem
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
	return s.recordProjectedMutationApply(
		ctx,
		result,
		&state,
		desiredHash,
		desiredHash,
		&desiredConfig,
	)
}

func (s *service) recordFailedApply(
	ctx context.Context,
	section SectionName,
	scope ScopeKind,
	workspaceID string,
	configLifecycle lifecycle.Lifecycle,
	cause error,
) (ApplyResult, error) {
	state, stateErr := s.ensureActiveConfigState(ctx)
	if stateErr != nil {
		return ApplyResult{}, stateErr
	}
	if configLifecycle == "" {
		configLifecycle = lifecycle.RestartRequired
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
		lifecycle:   configLifecycle,
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
	pending, err := s.createPendingApplyRecord(ctx, input)
	if err != nil {
		return ApplyRecord{}, err
	}
	return s.finalizeApplyRecord(ctx, pending, input)
}

func (s *service) createPendingApplyRecord(
	ctx context.Context,
	input applyRecordInput,
) (ApplyRecord, error) {
	if s.applyRecords == nil {
		return ApplyRecord{}, errors.New("settings: config apply records are not configured")
	}
	now := time.Now().UTC()
	return s.applyRecords.CreateApplyRecord(ctx, ApplyRecord{
		DesiredHash: input.desiredHash,
		ActiveHash:  input.activeHash,
		Generation:  input.generation,
		Actor:       mutationSourceFromContext(ctx),
		DiffClass:   lifecycle.DiffClass(input.lifecycle),
		Status:      lifecycle.StatusPendingApply,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (s *service) finalizeApplyRecord(
	ctx context.Context,
	pending ApplyRecord,
	input applyRecordInput,
) (ApplyRecord, error) {
	if s.applyRecords == nil {
		return ApplyRecord{}, errors.New("settings: config apply records are not configured")
	}
	pending.Status = input.status
	pending.Lifecycle = input.lifecycle
	pending.DiffClass = lifecycle.DiffClass(input.lifecycle)
	pending.NextAction = lifecycle.NextActionForLifecycle(input.lifecycle, input.status)
	pending.DesiredHash = input.desiredHash
	pending.ActiveHash = input.activeHash
	pending.Generation = input.generation
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
			generation = latest.Generation
			// A daemon boot makes the current file active even when the last applied record predates restart.
			if strings.TrimSpace(latest.ActiveHash) != "" && latest.ActiveHash != hash {
				generation++
			}
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
	hash, err := hashConfigSnapshot(&cfg)
	if err != nil {
		return "", aghconfig.Config{}, err
	}
	return hash, cfg, nil
}

func hashConfigSnapshot(cfg *aghconfig.Config) (string, error) {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("settings: marshal config snapshot: %w", err)
	}
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func mutationResultHasNoChanges(result MutationResult) bool {
	return slices.Contains(result.Warnings, sectionsNoChangesValue)
}

func mutationLifecycle(result MutationResult) lifecycle.Lifecycle {
	if result.Lifecycle != "" {
		return result.Lifecycle
	}
	if result.DiffClass != "" {
		return lifecycle.Lifecycle(result.DiffClass)
	}
	return lifecycle.RestartRequired
}

func skippedReason(skipped bool) string {
	if !skipped {
		return ""
	}
	return configApplyNoChangesReason
}

func skippedReloadResult(state *activeSnapshot, desiredHash string) ApplyResult {
	return ApplyResult{
		Record: ApplyRecord{
			DesiredHash: desiredHash,
			ActiveHash:  state.hash,
			Generation:  state.generation,
			Lifecycle:   lifecycle.Live,
			DiffClass:   lifecycle.DiffClassLive,
			Status:      lifecycle.StatusApplied,
			NextAction:  lifecycle.NextActionNone,
		},
		Applied:       true,
		NextAction:    lifecycle.NextActionNone,
		Skipped:       true,
		SkippedReason: configApplyNoChangesReason,
	}
}

func newRuntimeApplyPlan(
	state *activeSnapshot,
	desiredHash string,
	configLifecycle lifecycle.Lifecycle,
	noChanges bool,
) runtimeApplyPlan {
	plan := runtimeApplyPlan{
		status:     lifecycle.StatusApplied,
		activeHash: desiredHash,
		generation: state.generation,
		applied:    true,
	}
	if !noChanges {
		plan.generation = state.generation + 1
	}
	if configLifecycle == lifecycle.RestartRequired {
		plan.status = lifecycle.StatusBlocked
		plan.activeHash = state.hash
		plan.generation = state.generation
		plan.applied = false
	}
	return plan
}

func (s *service) persistRuntimeApply(
	ctx context.Context,
	state *activeSnapshot,
	desiredHash string,
	nextActiveHash string,
	nextActiveConfig *aghconfig.Config,
	configLifecycle lifecycle.Lifecycle,
	noChanges bool,
) (ApplyRecord, runtimeApplyPlan, error) {
	plan := newRuntimeApplyPlan(state, nextActiveHash, configLifecycle, noChanges)
	pending, err := s.createPendingApplyRecord(ctx, applyRecordInput{
		desiredHash: desiredHash,
		activeHash:  state.hash,
		generation:  state.generation,
		lifecycle:   configLifecycle,
	})
	if err != nil {
		return ApplyRecord{}, runtimeApplyPlan{}, err
	}
	if plan.applied && !noChanges {
		plan.partialFailures = s.reconcileRuntimeConfig(ctx, nextActiveConfig, configLifecycle)
		if len(plan.partialFailures) > 0 {
			plan.status = lifecycle.StatusFailed
			plan.activeHash = state.hash
			plan.generation = state.generation
			plan.applied = false
			plan.diagnostics = diagnosticsFromApplyFailures(plan.partialFailures)
		}
	}
	if len(plan.diagnostics) == 0 {
		plan.diagnostics = restartRequiredDiagnostics(configLifecycle, plan.status)
	}
	record, err := s.finalizeApplyRecord(ctx, pending, applyRecordInput{
		desiredHash:  desiredHash,
		activeHash:   plan.activeHash,
		generation:   plan.generation,
		lifecycle:    configLifecycle,
		status:       plan.status,
		diagnostics:  plan.diagnostics,
		appliedAtNow: plan.applied && !noChanges,
	})
	if err != nil {
		return ApplyRecord{}, runtimeApplyPlan{}, err
	}
	if plan.applied && !noChanges {
		s.advanceActiveConfig(nextActiveConfig, nextActiveHash, plan.generation)
	}
	return record, plan, nil
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

func diagnosticsFromApplyFailures(
	failures []ApplyFailure,
) []diagnosticcontract.DiagnosticItem {
	if len(failures) == 0 {
		return nil
	}
	items := make([]diagnosticcontract.DiagnosticItem, 0, len(failures))
	for _, failure := range failures {
		items = append(items, failure.Diagnostic)
	}
	return items
}

func (s *service) reconcileRuntimeConfig(
	ctx context.Context,
	desired *aghconfig.Config,
	configLifecycle lifecycle.Lifecycle,
) []ApplyFailure {
	if desired == nil || s.runtimeApplier == nil || !requiresRuntimeReconcile(configLifecycle) {
		return nil
	}
	snapshot := cloneActiveConfig(desired)
	return s.runtimeApplier.ApplyActiveConfig(ctx, &snapshot)
}

func requiresRuntimeReconcile(configLifecycle lifecycle.Lifecycle) bool {
	switch configLifecycle {
	case lifecycle.Live, lifecycle.LiveAdd, lifecycle.LiveRemoveIfUnused, lifecycle.SessionRebind:
		return true
	default:
		return false
	}
}

func (s *service) classifySectionApplyRequest(
	ctx context.Context,
	req SectionUpdateRequest,
) lifecycle.Lifecycle {
	switch req.Section {
	case SectionSkills:
		if req.Skills == nil {
			return lifecycle.Live
		}
		return s.classifySkillsRequest(ctx, req)
	case SectionGeneral:
		if req.General == nil {
			return lifecycle.RestartRequired
		}
		return s.classifyGeneralRequest(ctx, req)
	case SectionHooksExtensions:
		if req.HooksExtensions == nil {
			return lifecycle.RestartRequired
		}
		return s.classifyHooksExtensionsRequest(ctx, req)
	case SectionNetwork:
		if req.Network == nil {
			return lifecycle.RestartRequired
		}
		return s.classifyNetworkRequest(ctx, req)
	default:
		return lifecycle.RestartRequired
	}
}

func lifecycleForChangedPaths(paths []string, fallback lifecycle.Lifecycle) lifecycle.Lifecycle {
	if len(paths) == 0 {
		return lifecycle.Live
	}
	configLifecycle, _, err := lifecycle.ClassifyPaths(paths)
	if err != nil {
		return fallback
	}
	return configLifecycle
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
				return true, nil
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
	cloned.Providers = aghconfig.CloneProviderConfigs(cfg.Providers)
	cloned.Sandboxes = mapsClone(cfg.Sandboxes)
	cloned.MCPServers = append([]aghconfig.MCPServer(nil), cfg.MCPServers...)
	cloned.Hooks.Declarations = append([]hookspkg.HookDecl(nil), cfg.Hooks.Declarations...)
	return cloned
}

func mapsClone[K comparable, V any](source map[K]V) map[K]V {
	return maps.Clone(source)
}
