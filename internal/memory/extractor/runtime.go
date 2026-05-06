package extractor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
)

const (
	defaultCoalesceMax = 16
	actorKindSubagent  = "agent_subagent"
	actorKindRoot      = "agent_root"
	messageRoleAgent   = "assistant"
	sessionTypeDream   = "dream"
	sessionTypeSystem  = "system"
)

// Runtime owns asynchronous transcript extraction and daemon inbox production.
type Runtime struct {
	root         string
	extractor    memcontract.Extractor
	producer     *Producer
	events       EventSink
	logger       *slog.Logger
	now          func() time.Time
	coalesceMax  int
	inboxPath    string
	workerCtx    context.Context
	cancelWorker context.CancelFunc

	mu             sync.Mutex
	sessions       map[string]*sessionState
	toolWrites     map[string]toolWriteMarker
	droppedTurns   int64
	coalescedTurns int64
	closed         bool
	wg             sync.WaitGroup
}

type sessionState struct {
	inFlight bool
	queued   *request
}

type request struct {
	turn          memcontract.TurnRecord
	coalesceCount int
}

type toolWriteMarker struct {
	turnSeq int64
	pending bool
}

// RuntimeStats exposes bounded operational state for daemon status APIs.
type RuntimeStats struct {
	QueuedSessions   int
	InFlightSessions int
	DroppedTurns     int64
	CoalescedTurns   int64
	Closed           bool
}

// Option customizes the extractor runtime.
type Option func(*Runtime)

// WithEventSink records extractor telemetry.
func WithEventSink(sink EventSink) Option {
	return func(r *Runtime) {
		r.events = sink
	}
}

// WithLogger configures warning output.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Runtime) {
		if logger != nil {
			r.logger = logger
		}
	}
}

// WithClock injects deterministic time for tests.
func WithClock(now func() time.Time) Option {
	return func(r *Runtime) {
		if now != nil {
			r.now = now
		}
	}
}

// WithCoalesceMax configures the hard queue merge limit.
func WithCoalesceMax(limit int) Option {
	return func(r *Runtime) {
		if limit > 0 {
			r.coalesceMax = limit
		}
	}
}

// WithInboxPath overrides the default <root>/_inbox directory.
func WithInboxPath(path string) Option {
	return func(r *Runtime) {
		if strings.TrimSpace(path) != "" {
			r.inboxPath = path
		}
	}
}

// NewRuntime constructs a daemon-owned extractor runtime.
func NewRuntime(
	ctx context.Context,
	root string,
	extractor memcontract.Extractor,
	opts ...Option,
) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("memory extractor: runtime context is required")
	}
	if extractor == nil {
		return nil, errors.New("memory extractor: extractor is required")
	}
	clean, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	now := func() time.Time {
		return time.Now().UTC()
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runtime := &Runtime{
		root:         clean,
		extractor:    extractor,
		logger:       slog.Default(),
		now:          now,
		coalesceMax:  defaultCoalesceMax,
		workerCtx:    workerCtx,
		cancelWorker: cancel,
		sessions:     make(map[string]*sessionState),
		toolWrites:   make(map[string]toolWriteMarker),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(runtime)
		}
	}
	producer, err := NewProducer(clean, runtime.now, WithProducerInboxPath(runtime.inboxPath))
	if err != nil {
		cancel()
		return nil, err
	}
	runtime.producer = producer
	return runtime, nil
}

