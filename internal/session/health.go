package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/heartbeat"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/store"
)

// HealthStore is the durable store used by metadata-only session health.
type HealthStore interface {
	UpsertSessionHealth(ctx context.Context, health heartbeat.SessionHealth) (heartbeat.SessionHealth, error)
	GetSessionHealth(ctx context.Context, sessionID string) (heartbeat.SessionHealth, error)
	ListSessionHealth(ctx context.Context, query heartbeat.SessionHealthListQuery) ([]heartbeat.SessionHealth, error)
	ListSessionHealthRecoveryInputs(ctx context.Context, limit int) ([]heartbeat.SessionHealth, error)
	MarkSessionHealthStale(ctx context.Context, cutoff time.Time, updatedAt time.Time) (int64, error)
}

// HealthRecoveryResult summarizes one metadata-only restart recovery pass.
type HealthRecoveryResult struct {
	RefreshedActive int
	Recomputed      int
	MarkedStale     int64
}

type sessionHealthInput struct {
	activePrompt  bool
	attachable    bool
	touchPresence bool
	activityAt    time.Time
	lastError     string
}

// TouchSessionPresence records an idle metadata-only presence touch for an attachable session.
func (m *Manager) TouchSessionPresence(ctx context.Context, id string) (heartbeat.SessionHealth, error) {
	if m == nil {
		return heartbeat.SessionHealth{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return heartbeat.SessionHealth{}, errors.New("session: health context is required")
	}
	session, err := m.lookup(id)
	if err != nil {
		return heartbeat.SessionHealth{}, err
	}
	return m.persistSessionPresence(ctx, session, m.now())
}

// GetSessionHealth returns a route-ready metadata-only health read model.
func (m *Manager) GetSessionHealth(ctx context.Context, id string) (heartbeat.SessionHealth, error) {
	if m == nil {
		return heartbeat.SessionHealth{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return heartbeat.SessionHealth{}, errors.New("session: health context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return heartbeat.SessionHealth{}, errors.New("session: session id is required")
	}

	if session, ok := m.Get(target); ok {
		return m.persistSessionPresence(ctx, session, m.now())
	}

	existing, err := m.storedSessionHealth(ctx, target)
	if err != nil {
		return heartbeat.SessionHealth{}, err
	}
	meta, err := m.readMetaWithContext(ctx, target)
	if err != nil {
		return heartbeat.SessionHealth{}, err
	}
	health := m.sessionHealthFromInfo(sessionInfoFromMeta(meta), existing, m.now(), sessionHealthInput{})
	return m.storeSessionHealth(ctx, health)
}

// ListSessionHealth refreshes active rows, marks stale rows, and returns persisted health.
func (m *Manager) ListSessionHealth(
	ctx context.Context,
	query heartbeat.SessionHealthListQuery,
) ([]heartbeat.SessionHealth, error) {
	if m == nil {
		return nil, errors.New("session: manager is required")
	}
	if ctx == nil {
		return nil, errors.New("session: health context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if _, err := m.refreshActiveSessionHealth(ctx); err != nil {
		return nil, err
	}
	if err := m.markStaleSessionHealth(ctx, m.now()); err != nil {
		return nil, err
	}
	if m.sessionHealthStore == nil {
		return m.listInMemorySessionHealth(query), nil
	}
	rows, err := m.sessionHealthStore.ListSessionHealth(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("session: list session health: %w", err)
	}
	return rows, nil
}

// RecoverSessionHealth recomputes persisted rows after daemon restart before wake decisions run.
func (m *Manager) RecoverSessionHealth(ctx context.Context) (HealthRecoveryResult, error) {
	if m == nil {
		return HealthRecoveryResult{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return HealthRecoveryResult{}, errors.New("session: health recovery context is required")
	}

	refreshed, err := m.refreshActiveSessionHealth(ctx)
	if err != nil {
		return HealthRecoveryResult{}, err
	}
	result := HealthRecoveryResult{RefreshedActive: refreshed}
	if m.sessionHealthStore == nil {
		return result, nil
	}

	rows, err := m.sessionHealthStore.ListSessionHealthRecoveryInputs(ctx, 0)
	if err != nil {
		return HealthRecoveryResult{}, fmt.Errorf("session: list health recovery inputs: %w", err)
	}
	now := m.now()
	for _, row := range rows {
		if strings.TrimSpace(row.SessionID) == "" {
			continue
		}
		if _, ok := m.Get(row.SessionID); ok {
			continue
		}
		meta, readErr := m.readMetaWithContext(ctx, row.SessionID)
		if readErr != nil {
			if errors.Is(readErr, ErrSessionNotFound) {
				next := staleRecoveredSessionHealth(row, now)
				if !sessionHealthEqual(row, next) {
					if _, storeErr := m.storeSessionHealth(ctx, next); storeErr != nil {
						return HealthRecoveryResult{}, storeErr
					}
					result.Recomputed++
				}
				continue
			}
			return HealthRecoveryResult{}, fmt.Errorf(
				"session: read health recovery metadata for %q: %w",
				row.SessionID,
				readErr,
			)
		}
		next := m.sessionHealthFromInfo(sessionInfoFromMeta(meta), row, now, sessionHealthInput{})
		if sessionHealthEqual(row, next) {
			continue
		}
		if _, storeErr := m.storeSessionHealth(ctx, next); storeErr != nil {
			return HealthRecoveryResult{}, storeErr
		}
		result.Recomputed++
	}

	marked, err := m.markStaleSessionHealthCount(ctx, now)
	if err != nil {
		return HealthRecoveryResult{}, err
	}
	result.MarkedStale = marked
	return result, nil
}

func (m *Manager) persistSessionPresence(
	ctx context.Context,
	session *Session,
	at time.Time,
) (heartbeat.SessionHealth, error) {
	prompting := session != nil && session.IsPrompting()
	return m.persistSessionHealthForSession(ctx, session, at, sessionHealthInput{
		activePrompt:  prompting,
		attachable:    sessionAttachable(session),
		touchPresence: !prompting,
	})
}

func (m *Manager) persistSessionIdlePresence(
	ctx context.Context,
	session *Session,
	at time.Time,
) (heartbeat.SessionHealth, error) {
	return m.persistSessionHealthForSession(ctx, session, at, sessionHealthInput{
		activePrompt:  false,
		attachable:    sessionAttachable(session),
		touchPresence: true,
	})
}

func (m *Manager) persistSessionPromptActivity(
	ctx context.Context,
	session *Session,
	activityAt time.Time,
) (heartbeat.SessionHealth, error) {
	if activityAt.IsZero() {
		activityAt = m.now()
	}
	return m.persistSessionHealthForSession(ctx, session, activityAt, sessionHealthInput{
		activePrompt: true,
		attachable:   sessionAttachable(session),
		activityAt:   activityAt,
	})
}

func (m *Manager) persistSessionStoppedHealth(
	ctx context.Context,
	session *Session,
	at time.Time,
) (heartbeat.SessionHealth, error) {
	if at.IsZero() {
		at = m.now()
	}
	return m.persistSessionHealthForSession(ctx, session, at, sessionHealthInput{
		activePrompt: false,
		attachable:   false,
	})
}

func (m *Manager) persistSessionHealthForSession(
	ctx context.Context,
	session *Session,
	at time.Time,
	input sessionHealthInput,
) (heartbeat.SessionHealth, error) {
	if session == nil {
		return heartbeat.SessionHealth{}, errors.New("session: session is required")
	}
	if ctx == nil {
		return heartbeat.SessionHealth{}, errors.New("session: health context is required")
	}
	existing, err := m.storedSessionHealth(ctx, session.ID)
	if err != nil {
		return heartbeat.SessionHealth{}, err
	}
	health := m.sessionHealthFromInfo(session.Info(), existing, at, input)
	return m.storeSessionHealth(ctx, health)
}

func (m *Manager) sessionHealthFromInfo(
	info *Info,
	existing heartbeat.SessionHealth,
	now time.Time,
	input sessionHealthInput,
) heartbeat.SessionHealth {
	if now.IsZero() {
		if m != nil && m.now != nil {
			now = m.now()
		} else {
			now = time.Now().UTC()
		}
	}
	now = now.UTC()
	health := heartbeat.SessionHealth{
		LastActivityAt: existing.LastActivityAt,
		LastPresenceAt: existing.LastPresenceAt,
		UpdatedAt:      now,
	}
	if info != nil {
		health.SessionID = strings.TrimSpace(info.ID)
		health.WorkspaceID = strings.TrimSpace(info.WorkspaceID)
		health.AgentName = strings.TrimSpace(info.AgentName)
	}
	if !input.activityAt.IsZero() {
		health.LastActivityAt = input.activityAt.UTC()
	} else if activityAt := infoLastActivityAt(info); !activityAt.IsZero() {
		health.LastActivityAt = activityAt
	}
	if input.touchPresence {
		health.LastPresenceAt = now
	}
	health.ActivePrompt = input.activePrompt
	health.Attachable = input.attachable
	health.LastError = strings.TrimSpace(input.lastError)
	health.State = sessionHealthStateForInfo(info, input)
	health.Health = sessionHealthStatusForInfo(info)
	applySessionHealthEligibility(&health)
	return health.Normalize()
}

func staleRecoveredSessionHealth(row heartbeat.SessionHealth, now time.Time) heartbeat.SessionHealth {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	health := row.Normalize()
	health.State = heartbeat.SessionHealthStateDetached
	health.Health = heartbeat.SessionHealthStale
	health.ActivePrompt = false
	health.Attachable = false
	health.EligibleForWake = false
	health.IneligibilityReason = string(heartbeat.SessionHealthReasonStale)
	health.UpdatedAt = now.UTC()
	return health
}

func sessionHealthStateForInfo(info *Info, input sessionHealthInput) heartbeat.SessionHealthState {
	if info == nil {
		return heartbeat.SessionHealthStateDetached
	}
	switch info.State {
	case StateStopped:
		return heartbeat.SessionHealthStateStopped
	case StateActive:
		if input.activePrompt {
			return heartbeat.SessionHealthStatePrompting
		}
		if !input.attachable {
			return heartbeat.SessionHealthStateDetached
		}
		return heartbeat.SessionHealthStateIdle
	default:
		return heartbeat.SessionHealthStateDetached
	}
}

func sessionHealthStatusForInfo(info *Info) heartbeat.SessionHealthStatus {
	if info == nil {
		return heartbeat.SessionHealthUnknown
	}
	if info.State == StateStopped {
		return heartbeat.SessionHealthDead
	}
	if info.Liveness != nil &&
		strings.TrimSpace(info.Liveness.StallState) == store.SessionStallStateDetected {
		return heartbeat.SessionHealthDegraded
	}
	if info.State != StateActive {
		return heartbeat.SessionHealthUnknown
	}
	return heartbeat.SessionHealthHealthy
}

func applySessionHealthEligibility(health *heartbeat.SessionHealth) {
	if health == nil {
		return
	}
	health.EligibleForWake = false
	health.IneligibilityReason = ""
	switch {
	case health.Health == heartbeat.SessionHealthStale:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonStale)
	case health.Health == heartbeat.SessionHealthDead:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonDead)
	case health.Health == heartbeat.SessionHealthUnknown:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonUnknown)
	case health.Health == heartbeat.SessionHealthDegraded:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonHung)
	case health.ActivePrompt || health.State == heartbeat.SessionHealthStatePrompting:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonPromptActive)
	case !health.Attachable || health.State == heartbeat.SessionHealthStateDetached:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonNotAttachable)
	case health.State != heartbeat.SessionHealthStateIdle:
		health.IneligibilityReason = string(heartbeat.SessionHealthReasonUnhealthy)
	default:
		health.EligibleForWake = true
	}
}

