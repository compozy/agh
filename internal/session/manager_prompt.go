package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

type promptRequest struct {
	turnID     string
	target     string
	message    string
	turnSource TurnSource
	meta       acp.PromptMeta
}

type promptSubmissionPath string

const (
	promptSubmissionPathUserFacing promptSubmissionPath = "user_facing"
	promptSubmissionPathSynthetic  promptSubmissionPath = "synthetic"
)

type promptPumpLoopState struct {
	source    <-chan acp.AgentEvent
	runtime   <-chan acp.AgentEvent
	activity  *promptActivitySupervisor
	turnEnded bool
}

func (s *promptPumpLoopState) active() bool {
	return s != nil && (s.source != nil || s.runtime != nil)
}

func (s *promptPumpLoopState) sourceClosedShouldReturn() bool {
	if s == nil {
		return true
	}
	s.source = nil
	s.stopRuntime()
	return s.runtime == nil || s.turnEnded
}

func (s *promptPumpLoopState) runtimeClosedShouldReturn() bool {
	if s == nil {
		return true
	}
	s.runtime = nil
	return s.turnEnded || s.source == nil
}

func (s *promptPumpLoopState) turnEndedShouldReturn() bool {
	if s == nil {
		return true
	}
	s.turnEnded = true
	s.stopRuntime()
	return s.runtime == nil
}

func (s *promptPumpLoopState) stopRuntime() {
	if s == nil || s.activity == nil || s.runtime == nil {
		return
	}
	s.activity.stop()
}

func isPromptTerminalEvent(eventType string) bool {
	return eventType == acp.EventTypeDone || eventType == acp.EventTypeError
}

func isFatalPromptFailureEvent(event acp.AgentEvent) bool {
	if event.Type != acp.EventTypeError || event.Failure == nil {
		return false
	}
	return event.Failure.Kind == store.FailureProcess
}

// Prompt sends one prompt turn to an active session and mirrors the runtime stream into storage and observers.
func (m *Manager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	return m.PromptWithOpts(ctx, id, PromptOpts{
		Message:    msg,
		TurnSource: TurnSourceUser,
	})
}

// PromptNetwork sends one network-originated prompt turn to an active session.
func (m *Manager) PromptNetwork(
	ctx context.Context,
	id string,
	msg string,
	meta ...acp.PromptNetworkMeta,
) (<-chan acp.AgentEvent, error) {
	if len(meta) > 1 {
		return nil, errors.New("session: network prompt accepts at most one metadata value")
	}

	var promptMeta acp.PromptMeta
	if len(meta) > 0 {
		promptMeta.Network = &meta[0]
	}
	return m.PromptWithOpts(ctx, id, PromptOpts{
		Message:    msg,
		TurnSource: TurnSourceNetwork,
		PromptMeta: promptMeta,
	})
}

// PromptWithOpts sends one prompt turn with daemon-local provenance metadata.
func (m *Manager) PromptWithOpts(ctx context.Context, id string, opts PromptOpts) (<-chan acp.AgentEvent, error) {
	req, err := m.parsePromptRequest(ctx, id, opts)
	if err != nil {
		return nil, err
	}

	return m.submitPromptRequest(ctx, req)
}