// HandleSessionMessagePersisted converts the durable-message hook into an extractor turn.
func (r *Runtime) HandleSessionMessagePersisted(
	ctx context.Context,
	payload hookspkg.SessionMessagePersistedPayload,
) error {
	if r == nil {
		return errors.New("memory extractor: runtime is required")
	}
	if ctx == nil {
		return errors.New("memory extractor: hook context is required")
	}
	if strings.TrimSpace(payload.SessionType) == sessionTypeDream ||
		strings.TrimSpace(payload.SessionType) == sessionTypeSystem {
		return nil
	}
	if strings.TrimSpace(payload.ParentSessionID) != "" || strings.TrimSpace(payload.ActorKind) == actorKindSubagent {
		return nil
	}
	seq := payload.MessageSeq
	if seq <= 0 {
		return errors.New("memory extractor: persisted message sequence is required")
	}
	sessionID := firstNonEmpty(payload.SessionID, payload.RootSessionID)
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("memory extractor: persisted message session id is required")
	}
	if r.consumeToolWrite(sessionID, seq) {
		return nil
	}
	role := firstNonEmpty(payload.Role, messageRoleAgent)
	rootSessionID := firstNonEmpty(payload.RootSessionID, sessionID)
	actorKind := firstNonEmpty(payload.ActorKind, actorKindRoot)
	turn := memcontract.TurnRecord{
		SessionID:       sessionID,
		RootSessionID:   rootSessionID,
		AgentID:         firstNonEmpty(payload.AgentName, payload.ActorID, sessionID),
		ActorKind:       actorKind,
		WorkspaceID:     payload.WorkspaceID,
		SinceMessageSeq: seq,
		UntilMessageSeq: seq,
		Snapshot: memcontract.TranscriptSnapshot{
			Messages: []memcontract.TranscriptMessage{{
				Sequence: seq,
				Role:     role,
				Content:  payload.Text,
				At:       payload.Timestamp,
			}},
		},
		Trigger: memcontract.TriggerPostMessage,
	}
	return r.Enqueue(ctx, turn)
}

// RecordToolWrite marks an explicit root-agent memory tool write for turn-level mutual exclusion.
func (r *Runtime) RecordToolWrite(sessionID string, turnSeq int64) {
	if r == nil {
		return
	}
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.toolWrites == nil {
		r.toolWrites = make(map[string]toolWriteMarker)
	}
	r.toolWrites[trimmed] = toolWriteMarker{turnSeq: turnSeq, pending: true}
}

func (r *Runtime) consumeToolWrite(sessionID string, turnSeq int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	marker, exists := r.toolWrites[sessionID]
	if !exists || !marker.pending {
		return false
	}
	if marker.turnSeq > 0 && marker.turnSeq != turnSeq {
		if marker.turnSeq < turnSeq {
			delete(r.toolWrites, sessionID)
		}
		return false
	}
	delete(r.toolWrites, sessionID)
	return true
}

// Enqueue requests extraction for one transcript turn using one in-flight plus one queued item per session.
func (r *Runtime) Enqueue(ctx context.Context, turn memcontract.TurnRecord) error {
	if r == nil {
		return errors.New("memory extractor: runtime is required")
	}
	if ctx == nil {
		return errors.New("memory extractor: enqueue context is required")
	}
	req, err := newRequest(turn, r.now)
	if err != nil {
		return err
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return errors.New("memory extractor: runtime is closed")
	}
	state := r.sessions[req.turn.SessionID]
	if state == nil {
		state = &sessionState{}
		r.sessions[req.turn.SessionID] = state
	}
	if !state.inFlight {
		state.inFlight = true
		r.wg.Add(1)
		r.mu.Unlock()
		go r.runSession(req.turn.SessionID, req)
		return nil
	}
	if state.queued == nil {
		queued := req
		state.queued = &queued
		r.mu.Unlock()
		return nil
	}
	if state.queued.coalesceCount+1 > r.coalesceMax {
		dropped := *state.queued
		queued := req
		state.queued = &queued
		r.droppedTurns++
		r.mu.Unlock()
		r.recordEvent(ctx, Event{
			Op:   EventDropped,
			Turn: dropped.turn,
			Metadata: map[string]string{
				"dropped_until_message_seq":  strconv.FormatInt(dropped.turn.UntilMessageSeq, 10),
				"retained_until_message_seq": strconv.FormatInt(req.turn.UntilMessageSeq, 10),
			},
		})
		return nil
	}
	merged := mergeRequests(*state.queued, req)
	state.queued = &merged
	r.coalescedTurns++
	r.mu.Unlock()
	r.recordEvent(ctx, Event{
		Op:   EventCoalesced,
		Turn: merged.turn,
		Metadata: map[string]string{
			"coalesced_count": strconv.Itoa(merged.coalesceCount),
		},
	})
	return nil
}

