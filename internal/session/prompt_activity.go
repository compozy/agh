package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/diagnostics"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/subprocess"
	"github.com/compozy/agh/internal/transcript"
)

const (
	runtimeActivityKindPromptStarted   = "prompt_started"
	runtimeActivityKindAgentWaiting    = "agent_waiting"
	runtimeActivityKindWarning         = "warning"
	runtimeActivityKindTimeout         = "timeout"
	runtimeActivityEvidenceStallReason = "stall_reason"
)

type promptActivitySupervisor struct {
	manager *Manager
	session *Session

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	turnID     string
	turnSource TurnSource
	startedAt  time.Time
	deadlineAt *time.Time
	config     aghconfig.SessionSupervisionConfig
	events     chan acp.AgentEvent

	mu                    sync.Mutex
	activity              store.SessionActivityMeta
	warned                bool
	timedOut              bool
	unhealthy             bool
	unhealthyWarned       bool
	deadlineWarnAck       chan struct{}
	deadlineWarnEvent     acp.AgentEvent
	deadlineWarnPending   bool
	deadlineWarnDelivered bool
	closeOnce             sync.Once
}

func newPromptActivitySupervisor(
	ctx context.Context,
	manager *Manager,
	session *Session,
	turnState *promptTurnDispatchState,
	config aghconfig.SessionSupervisionConfig,
) *promptActivitySupervisor {
	supervisorBase := context.Background()
	if ctx != nil {
		supervisorBase = context.WithoutCancel(ctx)
	}
	supervisorCtx, cancel := context.WithCancel(supervisorBase)
	startedAt := time.Now().UTC()
	if manager != nil && manager.now != nil {
		startedAt = manager.now().UTC()
	}
	deadlineAt, hasDeadline := deadlineFromContext(ctx)
	if config.PromptDeadline > 0 {
		deadlineAt = startedAt.Add(config.PromptDeadline)
		hasDeadline = true
	}
	turnID := ""
	turnSource := TurnSourceUser
	if turnState != nil {
		turnID = strings.TrimSpace(turnState.turnID)
		turnSource = normalizeTurnSource(turnState.turnSource)
	}
	if turnSource == "" {
		turnSource = TurnSourceUser
	}

	return &promptActivitySupervisor{
		manager:    manager,
		session:    session,
		ctx:        supervisorCtx,
		cancel:     cancel,
		done:       make(chan struct{}),
		turnID:     turnID,
		turnSource: turnSource,
		startedAt:  startedAt,
		deadlineAt: deadlinePointer(deadlineAt, hasDeadline),
		config:     config,
		events:     make(chan acp.AgentEvent, 8),
		activity: store.SessionActivityMeta{
			TurnID:        turnID,
			TurnSource:    string(turnSource),
			TurnStartedAt: timePtr(startedAt),
		},
	}
}

func (s *promptActivitySupervisor) start() {
	if s == nil {
		return
	}
	s.touch(s.startedAt, runtimeActivityKindPromptStarted, "prompt started")
	go s.run()
}

func (s *promptActivitySupervisor) stop() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		s.cancel()
		<-s.done
	})
}

func (s *promptActivitySupervisor) eventsChannel() <-chan acp.AgentEvent {
	if s == nil {
		return nil
	}
	return s.events
}

func (s *promptActivitySupervisor) report(report acp.PromptActivityReport) {
	if s == nil {
		return
	}
	ts := report.Timestamp
	if ts.IsZero() {
		ts = s.now()
	}
	kind := strings.TrimSpace(report.Kind)
	if kind == "" {
		kind = runtimeActivityKindAgentWaiting
	}
	if kind == runtimeActivityKindAgentWaiting {
		s.recordWaitingHeartbeat(ts, report.Detail)
		return
	}
	s.touch(ts, kind, report.Detail)
}

func (s *promptActivitySupervisor) observeEvent(event acp.AgentEvent) {
	if s == nil {
		return
	}
	kind, detail, currentTool, toolCallID, clearTool := activityFromEvent(event)
	if kind == "" {
		return
	}
	s.touchWithTool(event.Timestamp, kind, detail, currentTool, toolCallID, clearTool)
}