func (m *Manager) submitPromptRequest(ctx context.Context, req promptRequest) (<-chan acp.AgentEvent, error) {
	session, err := m.lookupPromptSession(ctx, req.target)
	if err != nil {
		return nil, err
	}

	message, err := m.dispatchInputPreSubmit(ctx, session, req.turnID, req.turnSource, req.message)
	if err != nil {
		return nil, err
	}
	turnState := newPromptTurnDispatchState(session, req.turnID, req.turnSource, message)
	if err := m.dispatchTurnStart(ctx, turnState); err != nil {
		return nil, err
	}

	proc, err := session.beginExclusivePromptSetup()
	if err != nil {
		return nil, err
	}
	defer session.finishPromptSetup()
	session.setCurrentTurnID(req.turnID)
	session.setCurrentTurnSource(turnState.turnSource)
	session.setCurrentPromptMeta(req.meta)
	promptExecutionCtx, cancelPromptExecution := m.promptExecutionContext(ctx)
	session.setCurrentPromptCancel(cancelPromptExecution)
	clearTurnSource := true
	defer func() {
		if clearTurnSource {
			cancelPromptExecution()
			session.clearCurrentTurnID()
			session.clearCurrentTurnSource()
			session.clearCurrentPromptMeta()
			session.clearCurrentPromptCancel()
		}
	}()

	recordReq := req
	recordReq.message = message
	if err := m.recordPromptInputEvent(ctx, session, recordReq); err != nil {
		return nil, err
	}

	dispatchMessage, err := m.promptDispatchMessage(ctx, session, message)
	if err != nil {
		return nil, err
	}
	if _, err := m.persistSessionPromptActivity(ctx, session, m.now()); err != nil {
		return nil, err
	}
	activity := newPromptActivitySupervisor(promptExecutionCtx, m, session, turnState, m.supervision)
	activity.start()
	source, err := m.driver.Prompt(promptExecutionCtx, proc, acp.PromptRequest{
		TurnID:                    req.turnID,
		Message:                   dispatchMessage,
		Meta:                      req.meta,
		ActivityReporter:          activity.report,
		ActivityHeartbeatInterval: m.supervision.ActivityHeartbeatInterval,
	})
	if err != nil {
		cancelPromptExecution()
		activity.stop()
		activity.finish(m.now())
		return nil, fmt.Errorf("session: prompt session %q: %w", req.target, err)
	}

	clearTurnSource = false
	pumpCtx := m.fallbackLifecycleContext()
	out := m.startPromptPump(pumpCtx, ctx, session, turnState, source, activity, cancelPromptExecution)
	return out, nil
}

func (m *Manager) startPromptPump(
	promptExecutionCtx context.Context,
	callerCtx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	source <-chan acp.AgentEvent,
	activity *promptActivitySupervisor,
	cancelPromptExecution context.CancelFunc,
) <-chan acp.AgentEvent {
	out := make(chan acp.AgentEvent, m.promptBufSize)
	finishDrain := m.trackPromptDrain()
	go func() {
		defer finishDrain()
		m.pumpPrompt(
			promptExecutionCtx,
			callerCtx,
			session,
			turnState,
			source,
			activity.eventsChannel(),
			out,
			activity,
			cancelPromptExecution,
		)
	}()
	return out
}

func (m *Manager) promptDispatchMessage(ctx context.Context, session *Session, message string) (string, error) {
	if m.inputAugmenter == nil {
		return message, nil
	}
	augmented, err := m.inputAugmenter(ctx, session, message)
	if err != nil {
		return "", fmt.Errorf("session: augment prompt input: %w", err)
	}
	if strings.TrimSpace(augmented) == "" {
		return message, nil
	}
	return augmented, nil
}