// Stats returns current queue counters without blocking workers.
func (r *Runtime) Stats() RuntimeStats {
	if r == nil {
		return RuntimeStats{Closed: true}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	stats := RuntimeStats{
		DroppedTurns:   r.droppedTurns,
		CoalescedTurns: r.coalescedTurns,
		Closed:         r.closed,
	}
	for _, state := range r.sessions {
		if state == nil {
			continue
		}
		if state.inFlight {
			stats.InFlightSessions++
		}
		if state.queued != nil {
			stats.QueuedSessions++
		}
	}
	return stats
}

// Drain waits for the current queue to empty and asks the extractor to flush.
func (r *Runtime) Drain(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("memory extractor: drain context is required")
	}
	if err := r.waitWorkers(ctx); err != nil {
		return err
	}
	if err := r.extractor.Drain(ctx); err != nil {
		return fmt.Errorf("memory extractor: drain provider: %w", err)
	}
	return nil
}

// Close rejects new work, joins workers, and flushes the extractor.
func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("memory extractor: close context is required")
	}
	r.mu.Lock()
	alreadyClosed := r.closed
	r.closed = true
	r.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	err := r.Drain(ctx)
	if err != nil {
		r.cancelWorker()
		return err
	}
	r.cancelWorker()
	return nil
}

func (r *Runtime) waitWorkers(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("memory extractor: wait workers: %w", ctx.Err())
	}
}

func (r *Runtime) runSession(sessionID string, req request) {
	defer r.wg.Done()
	current := req
	for {
		r.process(current)
		r.mu.Lock()
		state := r.sessions[sessionID]
		if state == nil || state.queued == nil {
			delete(r.sessions, sessionID)
			r.mu.Unlock()
			return
		}
		current = *state.queued
		state.queued = nil
		r.mu.Unlock()
	}
}

func (r *Runtime) process(req request) {
	r.recordEvent(r.workerCtx, Event{
		Op:   EventStarted,
		Turn: req.turn,
		Metadata: map[string]string{
			"coalesced_count": strconv.Itoa(req.coalesceCount),
		},
	})
	candidates, err := r.extractor.Extract(r.workerCtx, req.turn)
	if err != nil {
		r.recordEvent(r.workerCtx, Event{Op: EventFailed, Turn: req.turn, Error: err.Error()})
		r.logger.Warn("memory extractor: extract failed", "session_id", req.turn.SessionID, "error", err)
		return
	}
	path, count, err := r.producer.Write(r.workerCtx, req.turn, candidates)
	if err != nil {
		r.recordEvent(r.workerCtx, Event{Op: EventFailed, Turn: req.turn, Error: err.Error()})
		r.logger.Warn("memory extractor: write inbox failed", "session_id", req.turn.SessionID, "error", err)
		return
	}
	r.recordEvent(r.workerCtx, Event{
		Op:       EventCompleted,
		Turn:     req.turn,
		TargetID: path,
		Metadata: map[string]string{
			"candidate_count": strconv.Itoa(count),
		},
	})
}

func (r *Runtime) recordEvent(ctx context.Context, event Event) {
	if r.events == nil {
		return
	}
	event.At = r.now().UTC()
	if err := r.events.RecordExtractorEvent(ctx, event); err != nil {
		r.logger.Warn("memory extractor: record event failed", "op", event.Op, "error", err)
	}
}

func newRequest(turn memcontract.TurnRecord, now func() time.Time) (request, error) {
	normalized, err := normalizeTurn(turn, now)
	if err != nil {
		return request{}, err
	}
	return request{turn: normalized}, nil
}

