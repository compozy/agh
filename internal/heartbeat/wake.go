package heartbeat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	// SyntheticReasonHeartbeatWake marks daemon-owned prompts emitted by the Heartbeat wake service.
	SyntheticReasonHeartbeatWake = "agent_heartbeat_wake"
)

var (
	// ErrSyntheticPromptBusy reports that the session prompt gate rejected a no-queue wake.
	ErrSyntheticPromptBusy = errors.New("heartbeat: synthetic prompt busy")
)

// WakeService evaluates Heartbeat policy and session health before issuing advisory wakes.
type WakeService interface {
	Wake(ctx context.Context, req WakeRequest) (WakeDecision, error)
	WakeMany(ctx context.Context, requests []WakeRequest) ([]WakeDecision, error)
}

// WakeStore provides persisted Heartbeat policy, wake state, and wake audit rows.
type WakeStore interface {
	GetLatestValidHeartbeatSnapshot(ctx context.Context, workspaceID string, agentName string) (Snapshot, error)
	GetHeartbeatWakeState(
		ctx context.Context,
		workspaceID string,
		agentName string,
		sessionID string,
	) (WakeState, error)
	UpsertHeartbeatWakeState(ctx context.Context, state WakeState) (WakeState, error)
	AppendHeartbeatWakeEvent(ctx context.Context, event WakeEvent) (WakeEvent, error)
}

// SyntheticWakePrompter injects one daemon-owned synthetic wake prompt through the session path.
type SyntheticWakePrompter interface {
	PromptHeartbeatWake(ctx context.Context, req SyntheticWakePromptRequest) (SyntheticWakePromptResult, error)
}

// WakeRequest identifies one advisory Heartbeat wake decision.
type WakeRequest struct {
	WorkspaceID string
	AgentName   string
	SessionID   string
	Source      WakeSource
	DryRun      bool
}

// WakeDecision reports the auditable result of one Heartbeat wake decision.
type WakeDecision struct {
	WakeEventID       string
	Result            WakeResult
	Reason            WakeReason
	PolicySnapshotID  string
	PolicyDigest      string
	ConfigDigest      string
	SyntheticPromptID string
	Diagnostics       []Diagnostic
}

// SyntheticWakePromptRequest carries the prompt input and stable Heartbeat correlation metadata.
type SyntheticWakePromptRequest struct {
	SessionID        string
	Message          string
	TurnID           string
	WakeEventID      string
	PolicySnapshotID string
	PolicyDigest     string
	ConfigDigest     string
	Summary          string
}

// SyntheticWakePromptResult reports the session prompt turn selected for a sent wake.
type SyntheticWakePromptResult struct {
	SyntheticPromptID string
}

// ManagedWakeService is the production Heartbeat wake decision engine.
type ManagedWakeService struct {
	store        WakeStore
	healthReader SessionHealthReader
	prompter     SyntheticWakePrompter
	config       aghconfig.HeartbeatConfig
	now          func() time.Time
	newID        func(prefix string) string
	mu           sync.Mutex
}

var _ WakeService = (*ManagedWakeService)(nil)

// WakeOption customizes the managed Heartbeat wake service.
type WakeOption func(*ManagedWakeService)

// WithWakeClock injects deterministic time for wake decisions.
func WithWakeClock(clock func() time.Time) WakeOption {
	return func(service *ManagedWakeService) {
		if clock != nil {
			service.now = clock
		}
	}
}

// WithWakeIDGenerator injects deterministic wake event and prompt turn IDs.
func WithWakeIDGenerator(generator func(prefix string) string) WakeOption {
	return func(service *ManagedWakeService) {
		if generator != nil {
			service.newID = generator
		}
	}
}