func (m *Manager) parsePromptRequest(ctx context.Context, id string, opts PromptOpts) (promptRequest, error) {
	if ctx == nil {
		return promptRequest{}, errors.New("session: prompt context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return promptRequest{}, errors.New("session: session id is required")
	}

	message := strings.TrimSpace(opts.Message)
	if message == "" {
		return promptRequest{}, errors.New("session: prompt message is required")
	}

	turnSource := normalizeTurnSource(opts.TurnSource)
	if turnSource == "" {
		return promptRequest{}, fmt.Errorf(
			"session: invalid turn source %q",
			strings.TrimSpace(string(opts.TurnSource)),
		)
	}

	meta, err := normalizePromptMeta(turnSource, opts.PromptMeta, promptSubmissionPathUserFacing)
	if err != nil {
		return promptRequest{}, err
	}

	return promptRequest{
		turnID:     m.newPromptTurnID(),
		target:     target,
		message:    message,
		turnSource: turnSource,
		meta:       meta,
	}, nil
}

func normalizePromptMeta(
	turnSource TurnSource,
	meta acp.PromptMeta,
	path promptSubmissionPath,
) (acp.PromptMeta, error) {
	normalized := meta.Normalize()
	if normalized.TurnSource == "" {
		normalized.TurnSource = string(turnSource)
	}
	if normalized.TurnSource != string(turnSource) {
		return acp.PromptMeta{}, fmt.Errorf(
			"session: prompt turn source %q does not match metadata turn_source %q",
			turnSource,
			normalized.TurnSource,
		)
	}
	if turnSource == TurnSourceSynthetic {
		if path != promptSubmissionPathSynthetic {
			return acp.PromptMeta{}, errors.New(
				"session: synthetic prompt turns require the dedicated synthetic submission path",
			)
		}
		if normalized.Synthetic == nil {
			return acp.PromptMeta{}, errors.New(
				"session: synthetic prompt turns require synthetic metadata",
			)
		}
	}
	if turnSource == TurnSourceUser && normalized.Network != nil {
		return acp.PromptMeta{}, errors.New("session: user prompt metadata cannot include network fields")
	}
	if err := normalized.Validate(); err != nil {
		return acp.PromptMeta{}, err
	}
	return normalized, nil
}

func (m *Manager) newPromptTurnID() string {
	if m == nil || m.newTurnID == nil {
		return newID("turn")
	}

	turnID := strings.TrimSpace(m.newTurnID())
	if turnID == "" {
		return newID("turn")
	}
	return turnID
}

func (m *Manager) lookupPromptSession(ctx context.Context, target string) (*Session, error) {
	session, err := m.lookup(target)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}

	meta, metaErr := m.readMetaWithContext(ctx, target)
	switch {
	case metaErr == nil:
		return nil, fmt.Errorf("%w: %s (%s)", ErrSessionNotActive, target, meta.State)
	case errors.Is(metaErr, ErrSessionNotFound):
		return nil, err
	default:
		return nil, metaErr
	}
}

func (m *Manager) recordPromptInputEvent(
	ctx context.Context,
	session *Session,
	req promptRequest,
) error {
	event := acp.AgentEvent{
		Type:      acp.EventTypeUserMessage,
		TurnID:    req.turnID,
		Timestamp: m.now(),
		Text:      req.message,
	}
	if req.turnSource == TurnSourceSynthetic {
		event.Type = acp.EventTypeSyntheticReentry
		event.Synthetic = clonePromptSyntheticMeta(req.meta.Synthetic)
	}
	event = m.normalizeEvent(session, req.turnID, event)
	if err := m.recordEvent(ctx, session, event); err != nil {
		return fmt.Errorf("session: persist prompt message for %q: %w", req.target, err)
	}
	m.notifyAgentEvent(ctx, session, event)
	return nil
}

func clonePromptSyntheticMeta(meta *acp.PromptSyntheticMeta) *acp.PromptSyntheticMeta {
	if meta == nil {
		return nil
	}

	cloned := meta.Normalize()
	if cloned.IsZero() {
		return nil
	}
	return &cloned
}