func (s *promptActivitySupervisor) finish(now time.Time) {
	if s == nil || s.session == nil {
		return
	}
	if now.IsZero() {
		now = s.now()
	}
	s.session.clearRuntimeActivity(now)
	if err := s.manager.writeMeta(s.session); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime activity clear failed", "turn_id", s.turnID, "error", err)
	}
	healthCtx, cancel := s.manager.detachedSessionHealthContext(s.ctx)
	defer cancel()
	if _, err := s.manager.persistSessionIdlePresence(healthCtx, s.session, now); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime idle health failed", "turn_id", s.turnID, "error", err)
	}
}

func (s *promptActivitySupervisor) emitRecoveredMarkerIfNeeded(reason string) {
	if s == nil || s.manager == nil || s.session == nil {
		return
	}
	if strings.TrimSpace(reason) == "" {
		return
	}
	s.manager.emitTranscriptMarker(
		s.ctx,
		s.session,
		s.turnID,
		transcript.MarkerSessionRecovered,
		"Runtime activity recovered.",
		map[string]any{runtimeActivityEvidenceStallReason: reason},
	)
}

func (s *promptActivitySupervisor) run() {
	defer close(s.done)
	defer close(s.events)

	interval := s.config.ActivityHeartbeatInterval
	if interval <= 0 {
		interval = aghconfig.DefaultSessionSupervisionConfig().ActivityHeartbeatInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var deadlineTimer *time.Timer
	var deadlineCh <-chan time.Time
	if deadlineAt := cloneTimePointer(s.deadlineAt); deadlineAt != nil {
		wait := max(time.Until(deadlineAt.UTC()), 0)
		deadlineTimer = time.NewTimer(wait)
		deadlineCh = deadlineTimer.C
		defer deadlineTimer.Stop()
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			if s.evaluate(now.UTC()) {
				return
			}
		case now := <-deadlineCh:
			s.handlePromptDeadline(now.UTC())
			return
		}
	}
}

func (s *promptActivitySupervisor) evaluate(now time.Time) bool {
	if s == nil {
		return false
	}
	processUnhealthy := s.handleUnhealthyProcess(now, true)
	if !processUnhealthy && s.shouldEmitProgress(now) {
		s.emitRuntimeEvent(acp.EventTypeRuntimeProgress, s.progressText(now), now, nil)
	}
	if s.shouldEmitWarning(now) {
		s.emitRuntimeEvent(acp.EventTypeRuntimeWarning, s.warningText(now), now, nil)
	}
	if s.shouldTimeout(now) {
		s.handleTimeout(now)
		return true
	}
	return false
}

func (s *promptActivitySupervisor) shouldEmitProgress(now time.Time) bool {
	if s.config.ProgressNotifyInterval <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	base := s.startedAt
	if s.activity.LastProgressAt != nil && !s.activity.LastProgressAt.IsZero() {
		base = s.activity.LastProgressAt.UTC()
	}
	if now.Sub(base) < s.config.ProgressNotifyInterval {
		return false
	}
	progressAt := now.UTC()
	s.activity.LastProgressAt = &progressAt
	s.activity.IdleSeconds = store.SessionActivityIdleSeconds(&s.activity, now)
	return true
}

func (s *promptActivitySupervisor) shouldEmitWarning(now time.Time) bool {
	if s.config.InactivityWarningAfter <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.warned || s.idleSecondsLocked(now) < int64(s.config.InactivityWarningAfter.Seconds()) {
		return false
	}
	s.warned = true
	s.activity.LastActivityKind = runtimeActivityKindWarning
	s.activity.LastActivityDetail = "runtime activity is stale"
	s.activity.IdleSeconds = s.idleSecondsLocked(now)
	return true
}

func (s *promptActivitySupervisor) shouldTimeout(now time.Time) bool {
	if s.config.InactivityTimeout <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timedOut || s.idleSecondsLocked(now) < int64(s.config.InactivityTimeout.Seconds()) {
		return false
	}
	s.timedOut = true
	s.activity.LastActivityKind = runtimeActivityKindTimeout
	s.activity.LastActivityDetail = "runtime activity timed out"
	s.activity.IdleSeconds = s.idleSecondsLocked(now)
	return true
}

func (s *promptActivitySupervisor) handleTimeout(now time.Time) {
	if s == nil || s.session == nil || s.manager == nil {
		return
	}
	s.handleTimeoutWithDetail(now, store.SessionStallReasonActivityTimeout, s.timeoutText(now), nil)
}