func (m *Manager) storeSessionHealth(
	ctx context.Context,
	health heartbeat.SessionHealth,
) (heartbeat.SessionHealth, error) {
	normalized := health.Normalize()
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = m.now()
	}
	if err := normalized.Validate(); err != nil {
		return heartbeat.SessionHealth{}, err
	}
	if m.sessionHealthStore == nil {
		return normalized, nil
	}
	previous, err := m.storedSessionHealth(ctx, normalized.SessionID)
	if err != nil {
		return heartbeat.SessionHealth{}, err
	}
	stored, err := m.sessionHealthStore.UpsertSessionHealth(ctx, normalized)
	if err != nil {
		return heartbeat.SessionHealth{}, fmt.Errorf("session: upsert health for %q: %w", normalized.SessionID, err)
	}
	if err := m.dispatchSessionHealthUpdateAfter(ctx, previous, stored); err != nil && m.logger != nil {
		m.logger.Warn(
			"session: session health hook failed",
			"session_id", stored.SessionID,
			"error", err,
		)
	}
	return stored, nil
}

func (m *Manager) storedSessionHealth(ctx context.Context, sessionID string) (heartbeat.SessionHealth, error) {
	if m == nil || m.sessionHealthStore == nil {
		return heartbeat.SessionHealth{}, nil
	}
	health, err := m.sessionHealthStore.GetSessionHealth(ctx, sessionID)
	if err != nil {
		if errors.Is(err, heartbeat.ErrSessionHealthNotFound) {
			return heartbeat.SessionHealth{}, nil
		}
		return heartbeat.SessionHealth{}, fmt.Errorf("session: get health for %q: %w", sessionID, err)
	}
	return health, nil
}