// CancelPrompt cancels prompt setup/execution for a known session.
func (m *Manager) CancelPrompt(ctx context.Context, id string) error {
	if m == nil {
		return errors.New("session: manager is required")
	}
	if ctx == nil {
		return errors.New("session: cancel prompt context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return errors.New("session: session id is required")
	}

	session, ok := m.Get(target)
	if !ok {
		if _, err := m.readMetaWithContext(ctx, target); err != nil {
			return err
		}
		return nil
	}
	if !session.IsPrompting() {
		return nil
	}
	turnID := session.CurrentTurnID()
	session.cancelCurrentPrompt()

	proc := session.processHandle()
	if proc == nil {
		m.emitTranscriptMarker(
			ctx,
			session,
			turnID,
			transcript.MarkerPromptCancel,
			"Prompt canceled by operator.",
			map[string]any{"source": "cancel_prompt"},
		)
		return nil
	}

	cancelErr := m.driver.Cancel(ctx, proc)
	if cancelErr != nil {
		if isProcessDone(proc) {
			return nil
		}
		return fmt.Errorf("session: cancel prompt for %q: %w", target, cancelErr)
	}
	if scoped, ok := m.driver.(ScopedInterrupter); ok && strings.TrimSpace(turnID) != "" {
		if _, err := scoped.Interrupt(ctx, target, turnID); err != nil &&
			!errors.Is(err, ErrScopedInterruptNotFound) {
			return fmt.Errorf("session: interrupt scoped tools for %q: %w", target, err)
		}
	}
	m.emitTranscriptMarker(
		ctx,
		session,
		turnID,
		transcript.MarkerPromptCancel,
		"Prompt canceled by operator.",
		map[string]any{"source": "cancel_prompt"},
	)
	return nil
}

// ApprovePermission resolves one pending interactive permission request for an active session.
func (m *Manager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if ctx == nil {
		return errors.New("session: approval context is required")
	}
	if err := req.Validate(); err != nil {
		return err
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return errors.New("session: session id is required")
	}

	session, ok := m.Get(target)
	if !ok {
		meta, err := m.readMetaWithContext(ctx, target)
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: %s (%s)", ErrSessionNotActive, target, meta.State)
	}

	if err := session.ApprovePermission(ctx, req); err != nil {
		switch {
		case errors.Is(err, ErrSessionNotActive):
			return err
		case errors.Is(err, acp.ErrPendingPermissionNotFound):
			return fmt.Errorf("%w: %s", ErrPendingPermissionNotFound, target)
		case errors.Is(err, acp.ErrPendingPermissionConflict):
			return fmt.Errorf("%w: %s", ErrPendingPermissionConflict, target)
		case errors.Is(err, acp.ErrPermissionDecisionUnsupported):
			return fmt.Errorf("%w: %s", ErrInvalidPermissionDecision, target)
		default:
			return err
		}
	}
	return nil
}

// RequestPermission asks an active session's permission path for a tool-call decision.
func (m *Manager) RequestPermission(
	ctx context.Context,
	id string,
	req acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	if ctx == nil {
		return acp.RequestPermissionResponse{}, errors.New("session: permission context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return acp.RequestPermissionResponse{}, errors.New("session: session id is required")
	}

	session, ok := m.Get(target)
	if !ok {
		meta, err := m.readMetaWithContext(ctx, target)
		if err != nil {
			return acp.RequestPermissionResponse{}, err
		}
		return acp.RequestPermissionResponse{}, fmt.Errorf("%w: %s (%s)", ErrSessionNotActive, target, meta.State)
	}

	response, err := session.RequestPermission(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotActive):
			return acp.RequestPermissionResponse{}, err
		default:
			return acp.RequestPermissionResponse{}, fmt.Errorf("session: request permission for %q: %w", target, err)
		}
	}
	return response, nil
}

func (m *Manager) pumpPrompt(
	ctx context.Context,
	deliveryCtx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	source <-chan acp.AgentEvent,
	runtime <-chan acp.AgentEvent,
	out chan<- acp.AgentEvent,
	activity *promptActivitySupervisor,
	releaseExecution context.CancelFunc,
) {
	var fatalPromptFailure *store.SessionFailure
	var fatalPromptError string

	defer func() {
		if releaseExecution != nil {
			releaseExecution()
		}
		if activity != nil {
			activity.stop()
			activity.finish(m.now())
		}
		m.finishPromptMessage(ctx, turnState, time.Time{})
		m.dispatchTurnEnd(ctx, turnState, time.Time{})
		if session != nil {
			session.clearCurrentTurnID()
			session.clearCurrentTurnSource()
			session.clearCurrentPromptMeta()
			session.clearCurrentPromptCancel()
		}
		notifier := m.currentTurnEndNotifier()
		if notifier != nil && session != nil {
			notifier(session.ID)
		}
		close(out)
		if session != nil {
			if fatalPromptFailure != nil {
				m.stopSessionAfterFatalPromptFailure(ctx, session, fatalPromptFailure, fatalPromptError)
			} else {
				m.startNextQueuedInputPrompt(session.ID)
				m.startNextQueuedSyntheticPrompt(session.ID)
			}
		}
	}()

	loop := promptPumpLoopState{source: source, runtime: runtime, activity: activity}
	for loop.active() {
		event, runtimeEvent, ok := nextPromptPumpEvent(ctx, &loop)
		if !ok {
			return
		}
		failure, errorText, stop := m.handlePromptPumpEvent(
			ctx,
			deliveryCtx,
			session,
			turnState,
			out,
			&loop,
			event,
			runtimeEvent,
		)
		if failure != nil {
			fatalPromptFailure = failure
			fatalPromptError = errorText
		}
		if stop {
			return
		}
	}
}