func (s *promptActivitySupervisor) handlePromptDeadline(now time.Time) {
	if s == nil || s.session == nil || s.manager == nil {
		return
	}
	if s.promptDeadlineWarningDelivered() {
		return
	}

	warning, ok := s.promptDeadlineWarningEvent(now)
	if !ok {
		return
	}
	ack := s.preparePromptDeadlineWarning(warning)
	s.recordRuntimeTimeout(now, store.SessionStallReasonPromptDeadlineExceeded)
	s.emitPreparedRuntimeEvent(warning)
	s.waitForPromptDeadlineWarningAck(ack)
	s.cancelPromptAfterRuntimeTimeout()
	s.stopSessionAfterRuntimeTimeout(store.SessionStallReasonPromptDeadlineExceeded)
}

func (s *promptActivitySupervisor) handleTimeoutWithDetail(
	now time.Time,
	stopDetail string,
	text string,
	raw json.RawMessage,
) {
	if s == nil || s.session == nil || s.manager == nil {
		return
	}
	s.recordRuntimeTimeout(now, stopDetail)
	s.emitRuntimeEvent(acp.EventTypeRuntimeWarning, text, now, raw)
	s.cancelPromptAfterRuntimeTimeout()
	s.stopSessionAfterRuntimeTimeout(stopDetail)
}

func (s *promptActivitySupervisor) recordRuntimeTimeout(now time.Time, stopDetail string) {
	if s == nil || s.session == nil || s.manager == nil {
		return
	}
	s.session.markRuntimeStalled(stopDetail, now)
	if err := s.manager.writeMeta(s.session); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime timeout stall failed", "turn_id", s.turnID, "error", err)
	}
	if _, err := s.manager.persistSessionPromptActivity(s.ctx, s.session, now); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime timeout health failed", "turn_id", s.turnID, "error", err)
	}
	s.manager.emitTranscriptMarker(
		s.ctx,
		s.session,
		s.turnID,
		transcript.MarkerPromptTimeout,
		"Runtime activity timed out.",
		map[string]any{runtimeActivityEvidenceStallReason: stopDetail},
	)
}

func (s *promptActivitySupervisor) cancelPromptAfterRuntimeTimeout() {
	if s == nil || s.session == nil || s.manager == nil {
		return
	}
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(s.ctx), s.config.TimeoutCancelGrace)
	cancelErr := s.manager.CancelPrompt(cancelCtx, s.session.ID)
	cancel()
	if cancelErr != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: cancel prompt after runtime timeout failed", "turn_id", s.turnID, "error", cancelErr)
	}
}

func (s *promptActivitySupervisor) stopSessionAfterRuntimeTimeout(stopDetail string) {
	if s == nil || s.session == nil || s.manager == nil {
		return
	}
	timer := time.NewTimer(s.config.TimeoutCancelGrace)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			goto forceStop
		case <-ticker.C:
			if !s.session.IsPrompting() {
				return
			}
		}
	}

forceStop:
	if !s.session.IsPrompting() {
		return
	}

	stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(s.ctx), s.timeoutStopDeadline())
	defer stopCancel()
	if err := s.manager.StopWithCause(
		stopCtx,
		s.session.ID,
		CauseTimeout,
		stopDetail,
	); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: stop session after runtime timeout failed", "turn_id", s.turnID, "error", err)
	}
}

func (s *promptActivitySupervisor) timeoutStopDeadline() time.Duration {
	if s == nil || s.config.TimeoutCancelGrace <= 0 {
		return aghconfig.DefaultSessionSupervisionConfig().TimeoutCancelGrace
	}
	if s.config.TimeoutCancelGrace < defaultLifecycleTimeout {
		return defaultLifecycleTimeout
	}
	return s.config.TimeoutCancelGrace
}

func (s *promptActivitySupervisor) touch(now time.Time, kind string, detail string) {
	s.touchWithTool(now, kind, detail, "", "", false)
}