func (m *Manager) refreshActiveSessionHealth(ctx context.Context) (int, error) {
	active := m.List()
	refreshed := 0
	for _, info := range active {
		if info == nil {
			continue
		}
		session, ok := m.Get(info.ID)
		if !ok {
			continue
		}
		if _, err := m.persistSessionPresence(ctx, session, m.now()); err != nil {
			return refreshed, err
		}
		refreshed++
	}
	return refreshed, nil
}

func (m *Manager) markStaleSessionHealth(ctx context.Context, now time.Time) error {
	_, err := m.markStaleSessionHealthCount(ctx, now)
	return err
}

func (m *Manager) markStaleSessionHealthCount(ctx context.Context, now time.Time) (int64, error) {
	if m.sessionHealthStore == nil {
		return 0, nil
	}
	if now.IsZero() {
		now = m.now()
	}
	cutoff := now.UTC().Add(-m.sessionHealthStaleAfter)
	marked, err := m.sessionHealthStore.MarkSessionHealthStale(ctx, cutoff, now.UTC())
	if err != nil {
		return 0, fmt.Errorf("session: mark stale health rows: %w", err)
	}
	return marked, nil
}

func (m *Manager) listInMemorySessionHealth(
	query heartbeat.SessionHealthListQuery,
) []heartbeat.SessionHealth {
	active := m.List()
	rows := make([]heartbeat.SessionHealth, 0, len(active))
	now := m.now()
	for _, info := range active {
		if info == nil {
			continue
		}
		health := m.sessionHealthFromInfo(info, heartbeat.SessionHealth{}, now, sessionHealthInput{
			activePrompt:  info.Liveness != nil && info.Liveness.Activity != nil,
			attachable:    info.State == StateActive,
			touchPresence: true,
		})
		if !sessionHealthMatchesQuery(health, query) {
			continue
		}
		rows = append(rows, health)
	}
	return rows
}