func (m *Manager) promptExecutionContext(ctx context.Context) (context.Context, context.CancelFunc) {
	var base context.Context
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	if base == nil {
		base = m.fallbackLifecycleContext()
	}
	executionCtx, cancel := context.WithCancel(base)
	lifecycleStop := context.AfterFunc(m.fallbackLifecycleContext(), cancel)
	return executionCtx, func() {
		lifecycleStop()
		cancel()
	}
}

func (m *Manager) stopSessionAfterFatalPromptFailure(
	ctx context.Context,
	session *Session,
	failure *store.SessionFailure,
	errorText string,
) {
	if m == nil || session == nil || failure == nil {
		return
	}
	if info := session.Info(); info == nil || info.State != StateActive {
		return
	}

	proc := session.processHandle()
	if proc == nil {
		return
	}

	summary := firstNonEmptySessionFailureText(
		failureSummary(failure, errorText),
		strings.TrimSpace(errorText),
		"agent runtime became unavailable during prompt",
	)
	proc.setWaitErrorOverride(acp.WrapFailure(store.FailureProcess, summary, errors.New(summary)))

	stopCtx, cancel := detachedPromptStopContext(ctx, m)
	defer cancel()

	if err := m.StopWithCause(stopCtx, session.ID, CauseProcessExited, summary); err != nil &&
		!errors.Is(err, ErrSessionNotFound) {
		m.sessionLogger(session).Warn(
			"session: stop after fatal prompt failure failed",
			"error", err,
			"failure_kind", failure.Kind,
		)
	}
}

func detachedPromptStopContext(ctx context.Context, m *Manager) (context.Context, context.CancelFunc) {
	base := ctx
	if base == nil && m != nil {
		base = m.lifecycleCtx
	}
	if base == nil {
		base = context.TODO()
	}
	return context.WithTimeout(context.WithoutCancel(base), defaultLifecycleTimeout)
}

func nextPromptPumpEvent(
	ctx context.Context,
	loop *promptPumpLoopState,
) (acp.AgentEvent, bool, bool) {
	for {
		if loop.runtime != nil {
			select {
			case event, ok := <-loop.runtime:
				if !ok {
					if loop.runtimeClosedShouldReturn() {
						return acp.AgentEvent{}, false, false
					}
				} else {
					return event, true, true
				}
			default:
			}
		}
		select {
		case <-ctx.Done():
			return acp.AgentEvent{}, false, false
		case event, ok := <-loop.source:
			if !ok {
				if loop.sourceClosedShouldReturn() {
					return acp.AgentEvent{}, false, false
				}
				continue
			}
			return event, false, true
		case event, ok := <-loop.runtime:
			if !ok {
				if loop.runtimeClosedShouldReturn() {
					return acp.AgentEvent{}, false, false
				}
				continue
			}
			return event, true, true
		}
	}
}

func (m *Manager) handlePromptPumpEvent(
	ctx context.Context,
	deliveryCtx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	out chan<- acp.AgentEvent,
	loop *promptPumpLoopState,
	event acp.AgentEvent,
	runtimeEvent bool,
) (*store.SessionFailure, string, bool) {
	if failure, errorText, stop, handled := m.emitPromptDeadlineWarningBeforeError(
		ctx,
		deliveryCtx,
		session,
		turnState,
		out,
		loop,
		event,
		runtimeEvent,
	); handled {
		return failure, errorText, stop
	}

	normalized, skip := m.preparePromptPumpEventForDelivery(ctx, session, turnState, loop, event, runtimeEvent)
	if skip {
		return nil, "", false
	}

	fatalPromptFailure := promptFatalFailure(normalized)
	m.observeRecordAndNotifyPromptEvent(ctx, session, turnState, loop, normalized, runtimeEvent)
	if stop := m.sendPromptPumpEvent(ctx, deliveryCtx, out, loop, normalized, runtimeEvent); stop {
		return fatalPromptFailure, normalized.Error, true
	}
	if stop := m.finishPromptTurnIfNeeded(ctx, turnState, loop, normalized); stop {
		return fatalPromptFailure, normalized.Error, true
	}

	return fatalPromptFailure, normalized.Error, false
}

