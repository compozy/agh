package session

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
)

const (
	runtimeActivityKindPromptStarted = "prompt_started"
	runtimeActivityKindAgentWaiting  = "agent_waiting"
	runtimeActivityKindWarning       = "warning"
	runtimeActivityKindTimeout       = "timeout"
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
	config     aghconfig.SessionSupervisionConfig
	events     chan acp.AgentEvent

	mu        sync.Mutex
	activity  store.SessionActivityMeta
	warned    bool
	timedOut  bool
	closeOnce sync.Once
}

func newPromptActivitySupervisor(
	ctx context.Context,
	manager *Manager,
	session *Session,
	turnState *promptTurnDispatchState,
	config aghconfig.SessionSupervisionConfig,
) *promptActivitySupervisor {
	if ctx == nil {
		ctx = context.Background()
	}
	supervisorCtx, cancel := context.WithCancel(ctx)
	startedAt := time.Now().UTC()
	if manager != nil && manager.now != nil {
		startedAt = manager.now().UTC()
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

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.evaluate(now.UTC())
		}
	}
}

func (s *promptActivitySupervisor) evaluate(now time.Time) {
	if s == nil {
		return
	}
	if s.shouldEmitProgress(now) {
		s.emitRuntimeEvent(acp.EventTypeRuntimeProgress, s.progressText(now), now)
	}
	if s.shouldEmitWarning(now) {
		s.emitRuntimeEvent(acp.EventTypeRuntimeWarning, s.warningText(now), now)
	}
	if s.shouldTimeout(now) {
		s.handleTimeout(now)
	}
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
	s.session.markRuntimeStalled(store.SessionStallReasonActivityTimeout, now)
	if err := s.manager.writeMeta(s.session); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime timeout stall failed", "turn_id", s.turnID, "error", err)
	}
	s.emitRuntimeEvent(acp.EventTypeRuntimeWarning, s.timeoutText(now), now)

	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(s.ctx), s.config.TimeoutCancelGrace)
	cancelErr := s.manager.CancelPrompt(cancelCtx, s.session.ID)
	cancel()
	if cancelErr != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: cancel prompt after runtime timeout failed", "turn_id", s.turnID, "error", cancelErr)
	}

	timer := time.NewTimer(s.config.TimeoutCancelGrace)
	defer timer.Stop()
	select {
	case <-s.ctx.Done():
		return
	case <-timer.C:
	}

	stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(s.ctx), s.config.TimeoutCancelGrace)
	defer stopCancel()
	if err := s.manager.StopWithCause(
		stopCtx,
		s.session.ID,
		CauseTimeout,
		store.SessionStallReasonActivityTimeout,
	); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: stop session after runtime timeout failed", "turn_id", s.turnID, "error", err)
	}
}

func (s *promptActivitySupervisor) touch(now time.Time, kind string, detail string) {
	s.touchWithTool(now, kind, detail, "", "", false)
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

	s.mu.Lock()
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

	s.session.observeRuntimeActivity(activity, now)
	if err := s.manager.writeMeta(s.session); err != nil {
		s.manager.sessionLogger(s.session).
			Warn("session: persist runtime activity failed", "turn_id", s.turnID, "error", err)
	}
}

func (s *promptActivitySupervisor) emitRuntimeEvent(eventType string, text string, now time.Time) {
	if s == nil {
		return
	}
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
	event := acp.AgentEvent{
		Type:      eventType,
		TurnID:    s.turnID,
		Timestamp: now.UTC(),
		Text:      strings.TrimSpace(text),
		Runtime:   &activity,
	}
	select {
	case <-s.ctx.Done():
	case s.events <- event:
	}
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

func (s *promptActivitySupervisor) idleSecondsLocked(now time.Time) int64 {
	return store.SessionActivityIdleSeconds(&s.activity, now)
}

func (s *promptActivitySupervisor) now() time.Time {
	if s != nil && s.manager != nil && s.manager.now != nil {
		return s.manager.now().UTC()
	}
	return time.Now().UTC()
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