func normalizeTurn(turn memcontract.TurnRecord, now func() time.Time) (memcontract.TurnRecord, error) {
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	turn.SessionID = strings.TrimSpace(turn.SessionID)
	turn.RootSessionID = firstNonEmpty(turn.RootSessionID, turn.SessionID)
	turn.ParentSessionID = strings.TrimSpace(turn.ParentSessionID)
	turn.AgentID = strings.TrimSpace(turn.AgentID)
	turn.ActorKind = firstNonEmpty(turn.ActorKind, actorKindRoot)
	turn.WorkspaceID = strings.TrimSpace(turn.WorkspaceID)
	turn.Trigger = turn.Trigger.Normalize()
	if turn.SessionID == "" {
		return memcontract.TurnRecord{}, errors.New("memory extractor: session id is required")
	}
	if turn.UntilMessageSeq <= 0 {
		return memcontract.TurnRecord{}, errors.New("memory extractor: until message sequence is required")
	}
	if turn.SinceMessageSeq <= 0 {
		turn.SinceMessageSeq = turn.UntilMessageSeq
	}
	if turn.SinceMessageSeq > turn.UntilMessageSeq {
		return memcontract.TurnRecord{}, errors.New("memory extractor: since message sequence exceeds until sequence")
	}
	if turn.Trigger == "" {
		turn.Trigger = memcontract.TriggerPostMessage
	}
	if err := turn.Trigger.Validate(); err != nil {
		return memcontract.TurnRecord{}, fmt.Errorf("memory extractor: trigger: %w", err)
	}
	for idx := range turn.Snapshot.Messages {
		turn.Snapshot.Messages[idx].Role = strings.TrimSpace(turn.Snapshot.Messages[idx].Role)
		turn.Snapshot.Messages[idx].Content = strings.TrimSpace(turn.Snapshot.Messages[idx].Content)
		if turn.Snapshot.Messages[idx].At.IsZero() {
			turn.Snapshot.Messages[idx].At = now().UTC()
		}
	}
	return turn, nil
}

func mergeRequests(existing request, next request) request {
	merged := existing
	if next.turn.SinceMessageSeq < merged.turn.SinceMessageSeq {
		merged.turn.SinceMessageSeq = next.turn.SinceMessageSeq
	}
	if next.turn.UntilMessageSeq > merged.turn.UntilMessageSeq {
		merged.turn.UntilMessageSeq = next.turn.UntilMessageSeq
	}
	merged.turn.Snapshot.Messages = append(merged.turn.Snapshot.Messages, next.turn.Snapshot.Messages...)
	merged.coalesceCount++
	return merged
}

func enrichCandidate(
	candidate memcontract.Candidate,
	turn memcontract.TurnRecord,
	now func() time.Time,
) memcontract.Candidate {
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	candidate.WorkspaceID = firstNonEmpty(candidate.WorkspaceID, turn.WorkspaceID)
	candidate.Origin = candidate.Origin.Normalize()
	if candidate.Origin == "" {
		candidate.Origin = memcontract.OriginExtractor
	}
	if candidate.SubmittedAt.IsZero() {
		candidate.SubmittedAt = now().UTC()
	}
	if candidate.Metadata == nil {
		candidate.Metadata = map[string]string{}
	}
	candidate.Metadata["session_id"] = turn.SessionID
	candidate.Metadata["root_session_id"] = turn.RootSessionID
	if strings.TrimSpace(turn.ParentSessionID) != "" {
		candidate.Metadata["parent_session_id"] = turn.ParentSessionID
	}
	candidate.Metadata["actor_kind"] = turn.ActorKind
	candidate.Metadata["agent_id"] = turn.AgentID
	candidate.Metadata["since_message_seq"] = strconv.FormatInt(turn.SinceMessageSeq, 10)
	candidate.Metadata["until_message_seq"] = strconv.FormatInt(turn.UntilMessageSeq, 10)
	candidate.Metadata["trigger"] = string(turn.Trigger.Normalize())
	return candidate
}