func (m *Manager) emitPromptDeadlineWarningBeforeError(
	ctx context.Context,
	deliveryCtx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	out chan<- acp.AgentEvent,
	loop *promptPumpLoopState,
	event acp.AgentEvent,
	runtimeEvent bool,
) (*store.SessionFailure, string, bool, bool) {
	if runtimeEvent || event.Type != acp.EventTypeError || loop.activity == nil {
		return nil, "", false, false
	}

	warning, ok := loop.activity.pendingPromptDeadlineWarning()
	if !ok && strings.Contains(event.Error, "context deadline exceeded") {
		warning, ok = loop.activity.promptDeadlineWarningEvent(m.now())
	}
	if !ok {
		return nil, "", false, false
	}

	failure, errorText, stop := m.handlePromptPumpEvent(
		ctx,
		deliveryCtx,
		session,
		turnState,
		out,
		loop,
		warning,
		true,
	)
	if failure == nil && errorText == "" && !stop {
		return nil, "", false, false
	}
	return failure, errorText, stop, true
}

func (m *Manager) preparePromptPumpEventForDelivery(
	ctx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	loop *promptPumpLoopState,
	event acp.AgentEvent,
	runtimeEvent bool,
) (acp.AgentEvent, bool) {
	normalized := m.normalizeEvent(session, turnState.turnID, event)
	if runtimeEvent && loop.activity != nil && loop.activity.shouldSkipDeliveredPromptDeadlineWarning(normalized) {
		return acp.AgentEvent{}, true
	}
	normalized = m.attachPromptFailureDiagnostics(ctx, session, normalized)
	normalized = m.preparePromptEvent(ctx, turnState, normalized)
	normalized = transcript.RedactAgentEvent(normalized)
	return normalized, false
}

func promptFatalFailure(event acp.AgentEvent) *store.SessionFailure {
	if !isFatalPromptFailureEvent(event) {
		return nil
	}
	return store.CloneSessionFailure(event.Failure)
}

func (m *Manager) observeRecordAndNotifyPromptEvent(
	ctx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	loop *promptPumpLoopState,
	normalized acp.AgentEvent,
	runtimeEvent bool,
) {
	if loop.activity != nil && !runtimeEvent {
		loop.activity.observeEvent(normalized)
	}
	if err := m.recordEvent(ctx, session, normalized); err != nil {
		m.sessionLogger(session).
			Warn("session: record prompt event failed", "turn_id", turnState.turnID, "error", err)
	}
	m.notifyAgentEvent(ctx, session, normalized)
	if kind, summary, evidence, ok := promptTranscriptMarker(normalized); ok {
		m.emitTranscriptMarker(ctx, session, turnState.turnID, kind, summary, evidence)
	}
}

func (m *Manager) emitTranscriptMarker(
	ctx context.Context,
	session *Session,
	turnID string,
	kind string,
	summary string,
	evidence map[string]any,
) {
	marker, err := transcript.NewMarker(kind, summary, m.now(), evidence)
	if err != nil {
		m.sessionLogger(session).Warn("session: build transcript marker failed", "kind", kind, "error", err)
		return
	}
	event, err := marker.AgentEvent("", turnID)
	if err != nil {
		m.sessionLogger(session).Warn("session: convert transcript marker failed", "kind", kind, "error", err)
		return
	}
	normalized := transcript.RedactAgentEvent(m.normalizeEvent(session, turnID, event))
	if err := m.recordEvent(ctx, session, normalized); err != nil {
		m.sessionLogger(session).Warn("session: record transcript marker failed", "kind", kind, "error", err)
		return
	}
	m.notifyAgentEvent(ctx, session, normalized)
}