func (s *promptActivitySupervisor) recordWaitingHeartbeat(now time.Time, detail string) {
	if s == nil || s.manager == nil || s.session == nil {
		return
	}
	if now.IsZero() {
		now = s.now()
	}
	processUnhealthy := s.handleUnhealthyProcess(now, true)
	if processUnhealthy {
		return
	}
	s.mu.Lock()
	s.unhealthy = false
	s.unhealthyWarned = false
	s.activity.TurnID = s.turnID
	s.activity.TurnSource = string(s.turnSource)
	s.activity.TurnStartedAt = timePtr(s.startedAt)
	if s.activity.LastActivityAt == nil || s.activity.LastActivityAt.IsZero() {
		startedAt := s.startedAt.UTC()
		s.activity.LastActivityAt = &startedAt
	}
	s.activity.LastActivityKind = runtimeActivityKindAgentWaiting
	s.activity.LastActivityDetail = strings.TrimSpace(detail)
	s.activity.IdleSeconds = store.SessionActivityIdleSeconds(&s.activity, now)
	activity := *store.CloneSessionActivityMeta(&s.activity)
	lastActivityAt := time.Time{}
	if s.activity.LastActivityAt != nil {
		lastActivityAt = s.activity.LastActivityAt.UTC()
	}
	s.mu.Unlock()

	stallState, stallReason := s.session.observeRuntimeActivity(activity, now)
	if err := s.manager.writeMeta(s.session); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime heartbeat failed", "turn_id", s.turnID, "error", err)
	}
	if _, err := s.manager.persistSessionPromptActivity(s.ctx, s.session, lastActivityAt); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime heartbeat health failed", "turn_id", s.turnID, "error", err)
	}
	if stallState == store.SessionStallStateDetected {
		s.emitRecoveredMarkerIfNeeded(stallReason)
	}
}

func (s *promptActivitySupervisor) touchWithTool(
	now time.Time,
	kind string,
	detail string,
	currentTool string,
	toolCallID string,
	clearTool bool,
) {
	if s == nil || s.manager == nil || s.session == nil {
		return
	}
	if now.IsZero() {
		now = s.now()
	}
	processUnhealthy := s.handleUnhealthyProcess(now, kind == runtimeActivityKindAgentWaiting)
	if processUnhealthy && kind == runtimeActivityKindAgentWaiting {
		return
	}
	s.mu.Lock()
	s.unhealthy = false
	s.unhealthyWarned = false
	s.activity.TurnID = s.turnID
	s.activity.TurnSource = string(s.turnSource)
	s.activity.TurnStartedAt = timePtr(s.startedAt)
	lastActivityAt := now.UTC()
	s.activity.LastActivityAt = &lastActivityAt
	s.activity.LastActivityKind = strings.TrimSpace(kind)
	s.activity.LastActivityDetail = strings.TrimSpace(detail)
	if strings.TrimSpace(currentTool) != "" {
		s.activity.CurrentTool = strings.TrimSpace(currentTool)
	}
	if strings.TrimSpace(toolCallID) != "" {
		s.activity.ToolCallID = strings.TrimSpace(toolCallID)
	}
	if clearTool {
		s.activity.CurrentTool = ""
		s.activity.ToolCallID = ""
	}
	s.activity.IdleSeconds = 0
	activity := *store.CloneSessionActivityMeta(&s.activity)
	s.mu.Unlock()

	stallState, stallReason := s.session.observeRuntimeActivity(activity, now)
	if err := s.manager.writeMeta(s.session); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime activity failed", "turn_id", s.turnID, "error", err)
	}
	if _, err := s.manager.persistSessionPromptActivity(s.ctx, s.session, lastActivityAt); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime activity health failed", "turn_id", s.turnID, "error", err)
	}
	if stallState == store.SessionStallStateDetected {
		s.emitRecoveredMarkerIfNeeded(stallReason)
	}
}

func (s *promptActivitySupervisor) emitRuntimeEvent(
	eventType string,
	text string,
	now time.Time,
	raw json.RawMessage,
) {
	if s == nil {
		return
	}
	s.emitPreparedRuntimeEvent(s.buildRuntimeEvent(eventType, text, now, raw))
}

func (s *promptActivitySupervisor) buildRuntimeEvent(
	eventType string,
	text string,
	now time.Time,
	raw json.RawMessage,
) acp.AgentEvent {
	activity := s.runtimeActivity(now)
	if s.session != nil && s.manager != nil {
		if meta := storeActivityFromRuntime(activity); meta != nil {
			s.session.observeRuntimeEventActivity(*meta, now)
			if err := s.manager.writeMeta(s.session); err != nil {
				s.manager.sessionLogger(s.session).
					Warn("session: persist runtime progress failed", "turn_id", s.turnID, "error", err)
			}
		}
	}
	return acp.AgentEvent{
		Type:      eventType,
		TurnID:    s.turnID,
		Timestamp: now.UTC(),
		Text:      strings.TrimSpace(text),
		Runtime:   &activity,
		Raw:       acp.CloneRawMessage(raw),
	}
}