func sessionHealthMatchesQuery(health heartbeat.SessionHealth, query heartbeat.SessionHealthListQuery) bool {
	switch {
	case strings.TrimSpace(query.WorkspaceID) != "" && health.WorkspaceID != strings.TrimSpace(query.WorkspaceID):
		return false
	case strings.TrimSpace(query.AgentName) != "" && health.AgentName != strings.TrimSpace(query.AgentName):
		return false
	case strings.TrimSpace(query.SessionID) != "" && health.SessionID != strings.TrimSpace(query.SessionID):
		return false
	case query.State != "" && health.State != query.State:
		return false
	case query.Health != "" && health.Health != query.Health:
		return false
	case query.EligibleForWake != nil && health.EligibleForWake != *query.EligibleForWake:
		return false
	default:
		return true
	}
}

func sessionAttachable(session *Session) bool {
	if session == nil {
		return false
	}
	info := session.Info()
	if info == nil || info.State != StateActive {
		return false
	}
	proc := session.processHandle()
	return proc != nil && !isProcessDone(proc)
}

func infoLastActivityAt(info *Info) time.Time {
	if info == nil || info.Liveness == nil {
		return time.Time{}
	}
	if info.Liveness.Activity != nil &&
		info.Liveness.Activity.LastActivityAt != nil &&
		!info.Liveness.Activity.LastActivityAt.IsZero() {
		return info.Liveness.Activity.LastActivityAt.UTC()
	}
	return time.Time{}
}