func promptTranscriptMarker(event acp.AgentEvent) (string, string, map[string]any, bool) {
	combined := strings.ToLower(strings.Join([]string{
		event.Text,
		event.Error,
		runtimeActivityKind(event.Runtime),
		runtimeActivityDetail(event.Runtime),
	}, " "))
	summary := firstNonEmpty(event.Text, event.Error, runtimeActivityDetail(event.Runtime))
	evidence := map[string]any{
		"event_type": event.Type,
	}
	switch {
	case event.Type == acp.EventTypeUserMessage && event.Action == acp.PromptActionSteered:
		if queueEntryID := strings.TrimSpace(event.RequestID); queueEntryID != "" {
			evidence["queue_entry_id"] = queueEntryID
		}
		if generation := strings.TrimSpace(event.Decision); generation != "" {
			evidence[promptEvidenceQueueGenerationKey] = generation
		}
		return transcript.MarkerPromptSteered,
			firstNonEmpty(summary, "Steering input injected at tool result boundary."),
			evidence,
			true
	case event.Type == acp.EventTypeRuntimeWarning && (strings.Contains(combined, "timeout") ||
		strings.Contains(combined, "timed out") ||
		strings.Contains(combined, "deadline exceeded")):
		return transcript.MarkerPromptTimeout, firstNonEmpty(summary, "Runtime activity timed out."), evidence, true
	case event.Type == acp.EventTypeRuntimeWarning && (strings.Contains(combined, "unhealthy") ||
		strings.Contains(combined, string(store.SessionStallReasonProcessUnhealthy)) ||
		strings.Contains(combined, "health check failed")):
		return transcript.MarkerSessionUnhealthy, firstNonEmpty(summary, "Runtime health check failed."), evidence, true
	case event.Type == acp.EventTypeError && event.Failure != nil:
		failure := event.Failure.Normalize()
		evidence["failure_kind"] = string(failure.Kind)
		if failure.Kind == store.FailureProviderAuth || failure.Kind == store.FailurePermission {
			if strings.Contains(combined, "mcp") &&
				(strings.Contains(combined, "auth") || strings.Contains(combined, "login")) {
				return transcript.MarkerMCPAuthRequired,
					firstNonEmpty(summary, failure.Summary, "MCP authentication is required."),
					evidence,
					true
			}
			return transcript.MarkerProviderFailure,
				firstNonEmpty(summary, failure.Summary, "Provider authentication failed."),
				evidence,
				true
		}
		return transcript.MarkerProviderFailure,
			firstNonEmpty(summary, failure.Summary, "Provider failed."),
			evidence,
			true
	default:
		return "", "", nil, false
	}
}

func runtimeActivityKind(activity *acp.RuntimeActivity) string {
	if activity == nil {
		return ""
	}
	return activity.LastActivityKind
}

func runtimeActivityDetail(activity *acp.RuntimeActivity) string {
	if activity == nil {
		return ""
	}
	return activity.LastActivityDetail
}

func (m *Manager) sendPromptPumpEvent(
	ctx context.Context,
	deliveryCtx context.Context,
	out chan<- acp.AgentEvent,
	loop *promptPumpLoopState,
	normalized acp.AgentEvent,
	runtimeEvent bool,
) bool {
	if ctx == nil {
		ctx = m.fallbackLifecycleContext()
	}
	if deliveryCtx == nil {
		deliveryCtx = ctx
	}
	select {
	case <-deliveryCtx.Done():
		ackPromptPumpRuntimeEvent(loop, normalized, runtimeEvent)
		return false
	default:
	}
	select {
	case out <- normalized:
	case <-deliveryCtx.Done():
		ackPromptPumpRuntimeEvent(loop, normalized, runtimeEvent)
		return false
	case <-ctx.Done():
		return true
	}
	ackPromptPumpRuntimeEvent(loop, normalized, runtimeEvent)
	return false
}

func ackPromptPumpRuntimeEvent(loop *promptPumpLoopState, normalized acp.AgentEvent, runtimeEvent bool) {
	if runtimeEvent && loop.activity != nil {
		loop.activity.ackPromptDeadlineWarning(normalized)
	}
}

func (m *Manager) finishPromptTurnIfNeeded(
	ctx context.Context,
	turnState *promptTurnDispatchState,
	loop *promptPumpLoopState,
	normalized acp.AgentEvent,
) bool {
	if !isPromptTerminalEvent(normalized.Type) {
		return false
	}
	m.dispatchTurnEnd(ctx, turnState, normalized.Timestamp)
	return loop.turnEndedShouldReturn()
}