// NewManagedWakeService creates the production Heartbeat wake service.
func NewManagedWakeService(
	store WakeStore,
	healthReader SessionHealthReader,
	prompter SyntheticWakePrompter,
	config aghconfig.HeartbeatConfig,
	options ...WakeOption,
) (*ManagedWakeService, error) {
	if store == nil {
		return nil, errors.New("heartbeat: wake store is required")
	}
	if healthReader == nil {
		return nil, errors.New("heartbeat: session health reader is required")
	}
	if prompter == nil {
		return nil, errors.New("heartbeat: synthetic wake prompter is required")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	service := &ManagedWakeService{
		store:        store,
		healthReader: healthReader,
		prompter:     prompter,
		config:       config,
		now:          time.Now,
		newID:        defaultWakeID,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service, nil
}

// Wake evaluates one policy/health decision and emits a synthetic prompt only when eligible.
func (s *ManagedWakeService) Wake(ctx context.Context, req WakeRequest) (WakeDecision, error) {
	if ctx == nil {
		return WakeDecision{}, errors.New("heartbeat: wake context is required")
	}
	normalized, err := normalizeWakeRequest(req)
	if err != nil {
		return WakeDecision{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wakeLocked(ctx, normalized)
}

// WakeMany evaluates a scheduler cycle and applies the configured max wake bound.
func (s *ManagedWakeService) WakeMany(ctx context.Context, requests []WakeRequest) ([]WakeDecision, error) {
	if ctx == nil {
		return nil, errors.New("heartbeat: wake context is required")
	}
	decisions := make([]WakeDecision, 0, len(requests))
	sent := 0
	var errs []error
	for _, req := range requests {
		if sent >= s.config.MaxWakesPerCycle {
			decision, err := s.recordRateLimited(ctx, req)
			if err != nil {
				errs = append(errs, err)
				decisions = append(decisions, s.failedWakeManyDecision(err))
				continue
			}
			decisions = append(decisions, decision)
			continue
		}
		decision, err := s.Wake(ctx, req)
		if err != nil {
			errs = append(errs, err)
			decisions = append(decisions, s.failedWakeManyDecision(err))
			continue
		}
		if decision.Result == WakeResultSent {
			sent++
		}
		decisions = append(decisions, decision)
	}
	return decisions, errors.Join(errs...)
}

func (s *ManagedWakeService) failedWakeManyDecision(err error) WakeDecision {
	decision := s.newDecision(WakeResultFailed, WakeReasonSyntheticPromptFailed, "", "", "")
	if err != nil {
		decision.Diagnostics = []Diagnostic{{
			Code:     "heartbeat_wake_error",
			Severity: diagnosticError,
			Message:  err.Error(),
		}}
	}
	return decision
}

func (s *ManagedWakeService) wakeLocked(ctx context.Context, req WakeRequest) (WakeDecision, error) {
	now := s.currentTime()
	if err := ctx.Err(); err != nil {
		return WakeDecision{}, err
	}
	if !s.config.Enabled {
		decision := s.newDecision(WakeResultSkipped, WakeReasonHeartbeatDisabled, "", "", "")
		return s.recordDecision(ctx, req, decision, WakeState{}, now)
	}

	snapshot, envelope, decision, hasPolicy, err := s.loadWakePolicy(ctx, req)
	if err != nil {
		return WakeDecision{}, err
	}
	state, stateErr := s.currentWakeState(ctx, req)
	if stateErr != nil {
		return WakeDecision{}, stateErr
	}
	if !hasPolicy {
		return s.recordDecision(ctx, req, decision, state, now)
	}
	if decision, done := s.policySkipDecision(snapshot, &envelope, now); done {
		return s.recordDecision(ctx, req, decision, state, now)
	}

	if !state.NextAllowedAt.IsZero() && state.NextAllowedAt.After(now) {
		result, reason := cooldownDecisionForSource(req.Source)
		decision := s.newDecision(result, reason, snapshot.ID, snapshot.Digest, snapshot.ConfigDigest)
		return s.recordDecision(ctx, req, decision, state, now)
	}

	healthDecision, done, err := s.healthSkipDecision(ctx, req, snapshot)
	if err != nil {
		return WakeDecision{}, err
	}
	if done {
		return s.recordDecision(ctx, req, healthDecision, state, now)
	}

	return s.dispatchWakePrompt(ctx, req, snapshot, &envelope, state, now)
}

func (s *ManagedWakeService) policySkipDecision(
	snapshot Snapshot,
	envelope *SnapshotEnvelope,
	now time.Time,
) (WakeDecision, bool) {
	if err := s.ensureCurrentConfigDigest(envelope, snapshot); err != nil {
		decision := s.newDecision(
			WakeResultSkipped,
			WakeReasonHeartbeatInvalid,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		)
		decision.Diagnostics = []Diagnostic{{
			Code:     "heartbeat_config_digest_stale",
			Severity: diagnosticWarning,
			Message:  err.Error(),
			Field:    "config_digest",
		}}
		return decision, true
	}
	if envelope == nil || !envelope.Active || !envelope.Prompt.Active {
		return s.newDecision(
			WakeResultSkipped,
			WakeReasonHeartbeatDisabled,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		), true
	}
	allowed, err := envelope.Preferences.AllowsAt(now)
	if err != nil {
		decision := s.newDecision(
			WakeResultSkipped,
			WakeReasonHeartbeatInvalid,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		)
		decision.Diagnostics = []Diagnostic{diagnosticForError(
			"heartbeat_time_window_invalid",
			snapshot.SourcePath,
			err,
			ErrInvalid,
		)}
		return decision, true
	}
	if !allowed {
		return s.newDecision(
			WakeResultSkipped,
			WakeReasonQuietWindow,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		), true
	}
	return WakeDecision{}, false
}

func (s *ManagedWakeService) healthSkipDecision(
	ctx context.Context,
	req WakeRequest,
	snapshot Snapshot,
) (WakeDecision, bool, error) {
	health, err := s.healthReader.GetSessionHealth(ctx, req.SessionID)
	if err != nil {
		if errors.Is(err, ErrSessionHealthNotFound) {
			decision := s.newDecision(
				WakeResultSkipped,
				WakeReasonSessionNotFound,
				snapshot.ID,
				snapshot.Digest,
				snapshot.ConfigDigest,
			)
			return decision, true, nil
		}
		return WakeDecision{}, false, fmt.Errorf("heartbeat: read session health for %q: %w", req.SessionID, err)
	}
	if reason := ineligibleWakeReason(req, health); reason != "" {
		decision := s.newDecision(
			WakeResultSkipped,
			reason,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		)
		return decision, true, nil
	}
	return WakeDecision{}, false, nil
}

func (s *ManagedWakeService) dispatchWakePrompt(
	ctx context.Context,
	req WakeRequest,
	snapshot Snapshot,
	envelope *SnapshotEnvelope,
	state WakeState,
	now time.Time,
) (WakeDecision, error) {
	decision := s.newDecision(WakeResultSent, WakeReasonSent, snapshot.ID, snapshot.Digest, snapshot.ConfigDigest)
	if req.DryRun {
		return dryRunDecision(decision), nil
	}
	promptResult, promptErr := s.prompter.PromptHeartbeatWake(ctx, SyntheticWakePromptRequest{
		SessionID:        req.SessionID,
		Message:          heartbeatWakePrompt(envelope),
		TurnID:           decision.SyntheticPromptID,
		WakeEventID:      decision.WakeEventID,
		PolicySnapshotID: snapshot.ID,
		PolicyDigest:     snapshot.Digest,
		ConfigDigest:     snapshot.ConfigDigest,
		Summary:          envelope.Summary,
	})
	if promptErr != nil {
		result := WakeResultFailed
		reason := WakeReasonSyntheticPromptFailed
		if errors.Is(promptErr, ErrSyntheticPromptBusy) {
			result = WakeResultSkipped
			reason = WakeReasonSessionPromptRace
		}
		decision = s.newDecision(result, reason, snapshot.ID, snapshot.Digest, snapshot.ConfigDigest)
		decision.Diagnostics = []Diagnostic{{
			Code:     string(reason),
			Severity: diagnosticWarning,
			Message:  promptErr.Error(),
		}}
		return s.recordDecision(ctx, req, decision, state, now)
	}
	if promptID := strings.TrimSpace(promptResult.SyntheticPromptID); promptID != "" {
		decision.SyntheticPromptID = promptID
	}
	return s.recordDecision(ctx, req, decision, state, now)
}

func (s *ManagedWakeService) loadWakePolicy(
	ctx context.Context,
	req WakeRequest,
) (Snapshot, SnapshotEnvelope, WakeDecision, bool, error) {
	snapshot, err := s.store.GetLatestValidHeartbeatSnapshot(ctx, req.WorkspaceID, req.AgentName)
	if err != nil {
		if errors.Is(err, ErrSnapshotNotFound) {
			decision := s.newDecision(WakeResultSkipped, WakeReasonHeartbeatNoPolicy, "", "", "")
			return Snapshot{}, SnapshotEnvelope{}, decision, false, nil
		}
		return Snapshot{}, SnapshotEnvelope{}, WakeDecision{}, false, fmt.Errorf(
			"heartbeat: load latest valid snapshot for %q/%q: %w",
			req.WorkspaceID,
			req.AgentName,
			err,
		)
	}
	envelope, err := snapshot.ResolvedEnvelope()
	if err != nil {
		decision := s.newDecision(
			WakeResultSkipped,
			WakeReasonHeartbeatInvalid,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		)
		decision.Diagnostics = []Diagnostic{{
			Code:     "heartbeat_snapshot_invalid",
			Severity: diagnosticError,
			Message:  err.Error(),
		}}
		return snapshot, SnapshotEnvelope{}, decision, false, nil
	}
	if !envelope.Valid {
		decision := s.newDecision(
			WakeResultSkipped,
			WakeReasonHeartbeatInvalid,
			snapshot.ID,
			snapshot.Digest,
			snapshot.ConfigDigest,
		)
		decision.Diagnostics = cloneDiagnostics(envelope.Diagnostics)
		return snapshot, envelope, decision, false, nil
	}
	return snapshot, envelope, WakeDecision{}, true, nil
}

func (s *ManagedWakeService) ensureCurrentConfigDigest(envelope *SnapshotEnvelope, snapshot Snapshot) error {
	provenance, err := ConfigProvenanceFor(s.config)
	if err != nil {
		return err
	}
	current := strings.TrimSpace(provenance.Digest)
	envelopeDigest := ""
	if envelope != nil {
		envelopeDigest = envelope.ConfigProvenance.Digest
	}
	stored := strings.TrimSpace(firstNonEmpty(snapshot.ConfigDigest, envelopeDigest))
	if current == "" || stored == "" || current == stored {
		return nil
	}
	return fmt.Errorf("heartbeat policy config digest is stale: current %s differs from snapshot %s", current, stored)
}

func (s *ManagedWakeService) currentWakeState(ctx context.Context, req WakeRequest) (WakeState, error) {
	state, err := s.store.GetHeartbeatWakeState(ctx, req.WorkspaceID, req.AgentName, req.SessionID)
	if err != nil {
		if errors.Is(err, ErrWakeStateNotFound) {
			return WakeState{}, nil
		}
		return WakeState{}, fmt.Errorf("heartbeat: get wake state for %q: %w", req.SessionID, err)
	}
	return state, nil
}

func (s *ManagedWakeService) recordRateLimited(ctx context.Context, req WakeRequest) (WakeDecision, error) {
	if ctx == nil {
		return WakeDecision{}, errors.New("heartbeat: wake context is required")
	}
	normalized, err := normalizeWakeRequest(req)
	if err != nil {
		return WakeDecision{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.currentTime()
	state, stateErr := s.currentWakeState(ctx, normalized)
	if stateErr != nil {
		return WakeDecision{}, stateErr
	}
	decision := s.newDecision(WakeResultRateLimited, WakeReasonHeartbeatRateLimited, "", "", "")
	return s.recordDecision(ctx, normalized, decision, state, now)
}

func (s *ManagedWakeService) recordDecision(
	ctx context.Context,
	req WakeRequest,
	decision WakeDecision,
	previous WakeState,
	now time.Time,
) (WakeDecision, error) {
	if req.DryRun {
		return dryRunDecision(decision), nil
	}
	if strings.TrimSpace(decision.WakeEventID) == "" {
		decision.WakeEventID = s.newID("hwe")
	}
	if decision.Result == WakeResultSent && strings.TrimSpace(decision.SyntheticPromptID) == "" {
		decision.SyntheticPromptID = s.newID("turn")
	}
	event := WakeEvent{
		ID:                decision.WakeEventID,
		WorkspaceID:       req.WorkspaceID,
		AgentName:         req.AgentName,
		SessionID:         req.SessionID,
		PolicySnapshotID:  decision.PolicySnapshotID,
		Source:            req.Source,
		Result:            decision.Result,
		Reason:            decision.Reason,
		SyntheticPromptID: decision.SyntheticPromptID,
		CreatedAt:         now,
		ExpiresAt:         now.Add(s.config.WakeEventRetention),
	}
	if _, err := s.store.AppendHeartbeatWakeEvent(ctx, event); err != nil {
		return WakeDecision{}, fmt.Errorf("heartbeat: append wake event %q: %w", event.ID, err)
	}
	state := nextWakeState(req, decision, previous, now, s.config.WakeCooldown)
	if _, err := s.store.UpsertHeartbeatWakeState(ctx, state); err != nil {
		return WakeDecision{}, fmt.Errorf("heartbeat: upsert wake state for %q: %w", req.SessionID, err)
	}
	return decision, nil
}

func dryRunDecision(decision WakeDecision) WakeDecision {
	decision.WakeEventID = ""
	decision.SyntheticPromptID = ""
	return decision
}

func nextWakeState(
	req WakeRequest,
	decision WakeDecision,
	previous WakeState,
	now time.Time,
	cooldown time.Duration,
) WakeState {
	state := previous.Normalize()
	state.WorkspaceID = req.WorkspaceID
	state.AgentName = req.AgentName
	state.SessionID = req.SessionID
	state.PolicySnapshotID = decision.PolicySnapshotID
	state.LastResult = decision.Result
	state.LastReason = decision.Reason
	state.UpdatedAt = now
	switch decision.Result {
	case WakeResultSent:
		state.LastWakeAt = now
		state.NextAllowedAt = now.Add(cooldown)
		state.CoalescedCount = 0
	case WakeResultCoalesced:
		state.CoalescedCount++
	}
	return state
}

func (s *ManagedWakeService) newDecision(
	result WakeResult,
	reason WakeReason,
	snapshotID string,
	policyDigest string,
	configDigest string,
) WakeDecision {
	decision := WakeDecision{
		WakeEventID:      s.newID("hwe"),
		Result:           result,
		Reason:           reason,
		PolicySnapshotID: strings.TrimSpace(snapshotID),
		PolicyDigest:     strings.TrimSpace(policyDigest),
		ConfigDigest:     strings.TrimSpace(configDigest),
		Diagnostics:      nil,
	}
	if result == WakeResultSent {
		decision.SyntheticPromptID = s.newID("turn")
	}
	return decision
}

func (s *ManagedWakeService) currentTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now().UTC()
	}
	now := s.now()
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func normalizeWakeRequest(req WakeRequest) (WakeRequest, error) {
	normalized := WakeRequest{
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		AgentName:   strings.TrimSpace(req.AgentName),
		SessionID:   strings.TrimSpace(req.SessionID),
		Source:      WakeSource(strings.TrimSpace(string(req.Source))),
		DryRun:      req.DryRun,
	}
	switch {
	case normalized.WorkspaceID == "":
		return WakeRequest{}, errors.New("heartbeat: wake workspace id is required")
	case normalized.AgentName == "":
		return WakeRequest{}, errors.New("heartbeat: wake agent name is required")
	case normalized.SessionID == "":
		return WakeRequest{}, errors.New("heartbeat: wake session id is required")
	case !ValidWakeSource(normalized.Source):
		return WakeRequest{}, fmt.Errorf("heartbeat: invalid wake source %q", normalized.Source)
	default:
		return normalized, nil
	}
}

func cooldownDecisionForSource(source WakeSource) (WakeResult, WakeReason) {
	switch source {
	case WakeSourceManual:
		return WakeResultRateLimited, WakeReasonCooldownActive
	default:
		return WakeResultCoalesced, WakeReasonCoalesced
	}
}

func ineligibleWakeReason(req WakeRequest, health SessionHealth) WakeReason {
	normalized := health.Normalize()
	if normalized.WorkspaceID != req.WorkspaceID || normalized.AgentName != req.AgentName {
		return WakeReasonHeartbeatNoEligible
	}
	if normalized.EligibleForWake {
		return ""
	}
	switch SessionHealthIneligibilityReason(normalized.IneligibilityReason) {
	case SessionHealthReasonPromptActive:
		return WakeReasonSessionPromptActive
	case SessionHealthReasonNotAttachable:
		return WakeReasonSessionNotAttachable
	case SessionHealthReasonStale,
		SessionHealthReasonHung,
		SessionHealthReasonDead,
		SessionHealthReasonUnknown,
		SessionHealthReasonUnhealthy:
		return WakeReasonSessionUnhealthy
	default:
		return WakeReasonHeartbeatNoEligible
	}
}

func heartbeatWakePrompt(envelope *SnapshotEnvelope) string {
	if envelope == nil {
		return "Agent Heartbeat requested a reorientation for this existing session."
	}
	summary := strings.TrimSpace(envelope.Prompt.Summary)
	if summary == "" {
		summary = strings.TrimSpace(envelope.Summary)
	}
	guidance := strings.TrimSpace(envelope.Prompt.GuidanceMarkdown)
	if guidance == "" {
		guidance = strings.TrimSpace(envelope.GuidanceMarkdown)
	}

	var builder strings.Builder
	builder.WriteString("Agent Heartbeat requested a reorientation for this existing session.")
	if summary != "" {
		builder.WriteString("\n\nPolicy summary:\n")
		builder.WriteString(summary)
	}
	if guidance != "" {
		builder.WriteString("\n\nPolicy guidance:\n")
		builder.WriteString(guidance)
	}
	builder.WriteString(
		"\n\nThis prompt does not assign ownership or include task-run credentials. " +
			"Inspect /agent/context and current session state before acting. " +
			"If task work is needed, inspect current task state through the task APIs first. " +
			"If there is nothing useful to do, record that no action is needed and remain idle.",
	)
	return builder.String()
}

func defaultWakeID(prefix string) string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		now := time.Now().UTC().UnixNano()
		if strings.TrimSpace(prefix) == "" {
			return fmt.Sprintf("%d", now)
		}
		return fmt.Sprintf("%s-%d", strings.TrimSpace(prefix), now)
	}
	if strings.TrimSpace(prefix) == "" {
		return hex.EncodeToString(random[:])
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(prefix), hex.EncodeToString(random[:]))
}
