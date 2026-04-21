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
	session, err := m.lookupPromptSession(req.target)
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

	beginPromptSetup := session.beginPromptSetup
	if req.turnSource == TurnSourceSynthetic {
		beginPromptSetup = session.beginExclusivePromptSetup
	}
	proc, err := beginPromptSetup()
	if err != nil {
		return nil, err
	}
	defer session.finishPromptSetup()
	session.setCurrentTurnSource(turnState.turnSource)
	session.setCurrentPromptMeta(req.meta)
	clearTurnSource := true
	defer func() {
		if clearTurnSource {
			session.clearCurrentTurnSource()
			session.clearCurrentPromptMeta()
		}
	}()

	recordReq := req
	recordReq.message = message
	if err := m.recordPromptInputEvent(ctx, session, recordReq); err != nil {
		return nil, err
	}

	dispatchMessage := message
	if m.inputAugmenter != nil {
		augmented, augmentErr := m.inputAugmenter(ctx, session, message)
		if augmentErr != nil {
			return nil, fmt.Errorf("session: augment prompt input: %w", augmentErr)
		}
		if strings.TrimSpace(augmented) != "" {
			dispatchMessage = augmented
		}
	}
	source, err := m.driver.Prompt(ctx, proc, acp.PromptRequest{
		TurnID:  req.turnID,
		Message: dispatchMessage,
		Meta:    req.meta,
	})
	if err != nil {
		return nil, fmt.Errorf("session: prompt session %q: %w", req.target, err)
	}

	out := make(chan acp.AgentEvent, m.promptBufSize)
	clearTurnSource = false
	// pumpPrompt terminates when the driver closes the source channel or the request context ends.
	go m.pumpPrompt(ctx, session, turnState, source, out)
	return out, nil
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

func (m *Manager) lookupPromptSession(target string) (*Session, error) {
	session, err := m.lookup(target)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}

	meta, metaErr := m.readMeta(target)
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

// CancelPrompt cooperatively cancels the active prompt turn for a known session.
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
		if _, err := m.readMeta(target); err != nil {
			return err
		}
		return nil
	}
	if !session.IsPrompting() {
		return nil
	}

	proc := session.processHandle()
	if proc == nil {
		return errors.New("session: agent process is not available")
	}

	cancelErr := m.driver.Cancel(ctx, proc)
	if cancelErr != nil && !isProcessDone(proc) {
		return fmt.Errorf("session: cancel prompt for %q: %w", target, cancelErr)
	}
	return cancelErr
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
		meta, err := m.readMeta(target)
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
		default:
			return err
		}
	}
	return nil
}

func (m *Manager) pumpPrompt(
	ctx context.Context,
	session *Session,
	turnState *promptTurnDispatchState,
	source <-chan acp.AgentEvent,
	out chan<- acp.AgentEvent,
) {
	defer close(out)
	defer func() {
		m.finishPromptMessage(ctx, turnState, time.Time{})
		m.dispatchTurnEnd(ctx, turnState, time.Time{})
		if session != nil {
			session.clearCurrentTurnSource()
			session.clearCurrentPromptMeta()
			m.startNextQueuedSyntheticPrompt(session.ID)
		}
		notifier := m.currentTurnEndNotifier()
		if notifier != nil && session != nil {
			notifier(session.ID)
		}
	}()

	for {
		var (
			event acp.AgentEvent
			ok    bool
		)
		select {
		case <-ctx.Done():
			return
		case event, ok = <-source:
			if !ok {
				return
			}
		}

		normalized := m.normalizeEvent(session, turnState.turnID, event)
		normalized = m.preparePromptEvent(ctx, turnState, normalized)
		if err := m.recordEvent(ctx, session, normalized); err != nil {
			m.sessionLogger(session).
				Warn("session: record prompt event failed", "turn_id", turnState.turnID, "error", err)
		}
		m.notifyAgentEvent(ctx, session, normalized)

		select {
		case out <- normalized:
		case <-ctx.Done():
			return
		}

		if normalized.Type == acp.EventTypeDone || normalized.Type == acp.EventTypeError {
			m.dispatchTurnEnd(ctx, turnState, normalized.Timestamp)
			return
		}
	}
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

	payload, err := marshalAgentEvent(event)
	if err != nil {
		return err
	}

	m.dispatchEventPreRecord(ctx, session, event, payload)

	if err := recorder.Record(ctx, store.SessionEvent{
		TurnID:    event.TurnID,
		Type:      event.Type,
		AgentName: session.Info().AgentName,
		Content:   payload,
		Timestamp: event.Timestamp,
	}); err != nil {
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

	m.dispatchEventPostRecord(ctx, session, event, payload)

	return nil
}

func marshalAgentEvent(event acp.AgentEvent) (string, error) {
	data, err := transcript.MarshalAgentEvent(event)
	if err != nil {
		return "", fmt.Errorf("session: marshal agent event: %w", err)
	}
	return data, nil
}