func (m *Manager) normalizeEvent(session *Session, turnID string, event acp.AgentEvent) acp.AgentEvent {
	normalized := event
	if strings.TrimSpace(normalized.TurnID) == "" {
		normalized.TurnID = turnID
	}
	if normalized.Timestamp.IsZero() {
		normalized.Timestamp = m.now()
	}
	if session != nil {
		info := session.Info()
		if strings.TrimSpace(normalized.SessionID) == "" {
			normalized.SessionID = info.ACPSessionID
		}
	}
	return normalized
}

func (m *Manager) recordEvent(ctx context.Context, session *Session, event acp.AgentEvent) error {
	recorder := session.recorderHandle()
	if recorder == nil {
		return errors.New("session: event recorder is not available")
	}
	event = m.enrichRecordedAgentEvent(session, event)

	payload, err := marshalAgentEvent(event)
	if err != nil {
		return err
	}

	m.dispatchEventPreRecord(ctx, session, event, payload)

	persisted, err := recordPersistedSessionEvent(ctx, recorder, store.SessionEvent{
		TurnID:    event.TurnID,
		Type:      event.Type,
		AgentName: session.Info().AgentName,
		Content:   payload,
		Timestamp: event.Timestamp,
	})
	if err != nil {
		return err
	}

	if event.Usage != nil {
		if err := recorder.RecordTokenUsage(ctx, store.TokenUsage{
			TurnID:           event.Usage.TurnID,
			InputTokens:      event.Usage.InputTokens,
			OutputTokens:     event.Usage.OutputTokens,
			TotalTokens:      event.Usage.TotalTokens,
			ThoughtTokens:    event.Usage.ThoughtTokens,
			CacheReadTokens:  event.Usage.CacheReadTokens,
			CacheWriteTokens: event.Usage.CacheWriteTokens,
			ContextUsed:      event.Usage.ContextUsed,
			ContextSize:      event.Usage.ContextSize,
			CostAmount:       event.Usage.CostAmount,
			CostCurrency:     event.Usage.CostCurrency,
			Timestamp:        event.Usage.Timestamp,
		}); err != nil {
			return err
		}
	}

	m.dispatchEventPostRecord(ctx, session, event, payload, persisted.Sequence)
	m.dispatchSessionMessagePersisted(ctx, session, event, persisted, payload)

	return nil
}

type persistedSessionEventRecorder interface {
	RecordPersisted(context.Context, store.SessionEvent) (store.SessionEvent, error)
}

func recordPersistedSessionEvent(
	ctx context.Context,
	recorder EventRecorder,
	event store.SessionEvent,
) (store.SessionEvent, error) {
	if persistedRecorder, ok := recorder.(persistedSessionEventRecorder); ok {
		return persistedRecorder.RecordPersisted(ctx, event)
	}
	if err := recorder.Record(ctx, event); err != nil {
		return store.SessionEvent{}, err
	}
	return event, nil
}

func marshalAgentEvent(event acp.AgentEvent) (string, error) {
	data, err := transcript.MarshalAgentEvent(event)
	if err != nil {
		return "", fmt.Errorf("session: marshal agent event: %w", err)
	}
	return data, nil
}

func (m *Manager) enrichRecordedAgentEvent(session *Session, event acp.AgentEvent) acp.AgentEvent {
	if session == nil {
		return event
	}

	enriched := event
	correlation := enriched.Normalize()
	meta := session.CurrentPromptMeta().Normalize()
	if meta.Synthetic != nil {
		synthetic := meta.Synthetic.Normalize()
		if correlation.TaskID == "" {
			correlation.TaskID = synthetic.TaskID
		}
		if correlation.RunID == "" {
			correlation.RunID = synthetic.TaskRunID
		}
		if correlation.WorkflowID == "" {
			correlation.WorkflowID = synthetic.WorkflowID
		}
		if correlation.ClaimTokenHash == "" {
			correlation.ClaimTokenHash = synthetic.ClaimTokenHash
		}
		if correlation.CoordinatorSessionID == "" {
			correlation.CoordinatorSessionID = synthetic.CoordinatorSessionID
		}
		if correlation.SchedulerReason == "" {
			correlation.SchedulerReason = synthetic.Reason
		}
	}

	enriched.EventCorrelation = correlation.Normalize()
	return enriched
}