func (s *promptActivitySupervisor) emitPreparedRuntimeEvent(event acp.AgentEvent) {
	if s == nil {
		return
	}
	select {
	case <-s.ctx.Done():
	case s.events <- event:
	}
}

func (s *promptActivitySupervisor) preparePromptDeadlineWarning(event acp.AgentEvent) chan struct{} {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deadlineWarnAck = make(chan struct{})
	s.deadlineWarnEvent = event
	s.deadlineWarnPending = true
	s.deadlineWarnDelivered = false
	return s.deadlineWarnAck
}

func (s *promptActivitySupervisor) waitForPromptDeadlineWarningAck(ack chan struct{}) {
	if s == nil || ack == nil {
		return
	}
	select {
	case <-ack:
	case <-s.ctx.Done():
	}
}

func (s *promptActivitySupervisor) ackPromptDeadlineWarning(event acp.AgentEvent) {
	if s == nil || !isPromptDeadlineWarningEvent(event) {
		return
	}

	s.mu.Lock()
	ack := s.deadlineWarnAck
	s.deadlineWarnPending = false
	s.deadlineWarnDelivered = true
	s.deadlineWarnEvent = acp.AgentEvent{}
	s.deadlineWarnAck = nil
	s.mu.Unlock()

	if ack == nil {
		return
	}
	close(ack)
}