func sessionHealthEqual(left heartbeat.SessionHealth, right heartbeat.SessionHealth) bool {
	left = left.Normalize()
	right = right.Normalize()
	return left.SessionID == right.SessionID &&
		left.WorkspaceID == right.WorkspaceID &&
		left.AgentName == right.AgentName &&
		left.State == right.State &&
		left.Health == right.Health &&
		left.ActivePrompt == right.ActivePrompt &&
		left.Attachable == right.Attachable &&
		left.EligibleForWake == right.EligibleForWake &&
		left.IneligibilityReason == right.IneligibilityReason &&
		left.LastError == right.LastError &&
		left.LastActivityAt.UTC().Equal(right.LastActivityAt.UTC()) &&
		left.LastPresenceAt.UTC().Equal(right.LastPresenceAt.UTC())
}

func (m *Manager) dispatchSessionHealthUpdateAfter(
	ctx context.Context,
	previous heartbeat.SessionHealth,
	current heartbeat.SessionHealth,
) error {
	if m == nil || !sessionHealthHookTransition(previous, current) {
		return nil
	}
	if !m.shouldDispatchSessionHealthHook(current) {
		return nil
	}
	payload := hookspkg.SessionHealthUpdateAfterPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionHealthUpdateAfter,
			Timestamp: current.UpdatedAt.UTC(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID:   current.SessionID,
			AgentName:   current.AgentName,
			WorkspaceID: current.WorkspaceID,
			State:       string(current.State),
			UpdatedAt:   current.UpdatedAt.UTC(),
		},
		Health:              string(current.Health),
		ActivePrompt:        current.ActivePrompt,
		Attachable:          current.Attachable,
		EligibleForWake:     current.EligibleForWake,
		IneligibilityReason: current.IneligibilityReason,
		LastActivityAt:      current.LastActivityAt.UTC(),
		LastPresenceAt:      current.LastPresenceAt.UTC(),
		LastError:           current.LastError,
	}
	_, err := m.hooks.authoredContext().DispatchSessionHealthUpdateAfter(ctx, payload)
	return err
}

func sessionHealthHookTransition(previous heartbeat.SessionHealth, current heartbeat.SessionHealth) bool {
	previous = previous.Normalize()
	current = current.Normalize()
	if strings.TrimSpace(current.SessionID) == "" {
		return false
	}
	if strings.TrimSpace(previous.SessionID) == "" {
		return true
	}
	return previous.State != current.State ||
		previous.Health != current.Health ||
		previous.EligibleForWake != current.EligibleForWake ||
		previous.IneligibilityReason != current.IneligibilityReason
}

func (m *Manager) shouldDispatchSessionHealthHook(current heartbeat.SessionHealth) bool {
	if m == nil {
		return false
	}
	sessionID := strings.TrimSpace(current.SessionID)
	if sessionID == "" {
		return false
	}
	now := current.UpdatedAt.UTC()
	if now.IsZero() {
		now = m.now()
	}
	m.sessionHealthHookMu.Lock()
	defer m.sessionHealthHookMu.Unlock()
	if m.sessionHealthHookLast == nil {
		m.sessionHealthHookLast = make(map[string]time.Time)
	}
	last := m.sessionHealthHookLast[sessionID]
	if !last.IsZero() && now.Sub(last) < m.sessionHealthHookMinInterval {
		return false
	}
	m.sessionHealthHookLast[sessionID] = now
	return true
}

func (m *Manager) detachedSessionHealthContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := ctx
	if base == nil && m != nil {
		base = m.lifecycleCtx
	}
	if base == nil {
		base = context.TODO()
	}
	return context.WithTimeout(context.WithoutCancel(base), defaultLifecycleTimeout)
}
