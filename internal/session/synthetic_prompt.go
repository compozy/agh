package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
)

// SyntheticPromptOpts carries daemon-owned synthetic prompt input plus
// wake-up metadata required for persistence and later reentry handling.
type SyntheticPromptOpts struct {
	Message  string
	Metadata acp.PromptSyntheticMeta
}

type queuedSyntheticPrompt struct {
	ctx     context.Context
	request promptRequest
	out     chan acp.AgentEvent
}

// PromptSynthetic submits one daemon-owned synthetic prompt turn.
func (m *Manager) PromptSynthetic(
	ctx context.Context,
	id string,
	opts SyntheticPromptOpts,
) (<-chan acp.AgentEvent, error) {
	req, err := m.parseSyntheticPromptRequest(ctx, id, opts)
	if err != nil {
		return nil, err
	}

	session, err := m.lookupPromptSession(req.target)
	if err != nil {
		return nil, err
	}

	dispatchCtx := context.WithoutCancel(ctx)
	if session.IsPrompting() || m.hasQueuedSyntheticPrompt(req.target) {
		return m.enqueueSyntheticPrompt(dispatchCtx, req), nil
	}

	eventsCh, err := m.submitPromptRequest(dispatchCtx, req)
	if err == nil {
		return eventsCh, nil
	}
	if !errors.Is(err, ErrPromptInProgress) {
		return nil, err
	}

	return m.enqueueSyntheticPrompt(dispatchCtx, req), nil
}

func (m *Manager) parseSyntheticPromptRequest(
	ctx context.Context,
	id string,
	opts SyntheticPromptOpts,
) (promptRequest, error) {
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

	meta, err := normalizePromptMeta(
		TurnSourceSynthetic,
		acp.PromptMeta{
			TurnSource: acp.PromptTurnSourceSynthetic,
			Synthetic:  &opts.Metadata,
		},
		promptSubmissionPathSynthetic,
	)
	if err != nil {
		return promptRequest{}, err
	}

	return promptRequest{
		turnID:     m.newPromptTurnID(),
		target:     target,
		message:    message,
		turnSource: TurnSourceSynthetic,
		meta:       meta,
	}, nil
}

func (m *Manager) enqueueSyntheticPrompt(ctx context.Context, req promptRequest) <-chan acp.AgentEvent {
	bufferSize := 1
	if m != nil && m.promptBufSize > 0 {
		bufferSize = m.promptBufSize
	}

	item := queuedSyntheticPrompt{
		ctx:     ctx,
		request: req,
		out:     make(chan acp.AgentEvent, bufferSize),
	}

	m.syntheticMu.Lock()
	m.syntheticQueues[req.target] = append(m.syntheticQueues[req.target], item)
	m.syntheticMu.Unlock()

	m.startNextQueuedSyntheticPrompt(req.target)
	return item.out
}

func (m *Manager) hasQueuedSyntheticPrompt(sessionID string) bool {
	if m == nil {
		return false
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return false
	}

	m.syntheticMu.Lock()
	defer m.syntheticMu.Unlock()
	return len(m.syntheticQueues[target]) > 0
}

func (m *Manager) startNextQueuedSyntheticPrompt(sessionID string) {
	if m == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	session, err := m.lookupPromptSession(target)
	if err != nil {
		m.failQueuedSyntheticPrompts(target, err)
		return
	}
	if session.IsPrompting() {
		return
	}

	item, ok := m.claimQueuedSyntheticPrompt(target)
	if !ok {
		return
	}

	source, err := m.submitPromptRequest(item.ctx, item.request)
	if err != nil {
		m.finishQueuedSyntheticDispatch(target)
		if errors.Is(err, ErrPromptInProgress) {
			m.requeueSyntheticPromptFront(target, item)
			return
		}
		m.emitQueuedSyntheticDispatchError(item, err)
		m.startNextQueuedSyntheticPrompt(target)
		return
	}

	go m.forwardQueuedSyntheticPrompt(target, item.out, source)
}

func (m *Manager) claimQueuedSyntheticPrompt(sessionID string) (queuedSyntheticPrompt, bool) {
	if m == nil {
		return queuedSyntheticPrompt{}, false
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return queuedSyntheticPrompt{}, false
	}

	m.syntheticMu.Lock()
	defer m.syntheticMu.Unlock()

	if m.syntheticDispatching[target] {
		return queuedSyntheticPrompt{}, false
	}

	queue := m.syntheticQueues[target]
	if len(queue) == 0 {
		return queuedSyntheticPrompt{}, false
	}

	item := queue[0]
	if len(queue) == 1 {
		delete(m.syntheticQueues, target)
	} else {
		m.syntheticQueues[target] = append([]queuedSyntheticPrompt(nil), queue[1:]...)
	}
	m.syntheticDispatching[target] = true
	return item, true
}

func (m *Manager) finishQueuedSyntheticDispatch(sessionID string) {
	if m == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	m.syntheticMu.Lock()
	defer m.syntheticMu.Unlock()
	delete(m.syntheticDispatching, target)
}

func (m *Manager) requeueSyntheticPromptFront(sessionID string, item queuedSyntheticPrompt) {
	if m == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	m.syntheticMu.Lock()
	defer m.syntheticMu.Unlock()

	queue := m.syntheticQueues[target]
	next := make([]queuedSyntheticPrompt, 0, len(queue)+1)
	next = append(next, item)
	next = append(next, queue...)
	m.syntheticQueues[target] = next
}

func (m *Manager) forwardQueuedSyntheticPrompt(
	sessionID string,
	out chan<- acp.AgentEvent,
	source <-chan acp.AgentEvent,
) {
	defer close(out)
	defer func() {
		m.finishQueuedSyntheticDispatch(sessionID)
		m.startNextQueuedSyntheticPrompt(sessionID)
	}()

	for event := range source {
		out <- event
	}
}

func (m *Manager) failQueuedSyntheticPrompts(sessionID string, err error) {
	if m == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	m.syntheticMu.Lock()
	queue := append([]queuedSyntheticPrompt(nil), m.syntheticQueues[target]...)
	delete(m.syntheticQueues, target)
	m.syntheticMu.Unlock()

	for _, item := range queue {
		m.emitQueuedSyntheticDispatchError(item, err)
	}
}

func (m *Manager) emitQueuedSyntheticDispatchError(item queuedSyntheticPrompt, err error) {
	defer close(item.out)

	if err == nil {
		return
	}

	summary := fmt.Sprintf("session: synthetic prompt dropped for %q: %v", item.request.target, err)
	if m != nil && m.logger != nil {
		m.logger.Warn(summary, "session_id", item.request.target, "turn_id", item.request.turnID)
	}

	timestamp := time.Now().UTC()
	if m != nil && m.now != nil {
		timestamp = m.now()
	}

	item.out <- acp.AgentEvent{
		Type:      acp.EventTypeError,
		TurnID:    item.request.turnID,
		Timestamp: timestamp,
		Error:     summary,
	}
}