func (s *promptActivitySupervisor) pendingPromptDeadlineWarning() (acp.AgentEvent, bool) {
	if s == nil {
		return acp.AgentEvent{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.deadlineWarnPending || s.deadlineWarnDelivered {
		return acp.AgentEvent{}, false
	}
	return s.deadlineWarnEvent, true
}

func (s *promptActivitySupervisor) shouldSkipDeliveredPromptDeadlineWarning(event acp.AgentEvent) bool {
	if s == nil || !isPromptDeadlineWarningEvent(event) {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deadlineWarnDelivered
}

func (s *promptActivitySupervisor) promptDeadlineWarningDelivered() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deadlineWarnDelivered
}

func (s *promptActivitySupervisor) promptDeadlineWarningEvent(now time.Time) (acp.AgentEvent, bool) {
	if s == nil {
		return acp.AgentEvent{}, false
	}

	s.mu.Lock()
	deadline := cloneTimePointer(s.deadlineAt)
	pending := s.deadlineWarnPending
	delivered := s.deadlineWarnDelivered
	existing := s.deadlineWarnEvent
	s.mu.Unlock()

	if deadline == nil || delivered {
		return acp.AgentEvent{}, false
	}
	if pending {
		return existing, true
	}

	raw, err := jsonMarshalDeadlineWarning(s.runtimeActivity(now))
	if err != nil && s.manager != nil && s.session != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: marshal prompt deadline warning failed", "turn_id", s.turnID, "error", err)
	}
	return s.buildRuntimeEvent(acp.EventTypeRuntimeWarning, s.promptDeadlineText(now), now, raw), true
}

func isPromptDeadlineWarningEvent(event acp.AgentEvent) bool {
	return event.Type == acp.EventTypeRuntimeWarning && event.Runtime != nil && event.Runtime.DeadlineAt != nil
}

func (s *promptActivitySupervisor) handleUnhealthyProcess(now time.Time, emitWarning bool) bool {
	if s == nil || s.manager == nil || s.session == nil {
		return false
	}
	proc := s.session.processHandle()
	if proc == nil {
		return false
	}
	health, ok := proc.HealthState()
	if !ok || !processHealthFailureDetected(health) {
		s.mu.Lock()
		s.unhealthy = false
		s.unhealthyWarned = false
		s.mu.Unlock()
		return false
	}

	shouldPersist := false
	shouldWarn := false
	s.mu.Lock()
	if !s.unhealthy {
		s.unhealthy = true
		shouldPersist = true
	}
	if emitWarning && !s.unhealthyWarned {
		s.unhealthyWarned = true
		shouldWarn = true
	}
	s.mu.Unlock()

	if shouldPersist {
		healthError := unhealthyProcessDiagnostic(health)
		s.session.markRuntimeStalled(store.SessionStallReasonProcessUnhealthy, now)
		if err := s.manager.writeMeta(s.session); err != nil {
			s.manager.sessionLogger(s.session).
				Warn("session: persist unhealthy runtime stall failed", "turn_id", s.turnID, "error", err)
		}
		if _, err := s.manager.persistSessionHealthForSession(s.ctx, s.session, now, sessionHealthInput{
			activePrompt: true,
			attachable:   sessionAttachable(s.session),
			lastError:    healthError,
		}); err != nil {
			s.manager.sessionLogger(s.session).
				Warn("session: persist unhealthy runtime health failed", "turn_id", s.turnID, "error", err)
		}
	}
	if shouldWarn {
		s.manager.emitTranscriptMarker(
			s.ctx,
			s.session,
			s.turnID,
			transcript.MarkerSessionUnhealthy,
			"Runtime health check failed.",
			map[string]any{runtimeActivityEvidenceStallReason: store.SessionStallReasonProcessUnhealthy},
		)
		s.emitRuntimeEvent(acp.EventTypeRuntimeWarning, unhealthyProcessText(health), now, nil)
	}
	return true
}

func (s *promptActivitySupervisor) runtimeActivity(now time.Time) acp.RuntimeActivity {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.IsZero() {
		now = s.now()
	}
	s.activity.IdleSeconds = store.SessionActivityIdleSeconds(&s.activity, now)
	activity := acp.RuntimeActivity{
		TurnID:             strings.TrimSpace(s.activity.TurnID),
		TurnSource:         strings.TrimSpace(s.activity.TurnSource),
		TurnStartedAt:      cloneTimePointer(s.activity.TurnStartedAt),
		DeadlineAt:         cloneTimePointer(s.deadlineAt),
		LastActivityAt:     cloneTimePointer(s.activity.LastActivityAt),
		LastActivityKind:   strings.TrimSpace(s.activity.LastActivityKind),
		LastActivityDetail: strings.TrimSpace(s.activity.LastActivityDetail),
		CurrentTool:        strings.TrimSpace(s.activity.CurrentTool),
		ToolCallID:         strings.TrimSpace(s.activity.ToolCallID),
		LastProgressAt:     cloneTimePointer(s.activity.LastProgressAt),
		IterationCurrent:   s.activity.IterationCurrent,
		IterationMax:       s.activity.IterationMax,
		IdleSeconds:        s.activity.IdleSeconds,
	}
	if !s.startedAt.IsZero() {
		elapsed := now.UTC().Sub(s.startedAt.UTC())
		if elapsed > 0 {
			activity.ElapsedSeconds = int64(elapsed.Seconds())
			activity.ElapsedMS = elapsed.Milliseconds()
		}
	}
	return activity
}

func (s *promptActivitySupervisor) progressText(now time.Time) string {
	activity := s.runtimeActivity(now)
	parts := []string{"Still working..."}
	elapsedMinutes := activity.ElapsedSeconds / 60
	if elapsedMinutes > 0 {
		detail := fmt.Sprintf("%d min elapsed", elapsedMinutes)
		if activity.IterationCurrent > 0 && activity.IterationMax > 0 {
			detail += fmt.Sprintf(" - iteration %d/%d", activity.IterationCurrent, activity.IterationMax)
		}
		if activity.CurrentTool != "" {
			detail += ", running: " + activity.CurrentTool
		} else if activity.LastActivityKind != "" {
			detail += ", last activity: " + activity.LastActivityKind
		}
		parts = append(parts, "("+detail+")")
	}
	return strings.Join(parts, " ")
}

func (s *promptActivitySupervisor) warningText(now time.Time) string {
	activity := s.runtimeActivity(now)
	return fmt.Sprintf("Runtime activity is stale (%d seconds idle).", activity.IdleSeconds)
}

func (s *promptActivitySupervisor) timeoutText(now time.Time) string {
	activity := s.runtimeActivity(now)
	return fmt.Sprintf("Runtime activity timed out (%d seconds idle).", activity.IdleSeconds)
}

func (s *promptActivitySupervisor) promptDeadlineText(now time.Time) string {
	activity := s.runtimeActivity(now)
	if activity.ElapsedMS <= 0 {
		return "Prompt deadline exceeded."
	}
	return fmt.Sprintf("Prompt deadline exceeded after %d ms.", activity.ElapsedMS)
}

func unhealthyProcessText(health subprocess.HealthState) string {
	parts := []string{"Runtime health check failed; prompt may be stalled."}
	if detail := strings.TrimSpace(health.Message); detail != "" {
		parts = append(parts, detail)
	}
	if lastErr := strings.TrimSpace(health.LastError); lastErr != "" {
		parts = append(parts, lastErr)
	}
	return strings.Join(parts, " ")
}

func unhealthyProcessDiagnostic(health subprocess.HealthState) string {
	return diagnostics.RedactAndBound(unhealthyProcessText(health), maxSessionFailureSummaryBytes)
}

func processHealthFailureDetected(health subprocess.HealthState) bool {
	if health.Healthy {
		return false
	}
	return !health.LastCheckedAt.IsZero() ||
		health.ConsecutiveFailures > 0 ||
		strings.TrimSpace(health.LastError) != "" ||
		strings.TrimSpace(health.Message) != "" ||
		len(health.Details) > 0
}

func (s *promptActivitySupervisor) idleSecondsLocked(now time.Time) int64 {
	return store.SessionActivityIdleSeconds(&s.activity, now)
}

func (s *promptActivitySupervisor) now() time.Time {
	if s != nil && s.manager != nil && s.manager.now != nil {
		return s.manager.now().UTC()
	}
	return time.Now().UTC()
}

func deadlineFromContext(ctx context.Context) (time.Time, bool) {
	if ctx == nil {
		return time.Time{}, false
	}
	return ctx.Deadline()
}

func deadlinePointer(value time.Time, ok bool) *time.Time {
	if !ok || value.IsZero() {
		return nil
	}
	deadline := value.UTC()
	return &deadline
}

func jsonMarshalDeadlineWarning(activity acp.RuntimeActivity) (json.RawMessage, error) {
	payload := map[string]any{
		"elapsed_ms": activity.ElapsedMS,
	}
	if activity.DeadlineAt != nil && !activity.DeadlineAt.IsZero() {
		payload["deadline_at"] = activity.DeadlineAt.UTC().Format(time.RFC3339Nano)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func activityFromEvent(
	event acp.AgentEvent,
) (kind string, detail string, currentTool string, toolCallID string, clearTool bool) {
	kind = strings.TrimSpace(event.Type)
	switch kind {
	case "":
		return "", "", "", "", false
	case acp.EventTypeRuntimeProgress, acp.EventTypeRuntimeWarning:
		return "", "", "", "", false
	case acp.EventTypeToolCall:
		detail = firstNonEmpty(event.Title, event.Text, event.ToolCallID)
		currentTool = firstNonEmpty(event.Title, event.Resource)
		toolCallID = event.ToolCallID
	case acp.EventTypeToolResult:
		detail = firstNonEmpty(event.Title, event.Text, event.ToolCallID)
		clearTool = true
	default:
		detail = firstNonEmpty(event.Title, event.Text, event.Error, event.Action, event.Resource)
	}
	return kind, detail, currentTool, toolCallID, clearTool
}

func storeActivityFromRuntime(activity acp.RuntimeActivity) *store.SessionActivityMeta {
	meta := &store.SessionActivityMeta{
		TurnID:             strings.TrimSpace(activity.TurnID),
		TurnSource:         strings.TrimSpace(activity.TurnSource),
		TurnStartedAt:      cloneTimePointer(activity.TurnStartedAt),
		LastActivityAt:     cloneTimePointer(activity.LastActivityAt),
		LastActivityKind:   strings.TrimSpace(activity.LastActivityKind),
		LastActivityDetail: strings.TrimSpace(activity.LastActivityDetail),
		CurrentTool:        strings.TrimSpace(activity.CurrentTool),
		ToolCallID:         strings.TrimSpace(activity.ToolCallID),
		LastProgressAt:     cloneTimePointer(activity.LastProgressAt),
		IterationCurrent:   activity.IterationCurrent,
		IterationMax:       activity.IterationMax,
		IdleSeconds:        activity.IdleSeconds,
	}
	if meta.TurnID == "" && meta.LastActivityAt == nil && meta.LastProgressAt == nil {
		return nil
	}
	return meta
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}
