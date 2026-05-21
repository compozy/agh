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

// BusyInputMode selects how a user-facing prompt behaves while a session is busy.
type BusyInputMode string

const (
	BusyInputModeQueue     BusyInputMode = "queue"
	BusyInputModeInterrupt BusyInputMode = "interrupt"
	BusyInputModeSteer     BusyInputMode = "steer"
)

const promptEvidenceQueueGenerationKey = "queue_generation"

// SendPromptOpts carries one user-facing prompt plus optional busy-input mode.
type SendPromptOpts struct {
	Message string
	Mode    BusyInputMode
}

// SendPromptResult reports whether input streamed immediately or was staged.
type SendPromptResult struct {
	Status                     string
	Mode                       BusyInputMode
	Events                     <-chan acp.AgentEvent
	QueueEntryID               string
	QueuePosition              int
	QueueGeneration            int64
	EstimatedSendAt            *time.Time
	PreviousTurnID             string
	NewTurnID                  string
	Interrupted                bool
	Staged                     bool
	Queued                     bool
	CanceledQueuedEntries      int
	FallbackModeIfNoToolResult string
}

// SendPrompt submits a user-facing prompt and applies busy-input policy when a turn is active.
func (m *Manager) SendPrompt(ctx context.Context, id string, opts SendPromptOpts) (SendPromptResult, error) {
	if m == nil {
		return SendPromptResult{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return SendPromptResult{}, errors.New("session: prompt context is required")
	}
	mode, err := m.normalizeBusyInputMode(opts.Mode)
	if err != nil {
		return SendPromptResult{}, err
	}
	req, err := m.parsePromptRequest(ctx, id, PromptOpts{Message: opts.Message, TurnSource: TurnSourceUser})
	if err != nil {
		return SendPromptResult{}, err
	}
	session, err := m.lookupPromptSession(ctx, req.target)
	if err != nil {
		return SendPromptResult{}, err
	}
	if session.IsPrompting() {
		switch mode {
		case BusyInputModeQueue:
			return m.enqueueBusyPrompt(ctx, session, req)
		case BusyInputModeInterrupt:
			return m.interruptAndSubmitPrompt(ctx, session, req)
		case BusyInputModeSteer:
			return m.stageSteerPrompt(ctx, session, req)
		}
	}

	events, err := m.submitPromptRequest(ctx, req)
	if err != nil {
		return SendPromptResult{}, err
	}
	return SendPromptResult{
		Status:    "accepted",
		Mode:      mode,
		Events:    events,
		NewTurnID: req.turnID,
	}, nil
}

// InterruptPrompt cancels the active user/session turn and fences stale queued input.
func (m *Manager) InterruptPrompt(ctx context.Context, id string) (SendPromptResult, error) {
	if m == nil {
		return SendPromptResult{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return SendPromptResult{}, errors.New("session: interrupt prompt context is required")
	}
	target := strings.TrimSpace(id)
	if target == "" {
		return SendPromptResult{}, errors.New("session: session id is required")
	}
	session, err := m.lookupPromptSession(ctx, target)
	if err != nil {
		return SendPromptResult{}, err
	}
	previousTurnID := session.CurrentTurnID()
	generation, canceled, err := m.advanceInputGeneration(ctx, session.ID)
	if err != nil {
		return SendPromptResult{}, err
	}
	if err := m.CancelPrompt(ctx, session.ID); err != nil {
		return SendPromptResult{}, err
	}
	m.emitTranscriptMarker(
		ctx,
		session,
		previousTurnID,
		transcript.MarkerPromptInterrupted,
		"Prompt interrupted by operator.",
		map[string]any{
			promptEvidenceQueueGenerationKey: generation,
			"canceled_queue_entries":         canceled,
		},
	)
	return SendPromptResult{
		Status:                "interrupted",
		Mode:                  BusyInputModeInterrupt,
		PreviousTurnID:        previousTurnID,
		QueueGeneration:       generation,
		Interrupted:           true,
		CanceledQueuedEntries: canceled,
	}, nil
}

// SteerPrompt stages guidance for the active turn, falling back to queue dispatch if no tool boundary arrives.
func (m *Manager) SteerPrompt(ctx context.Context, id string, msg string) (SendPromptResult, error) {
	if m == nil {
		return SendPromptResult{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return SendPromptResult{}, errors.New("session: steer prompt context is required")
	}
	req, err := m.parsePromptRequest(ctx, id, PromptOpts{Message: msg, TurnSource: TurnSourceUser})
	if err != nil {
		return SendPromptResult{}, err
	}
	session, err := m.lookupPromptSession(ctx, req.target)
	if err != nil {
		return SendPromptResult{}, err
	}
	if !session.IsPrompting() {
		return SendPromptResult{}, fmt.Errorf("%w: %s", ErrPromptNotInProgress, session.ID)
	}
	return m.stageSteerPrompt(ctx, session, req)
}

// CancelQueuedPrompt cancels one pending busy-input queue entry.
func (m *Manager) CancelQueuedPrompt(ctx context.Context, id string, queueEntryID string) (SendPromptResult, error) {
	if m == nil {
		return SendPromptResult{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return SendPromptResult{}, errors.New("session: cancel queued prompt context is required")
	}
	if m.inputQueue == nil {
		return SendPromptResult{}, errors.New("session: input queue is not configured")
	}
	session, err := m.lookupPromptSession(ctx, id)
	if err != nil {
		return SendPromptResult{}, err
	}
	entry, err := m.inputQueue.Cancel(ctx, session.ID, queueEntryID)
	if err != nil {
		return SendPromptResult{}, err
	}
	m.emitTranscriptMarker(
		ctx,
		session,
		session.CurrentTurnID(),
		transcript.MarkerPromptDropped,
		"Queued input canceled by operator.",
		queueEntryEvidence(entry, 0),
	)
	return SendPromptResult{
		Status:          "canceled",
		Mode:            BusyInputMode(entry.Mode),
		QueueEntryID:    entry.ID,
		QueueGeneration: entry.SessionGeneration,
	}, nil
}

func (m *Manager) enqueueBusyPrompt(
	ctx context.Context,
	session *Session,
	req promptRequest,
) (SendPromptResult, error) {
	if m.inputQueue == nil {
		return SendPromptResult{}, ErrPromptInProgress
	}
	generation, err := m.currentInputGeneration(ctx, session.ID)
	if err != nil {
		return SendPromptResult{}, err
	}
	entry, position, err := m.inputQueue.Enqueue(ctx, session.ID, req.message, generation)
	if err != nil {
		if errors.Is(err, store.ErrSessionInputQueueFull) {
			m.emitTranscriptMarker(
				ctx,
				session,
				session.CurrentTurnID(),
				transcript.MarkerPromptDropped,
				"Queued input rejected because the session input queue is full.",
				map[string]any{promptEvidenceQueueGenerationKey: generation, "queue_cap": m.busyInput.QueueCap},
			)
		}
		return SendPromptResult{}, err
	}
	m.emitTranscriptMarker(
		ctx,
		session,
		session.CurrentTurnID(),
		transcript.MarkerPromptQueued,
		"Input queued while the session is busy.",
		queueEntryEvidence(entry, position),
	)
	return SendPromptResult{
		Status:          "queued",
		Mode:            BusyInputModeQueue,
		QueueEntryID:    entry.ID,
		QueuePosition:   position,
		QueueGeneration: entry.SessionGeneration,
		Queued:          true,
	}, nil
}

func (m *Manager) stageSteerPrompt(
	ctx context.Context,
	session *Session,
	req promptRequest,
) (SendPromptResult, error) {
	if m.inputQueue == nil {
		return SendPromptResult{}, ErrPromptInProgress
	}
	generation, err := m.currentInputGeneration(ctx, session.ID)
	if err != nil {
		return SendPromptResult{}, err
	}
	entry, err := m.inputQueue.StageSteer(ctx, session.ID, req.message, generation)
	if err != nil {
		return SendPromptResult{}, err
	}
	m.emitTranscriptMarker(
		ctx,
		session,
		session.CurrentTurnID(),
		transcript.MarkerPromptSteered,
		"Steering input staged while the session is busy.",
		queueEntryEvidence(entry, 0),
	)
	return SendPromptResult{
		Status:                     "staged",
		Mode:                       BusyInputModeSteer,
		QueueEntryID:               entry.ID,
		QueueGeneration:            entry.SessionGeneration,
		Staged:                     true,
		FallbackModeIfNoToolResult: string(BusyInputModeQueue),
	}, nil
}

func (m *Manager) interruptAndSubmitPrompt(
	ctx context.Context,
	session *Session,
	req promptRequest,
) (SendPromptResult, error) {
	previousTurnID := session.CurrentTurnID()
	generation, canceled, err := m.advanceInputGeneration(ctx, session.ID)
	if err != nil {
		return SendPromptResult{}, err
	}
	if err := m.CancelPrompt(ctx, session.ID); err != nil {
		return SendPromptResult{}, err
	}
	waitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.supervision.TimeoutCancelGrace)
	defer cancel()
	if err := waitForPromptIdle(waitCtx, session); err != nil {
		return SendPromptResult{}, err
	}
	events, err := m.submitPromptRequest(ctx, req)
	if err != nil {
		return SendPromptResult{}, err
	}
	m.emitTranscriptMarker(
		ctx,
		session,
		previousTurnID,
		transcript.MarkerPromptInterrupted,
		"Prompt interrupted and replaced by operator input.",
		map[string]any{
			promptEvidenceQueueGenerationKey: generation,
			"canceled_queue_entries":         canceled,
			"new_turn_id":                    req.turnID,
		},
	)
	return SendPromptResult{
		Status:                "accepted",
		Mode:                  BusyInputModeInterrupt,
		Events:                events,
		PreviousTurnID:        previousTurnID,
		NewTurnID:             req.turnID,
		QueueGeneration:       generation,
		Interrupted:           true,
		CanceledQueuedEntries: canceled,
	}, nil
}

func (m *Manager) startNextQueuedInputPrompt(sessionID string) {
	target, session, entry, ok := m.claimNextQueuedInputPrompt(sessionID)
	if !ok {
		return
	}
	if entry.Mode == store.SessionInputQueueModeSteer {
		m.emitQueuedSteerFallback(session, entry)
	}
	req, ok := m.newQueuedInputPromptRequest(session, target, entry)
	if !ok {
		return
	}
	events, err := m.submitPromptRequest(m.fallbackLifecycleContext(), req)
	if err != nil {
		m.handleQueuedInputDispatchError(session, target, entry, req, err)
		return
	}
	m.acceptQueuedInputDispatch(session, target, entry, req)
	go m.drainQueuedInputEvents(target, events)
}

func (m *Manager) claimNextQueuedInputPrompt(
	sessionID string,
) (string, *Session, store.SessionInputQueueEntry, bool) {
	if m == nil || m.inputQueue == nil {
		return "", nil, store.SessionInputQueueEntry{}, false
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return "", nil, store.SessionInputQueueEntry{}, false
	}
	session, err := m.lookupPromptSession(m.fallbackLifecycleContext(), target)
	if err != nil || session.IsPrompting() {
		return "", nil, store.SessionInputQueueEntry{}, false
	}
	entry, ok, err := m.inputQueue.ClaimNext(m.fallbackLifecycleContext(), target)
	if err != nil {
		m.sessionLogger(session).Warn("session: claim queued input failed", "error", err)
		return "", nil, store.SessionInputQueueEntry{}, false
	}
	return target, session, entry, ok
}

func (m *Manager) emitQueuedSteerFallback(session *Session, entry store.SessionInputQueueEntry) {
	evidence := queueEntryEvidence(entry, 0)
	evidence["fallback_to_queue"] = true
	m.emitTranscriptMarker(
		m.fallbackLifecycleContext(),
		session,
		session.CurrentTurnID(),
		transcript.MarkerPromptSteered,
		"Staged steering input fell back to queued dispatch.",
		evidence,
	)
}

func (m *Manager) newQueuedInputPromptRequest(
	session *Session,
	target string,
	entry store.SessionInputQueueEntry,
) (promptRequest, bool) {
	meta, err := normalizePromptMeta(
		TurnSourceUser,
		acp.PromptMeta{TurnSource: string(TurnSourceUser)},
		promptSubmissionPathUserFacing,
	)
	if err != nil {
		m.sessionLogger(session).Warn(
			"session: normalize queued input metadata failed",
			"entry_id",
			entry.ID,
			"error",
			err,
		)
		return promptRequest{}, false
	}
	return promptRequest{
		turnID:     m.newPromptTurnID(),
		target:     target,
		message:    entry.Text,
		turnSource: TurnSourceUser,
		meta:       meta,
	}, true
}

func (m *Manager) handleQueuedInputDispatchError(
	session *Session,
	target string,
	entry store.SessionInputQueueEntry,
	req promptRequest,
	cause error,
) {
	if errors.Is(cause, ErrPromptInProgress) {
		if err := m.inputQueue.Release(m.fallbackLifecycleContext(), target, entry.ID); err != nil {
			m.sessionLogger(session).Warn("session: release queued input failed", "entry_id", entry.ID, "error", err)
		}
		return
	}
	if err := m.inputQueue.MarkFailed(m.fallbackLifecycleContext(), target, entry.ID, cause.Error()); err != nil {
		m.sessionLogger(session).Warn("session: mark queued input failed", "entry_id", entry.ID, "error", err)
	}
	m.emitTranscriptMarker(
		m.fallbackLifecycleContext(),
		session,
		req.turnID,
		transcript.MarkerPromptDropped,
		"Queued input failed before dispatch.",
		queueEntryEvidence(entry, 0),
	)
	m.startNextQueuedInputPrompt(target)
}

func (m *Manager) acceptQueuedInputDispatch(
	session *Session,
	target string,
	entry store.SessionInputQueueEntry,
	req promptRequest,
) {
	if err := m.inputQueue.MarkSent(m.fallbackLifecycleContext(), target, entry.ID); err != nil {
		m.sessionLogger(session).Warn("session: mark queued input sent failed", "entry_id", entry.ID, "error", err)
	}
	m.emitTranscriptMarker(
		m.fallbackLifecycleContext(),
		session,
		req.turnID,
		transcript.MarkerPromptAccepted,
		"Queued input accepted for dispatch.",
		queueEntryEvidence(entry, 0),
	)
}

func (m *Manager) drainQueuedInputEvents(sessionID string, events <-chan acp.AgentEvent) {
	finishDrain := m.trackPromptDrain()
	defer finishDrain()
	for range events {
		continue
	}
	m.startNextQueuedInputPrompt(sessionID)
}

func (m *Manager) normalizeBusyInputMode(mode BusyInputMode) (BusyInputMode, error) {
	value := strings.TrimSpace(string(mode))
	if value == "" {
		value = strings.TrimSpace(m.busyInput.Normalize().DefaultMode)
	}
	switch BusyInputMode(value) {
	case BusyInputModeQueue, BusyInputModeInterrupt, BusyInputModeSteer:
		return BusyInputMode(value), nil
	default:
		return "", fmt.Errorf("session: invalid busy input mode %q", value)
	}
}

func (m *Manager) currentInputGeneration(ctx context.Context, sessionID string) (int64, error) {
	if m.inputQueue == nil {
		return 0, nil
	}
	return m.inputQueue.CurrentGeneration(ctx, sessionID)
}

func (m *Manager) advanceInputGeneration(ctx context.Context, sessionID string) (int64, int, error) {
	if m.inputQueue == nil {
		return 0, 0, nil
	}
	return m.inputQueue.AdvanceGeneration(ctx, sessionID)
}

func queueEntryEvidence(entry store.SessionInputQueueEntry, position int) map[string]any {
	evidence := map[string]any{
		"queue_entry_id":                 entry.ID,
		promptEvidenceQueueGenerationKey: entry.SessionGeneration,
		"queue_status":                   entry.Status,
		"mode":                           entry.Mode,
	}
	if position > 0 {
		evidence["queue_position"] = position
	}
	return evidence
}

func waitForPromptIdle(ctx context.Context, session *Session) error {
	if ctx == nil {
		return errors.New("session: wait prompt idle context is required")
	}
	if session == nil {
		return errors.New("session: session is required")
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if !session.IsPrompting() {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("session: wait for prompt interrupt: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
