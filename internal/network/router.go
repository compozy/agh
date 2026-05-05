package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

var (
	// ErrLocalPeerNotFound reports an unknown local session sender.
	ErrLocalPeerNotFound = errors.New("network: local peer not found")
	// ErrTargetPeerNotFound reports a directed send target missing from presence.
	ErrTargetPeerNotFound = errors.New("network: target peer not found")
	// ErrDuplicateEnvelope reports a replay-window duplicate.
	ErrDuplicateEnvelope = errors.New("network: duplicate envelope")
	// ErrEnvelopeNotTarget reports a directed envelope for a peer this daemon does not own.
	ErrEnvelopeNotTarget = errors.New("network: envelope not targeted to a local peer")
)

// RouterTransport is the narrow publish surface consumed by the router.
type RouterTransport interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

// SendRequest carries one caller-supplied outbound envelope request.
type SendRequest struct {
	SessionID   string
	Channel     string
	Surface     *Surface
	ThreadID    *string
	DirectID    *string
	Kind        Kind
	To          *string
	Body        json.RawMessage
	WorkID      *string
	ReplyTo     *string
	TraceID     *string
	CausationID *string
	ExpiresAt   *int64
	ID          *string
	Ext         ExtensionMap
}

// SendResult summarizes one outbound publish.
type SendResult struct {
	ID       string
	Subject  string
	Envelope Envelope
}

// Delivery is one accepted inbound message delivery target.
type Delivery struct {
	SessionID string
	PeerID    string
	Envelope  Envelope
}

// RouteResult is the router decision for one inbound envelope.
type RouteResult struct {
	Envelope   *Envelope
	Deliveries []Delivery
	Generated  []Envelope
	Duplicate  bool
	Ignored    bool
	Rejected   bool
	ReasonCode *ReasonCode
}

// Heartbeat owns one periodic greet publisher.
type Heartbeat struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

// Stop cancels the heartbeat and waits for its goroutine to exit.
func (h *Heartbeat) Stop() {
	if h == nil {
		return
	}

	h.once.Do(func() {
		if h.stop != nil {
			close(h.stop)
		}
	})
	<-h.done
}

// Done returns the heartbeat completion signal.
func (h *Heartbeat) Done() <-chan struct{} {
	if h == nil {
		return nil
	}
	return h.done
}

// RouterOption customizes router construction.
type RouterOption func(*Router)

// WithRouterClock overrides the clock used for send and receive decisions.
func WithRouterClock(now func() time.Time) RouterOption {
	return func(router *Router) {
		router.now = now
	}
}

// Router handles outbound subject selection plus inbound receiver policy.
type Router struct {
	peers        *PeerRegistry
	transport    RouterTransport
	maxReplayAge time.Duration
	now          func() time.Time

	mu    sync.Mutex
	seen  map[string]time.Time
	works map[string]Work
}

type receiveState struct {
	result            RouteResult
	envelope          Envelope
	directedTarget    LocalPeer
	hasDirectedTarget bool
	now               time.Time
}

// NewRouter constructs the routing runtime on top of a peer registry.
func NewRouter(
	peers *PeerRegistry,
	transport RouterTransport,
	maxReplayAge time.Duration,
	opts ...RouterOption,
) (*Router, error) {
	if peers == nil {
		return nil, fmt.Errorf("%w: peer registry is required", ErrInvalidField)
	}
	if maxReplayAge <= 0 {
		maxReplayAge = DefaultMaxReplayAge
	}

	router := &Router{
		peers:        peers,
		transport:    transport,
		maxReplayAge: maxReplayAge,
		now:          func() time.Time { return time.Now().UTC() },
		seen:         make(map[string]time.Time),
		works:        make(map[string]Work),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(router)
		}
	}
	if router.now == nil {
		router.now = func() time.Time { return time.Now().UTC() }
	}

	return router, nil
}

// Leave removes the local sender presence for one session.
func (r *Router) Leave(sessionID string) (LocalPeer, bool) {
	if r == nil || r.peers == nil {
		return LocalPeer{}, false
	}
	return r.peers.LeaveLocal(sessionID)
}

// PublishGreet advertises one local peer card to its joined channel.
func (r *Router) PublishGreet(ctx context.Context, sessionID string, summary string) (SendResult, error) {
	if r == nil || r.peers == nil {
		return SendResult{}, fmt.Errorf("%w: router peer registry is required", ErrInvalidField)
	}

	local, ok := r.peers.LocalBySession(sessionID)
	if !ok {
		return SendResult{}, fmt.Errorf("%w: session=%q", ErrLocalPeerNotFound, strings.TrimSpace(sessionID))
	}

	body, err := json.Marshal(GreetBody{
		PeerCard: clonePeerCard(local.PeerCard),
		Summary:  ResolveGreetSummary(local.PeerCard, summary),
	})
	if err != nil {
		return SendResult{}, fmt.Errorf("network: marshal greet body: %w", err)
	}

	return r.Send(ctx, SendRequest{
		SessionID: local.SessionID,
		Channel:   local.Channel,
		Kind:      KindGreet,
		Body:      body,
	})
}

// StartHeartbeat publishes greet immediately, then keeps re-greeting on the configured interval.
func (r *Router) StartHeartbeat(ctx context.Context, sessionID string, summary string) (*Heartbeat, error) {
	if ctx == nil {
		return nil, errors.New("network: heartbeat context is required")
	}
	if r == nil || r.peers == nil {
		return nil, fmt.Errorf("%w: router peer registry is required", ErrInvalidField)
	}

	interval := r.peers.GreetInterval()
	if interval <= 0 {
		return nil, fmt.Errorf("%w: greet interval must be positive", ErrInvalidField)
	}
	if _, err := r.PublishGreet(ctx, sessionID, summary); err != nil {
		return nil, err
	}

	heartbeat := &Heartbeat{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	go func() {
		defer close(heartbeat.done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeat.stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := r.PublishGreet(
					ctx,
					sessionID,
					summary,
				); err != nil &&
					errors.Is(err, ErrLocalPeerNotFound) {
					return
				}
			}
		}
	}()

	return heartbeat, nil
}

// Send validates one outbound request, enforces presence preflight, and publishes it.
func (r *Router) Send(ctx context.Context, req SendRequest) (SendResult, error) {
	if ctx == nil {
		return SendResult{}, errors.New("network: send context is required")
	}
	if r == nil {
		return SendResult{}, errors.New("network: router is required")
	}

	now := r.now().UTC()
	envelope, err := r.buildEnvelope(req, now)
	if err != nil {
		return SendResult{}, err
	}
	if envelope.IsDirected() && !r.peers.HasPresence(envelope.Channel, *envelope.To, now) {
		return SendResult{}, fmt.Errorf(
			"%w: peer_id=%q channel=%q",
			ErrTargetPeerNotFound,
			*envelope.To,
			envelope.Channel,
		)
	}
	if err := r.validateSendLifecycle(envelope, now); err != nil {
		return SendResult{}, err
	}

	subject, err := subjectForEnvelope(envelope)
	if err != nil {
		return SendResult{}, err
	}
	if err := r.publishEnvelope(ctx, envelope); err != nil {
		return SendResult{}, err
	}
	r.syncSentLifecycle(envelope, now)

	return SendResult{
		ID:       envelope.ID,
		Subject:  subject,
		Envelope: envelope,
	}, nil
}

// Receive validates one inbound envelope, updates presence, and returns delivery decisions.
func (r *Router) Receive(ctx context.Context, payload []byte) (RouteResult, error) {
	if ctx == nil {
		return RouteResult{}, errors.New("network: receive context is required")
	}
	if r == nil {
		return RouteResult{}, errors.New("network: router is required")
	}

	now := r.now().UTC()
	envelope, err := ParseEnvelope(payload, ValidateOptions{Now: now, MaxReplayAge: r.maxReplayAge})
	if err != nil {
		return r.handleReceiveParseError(ctx, payload, now, err)
	}

	state, done, err := r.prepareReceiveState(ctx, envelope, now)
	if err != nil {
		return RouteResult{}, err
	}
	if done {
		return state.result, nil
	}
	return r.dispatchReceivedEnvelope(ctx, &state)
}

func (r *Router) handleReceiveParseError(
	ctx context.Context,
	payload []byte,
	now time.Time,
	parseErr error,
) (RouteResult, error) {
	reason := reasonCodeForReceiveError(parseErr)
	result := RouteResult{
		Rejected:   true,
		ReasonCode: &reason,
	}
	partial := parseEnvelopeSummary(payload)
	if partial == nil {
		return result, nil
	}
	result.Envelope = partial
	if !shouldEmitWorkReceipt(*partial) {
		return result, nil
	}

	directedTarget, ok, err := r.resolveDirectedTarget(*partial)
	if err != nil {
		return RouteResult{}, err
	}
	if !ok {
		return result, nil
	}
	status, emit := rejectionReceiptStatus(reason)
	result, _, err = r.appendPublishedReceipt(ctx, result, directedTarget, *partial, now, status, emit)
	return result, err
}

func (r *Router) prepareReceiveState(
	ctx context.Context,
	envelope Envelope,
	now time.Time,
) (receiveState, bool, error) {
	state := receiveState{
		result:   RouteResult{Envelope: &envelope},
		envelope: envelope,
		now:      now,
	}
	directedTarget, ok, err := r.resolveDirectedTarget(envelope)
	if err != nil {
		return receiveState{}, false, err
	}
	state.directedTarget = directedTarget
	state.hasDirectedTarget = ok
	if envelope.IsDirected() && !ok {
		reason := ReasonCodeNotTarget
		state.result.Rejected = true
		state.result.ReasonCode = &reason
		return state, true, nil
	}
	if !r.markSeen(envelope, now) {
		return state, false, nil
	}

	reason := ReasonCodeDuplicate
	state.result.Duplicate = true
	state.result.Rejected = true
	state.result.ReasonCode = &reason
	if !shouldEmitWorkReceipt(envelope) || !ok {
		return state, true, nil
	}

	result, _, err := r.appendPublishedReceipt(
		ctx,
		state.result,
		directedTarget,
		envelope,
		now,
		ReceiptStatusDuplicate,
		true,
	)
	if err != nil {
		return receiveState{}, false, err
	}
	state.result = result
	return state, true, nil
}

func (r *Router) dispatchReceivedEnvelope(ctx context.Context, state *receiveState) (RouteResult, error) {
	switch state.envelope.Kind {
	case KindGreet:
		return r.handleReceivedGreet(state)
	case KindWhois:
		return r.handleWhois(
			ctx,
			state.envelope,
			state.result,
			state.directedTarget,
			state.hasDirectedTarget,
			state.now,
		)
	case KindSay:
		return r.handleReceivedSay(ctx, state)
	case KindCapability:
		return r.handleReceivedCapability(ctx, state)
	case KindReceipt, KindTrace:
		return r.handleReceivedLifecycle(ctx, state)
	default:
		reason := ReasonCodeUnsupportedKind
		state.result.Rejected = true
		state.result.ReasonCode = &reason
		return state.result, nil
	}
}

func (r *Router) handleReceivedGreet(state *receiveState) (RouteResult, error) {
	body, err := decodeReceivedBody[GreetBody](state.envelope, "greet")
	if err != nil {
		return RouteResult{}, err
	}
	if _, _, refreshErr := r.peers.RefreshRemote(state.envelope.Channel, body.PeerCard, state.now); refreshErr != nil {
		return RouteResult{}, refreshErr
	}
	return state.result, nil
}

func (r *Router) handleReceivedSay(ctx context.Context, state *receiveState) (RouteResult, error) {
	result := state.result
	deliver := true
	if state.envelope.WorkID != nil {
		var err error
		result, deliver, err = r.applyReceiveLifecycle(ctx, state, true)
		if err != nil {
			return RouteResult{}, err
		}
	}
	if !deliver {
		return result, nil
	}
	if state.envelope.IsDirected() {
		if delivery, ok := deliveryFromLocalPeer(state.directedTarget, state.envelope); ok {
			result.Deliveries = []Delivery{delivery}
		}
		return result, nil
	}
	result.Deliveries = deliveriesFromLocalPeers(r.peers.LocalPeers(state.envelope.Channel), state.envelope)
	return result, nil
}

func (r *Router) handleReceivedCapability(ctx context.Context, state *receiveState) (RouteResult, error) {
	result, deliver, err := r.applyReceiveLifecycle(ctx, state, false)
	if err != nil {
		return RouteResult{}, err
	}
	if !deliver {
		return result, nil
	}
	if state.envelope.IsDirected() {
		if delivery, ok := deliveryFromLocalPeer(state.directedTarget, state.envelope); ok {
			result.Deliveries = []Delivery{delivery}
		}
		return result, nil
	}
	result.Deliveries = deliveriesFromLocalPeers(r.peers.LocalPeers(state.envelope.Channel), state.envelope)
	return result, nil
}

func (r *Router) handleReceivedLifecycle(ctx context.Context, state *receiveState) (RouteResult, error) {
	result, deliver, err := r.applyReceiveLifecycle(ctx, state, true)
	if err != nil {
		return RouteResult{}, err
	}
	if deliver {
		if delivery, ok := deliveryFromLocalPeer(state.directedTarget, state.envelope); ok {
			result.Deliveries = []Delivery{delivery}
		}
	}
	return result, nil
}

func (r *Router) applyReceiveLifecycle(
	ctx context.Context,
	state *receiveState,
	emitRejectionReceipt bool,
) (RouteResult, bool, error) {
	result := state.result
	if state.envelope.WorkID == nil {
		return result, true, nil
	}

	lifecycleResult, err := r.applyLifecycle(state.envelope, state.now)
	switch {
	case err == nil:
		switch lifecycleResult.Action {
		case LifecycleActionIgnored:
			result.Ignored = true
			return result, false, nil
		case LifecycleActionRejectWork:
			reason := ReasonCodeWorkClosed
			if lifecycleResult.ReasonCode != nil {
				reason = *lifecycleResult.ReasonCode
			}
			result.Rejected = true
			result.ReasonCode = &reason
			if !emitRejectionReceipt || !shouldEmitWorkReceipt(state.envelope) {
				return result, false, nil
			}
			return r.appendPublishedReceipt(
				ctx,
				result,
				state.directedTarget,
				state.envelope,
				state.now,
				ReceiptStatusRejected,
				true,
			)
		default:
			return result, true, nil
		}
	case errors.Is(err, ErrWorkContainerMismatch):
		reason := ReasonCodeWorkContainerMismatch
		result.Rejected = true
		result.ReasonCode = &reason
		if !emitRejectionReceipt || !shouldEmitWorkReceipt(state.envelope) {
			return result, false, nil
		}
		return r.appendPublishedReceipt(
			ctx,
			result,
			state.directedTarget,
			state.envelope,
			state.now,
			ReceiptStatusRejected,
			true,
		)
	case errors.Is(err, ErrWorkActorNotAllowed),
		errors.Is(err, ErrWorkNotFound):
		result.Ignored = true
		return result, false, nil
	case errors.Is(err, ErrInvalidStateTransition):
		reason := ReasonCodeInternal
		result.Rejected = true
		result.ReasonCode = &reason
		return result, false, nil
	default:
		return RouteResult{}, false, err
	}
}

func (r *Router) appendPublishedReceipt(
	ctx context.Context,
	result RouteResult,
	directedTarget LocalPeer,
	envelope Envelope,
	now time.Time,
	status ReceiptStatus,
	emit bool,
) (RouteResult, bool, error) {
	if !emit {
		return result, false, nil
	}
	receipt, built, err := buildWorkReceipt(directedTarget, envelope, now, status, *result.ReasonCode, nil)
	if err != nil {
		return RouteResult{}, false, err
	}
	if !built {
		return result, false, nil
	}

	result.Generated = append(result.Generated, receipt)
	if err := r.publishGenerated(ctx, result.Generated); err != nil {
		return RouteResult{}, false, err
	}
	return result, false, nil
}

func decodeReceivedBody[T any](env Envelope, label string) (T, error) {
	var zero T
	body, err := env.DecodeBody()
	if err != nil {
		return zero, err
	}
	typed, ok := body.(T)
	if !ok {
		return zero, fmt.Errorf("network: unexpected %s body type %T", label, body)
	}
	return typed, nil
}

func (r *Router) handleWhois(
	ctx context.Context,
	envelope Envelope,
	result RouteResult,
	directedTarget LocalPeer,
	hasDirectedTarget bool,
	now time.Time,
) (RouteResult, error) {
	body, err := envelope.DecodeBody()
	if err != nil {
		return RouteResult{}, err
	}
	whois, ok := body.(WhoisBody)
	if !ok {
		return RouteResult{}, fmt.Errorf("network: unexpected whois body type %T", body)
	}

	switch whois.Type {
	case WhoisTypeResponse:
		if whois.PeerCard != nil {
			capabilityCatalog, capabilityCatalogKnown := decodeWhoisCapabilityCatalogResponseExt(envelope.Ext)
			if _, _, refreshErr := r.peers.RefreshRemoteWithCapabilityCatalog(
				envelope.Channel,
				*whois.PeerCard,
				capabilityCatalog,
				capabilityCatalogKnown,
				now,
			); refreshErr != nil {
				return RouteResult{}, refreshErr
			}
		}
		if hasDirectedTarget {
			if delivery, ok := deliveryFromLocalPeer(directedTarget, envelope); ok {
				result.Deliveries = []Delivery{delivery}
			}
		}
		return result, nil
	case WhoisTypeRequest:
		return r.handleWhoisRequest(ctx, envelope, result, whois, directedTarget, hasDirectedTarget, now)
	default:
		reason := ReasonCodeMalformed
		result.Rejected = true
		result.ReasonCode = &reason
		return result, nil
	}
}

func (r *Router) handleWhoisRequest(
	ctx context.Context,
	envelope Envelope,
	result RouteResult,
	whois WhoisBody,
	directedTarget LocalPeer,
	hasDirectedTarget bool,
	now time.Time,
) (RouteResult, error) {
	discoveryRequest := parseWhoisCapabilityDiscoveryRequest(envelope.Ext)
	if envelope.IsDirected() && hasDirectedTarget && isEnvelopeSender(directedTarget, envelope) {
		result.Ignored = true
		return result, nil
	}
	responders := r.whoisRequestResponders(envelope, whois, directedTarget, hasDirectedTarget)

	for _, responder := range responders {
		reply, err := r.buildWhoisResponseEnvelope(envelope, responder, discoveryRequest, now)
		if err != nil {
			return RouteResult{}, err
		}
		result.Generated = append(result.Generated, reply)
	}
	if len(result.Generated) > 0 {
		if err := r.publishGenerated(ctx, result.Generated); err != nil {
			return RouteResult{}, err
		}
	}
	return result, nil
}

func (r *Router) whoisRequestResponders(
	envelope Envelope,
	whois WhoisBody,
	directedTarget LocalPeer,
	hasDirectedTarget bool,
) []LocalPeer {
	if envelope.IsDirected() {
		if hasDirectedTarget && !isEnvelopeSender(directedTarget, envelope) {
			return []LocalPeer{directedTarget}
		}
		return nil
	}
	matches := r.peers.MatchLocalPeers(envelope.Channel, whois.Query)
	responders := make([]LocalPeer, 0, len(matches))
	for _, peer := range matches {
		if isEnvelopeSender(peer, envelope) {
			continue
		}
		responders = append(responders, peer)
	}
	return responders
}

func (r *Router) buildWhoisResponseEnvelope(
	request Envelope,
	responder LocalPeer,
	discoveryRequest whoisCapabilityDiscoveryRequest,
	now time.Time,
) (Envelope, error) {
	responseCard := clonePeerCard(responder.PeerCard)
	if len(responder.CapabilityCatalog) != 0 {
		responseCatalog := responder.CapabilityCatalog
		if discoveryRequest.includeCapabilityCatalog {
			responseCatalog = selectWhoisCapabilityCatalog(
				responder.CapabilityCatalog,
				discoveryRequest.capabilityIDs,
			)
		}
		if err := applyCapabilityBriefProjection(&responseCard, responseCatalog); err != nil {
			return Envelope{}, err
		}
	}

	payload, err := marshalEnvelopeBody(WhoisBody{
		Type:     WhoisTypeResponse,
		PeerCard: &responseCard,
	})
	if err != nil {
		return Envelope{}, err
	}
	responseExt, err := buildWhoisCapabilityCatalogResponseExt(
		discoveryRequest,
		responder.CapabilityCatalog,
	)
	if err != nil {
		return Envelope{}, err
	}

	reply := Envelope{
		Protocol: ProtocolV0,
		ID:       store.NewID("msg"),
		Kind:     KindWhois,
		Channel:  request.Channel,
		From:     responder.PeerID,
		To:       ptrString(request.From),
		ReplyTo:  ptrString(request.ID),
		TS:       now.Unix(),
		Body:     payload,
		Proof:    nil,
		Ext:      responseExt,
	}
	if err := ValidateEnvelope(reply, ValidateOptions{Now: now, MaxReplayAge: r.maxReplayAge}); err != nil {
		return Envelope{}, err
	}
	if err := ensureEnvelopeSizeLimit(reply); err != nil {
		return Envelope{}, err
	}
	return reply, nil
}

func (r *Router) buildEnvelope(req SendRequest, now time.Time) (Envelope, error) {
	if r.peers == nil {
		return Envelope{}, fmt.Errorf("%w: peer registry is required", ErrInvalidField)
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Envelope{}, fmt.Errorf("%w: session is required", ErrMissingField)
	}

	local, ok := r.peers.LocalBySession(sessionID)
	if !ok {
		return Envelope{}, fmt.Errorf("%w: session=%q", ErrLocalPeerNotFound, sessionID)
	}

	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		return Envelope{}, fmt.Errorf("%w: channel is required", ErrMissingField)
	}
	if local.Channel != channel {
		return Envelope{}, fmt.Errorf("%w: session=%q channel=%q", ErrLocalPeerNotFound, sessionID, channel)
	}

	id := normalizeOptionalIdentifier(req.ID)
	if id == nil {
		id = ptrString(store.NewID("msg"))
	}
	envelope := Envelope{
		Protocol:    ProtocolV0,
		ID:          *id,
		Kind:        Kind(strings.TrimSpace(string(req.Kind))),
		Channel:     channel,
		Surface:     normalizeOptionalSurface(req.Surface),
		ThreadID:    normalizeOptionalIdentifier(req.ThreadID),
		DirectID:    normalizeOptionalIdentifier(req.DirectID),
		From:        local.PeerID,
		To:          normalizeOptionalIdentifier(req.To),
		WorkID:      normalizeOptionalIdentifier(req.WorkID),
		ReplyTo:     normalizeOptionalIdentifier(req.ReplyTo),
		TraceID:     normalizeOptionalIdentifier(req.TraceID),
		CausationID: normalizeOptionalIdentifier(req.CausationID),
		TS:          now.Unix(),
		ExpiresAt:   cloneInt64Ptr(req.ExpiresAt),
		Body:        cloneRawMessage(req.Body),
		Ext:         cloneExtensionMap(req.Ext),
	}

	if err := ValidateEnvelope(envelope, ValidateOptions{Now: now, MaxReplayAge: r.maxReplayAge}); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

func (r *Router) resolveDirectedTarget(envelope Envelope) (LocalPeer, bool, error) {
	if !envelope.IsDirected() {
		return LocalPeer{}, false, nil
	}
	if r.peers == nil {
		return LocalPeer{}, false, fmt.Errorf("%w: peer registry is required", ErrInvalidField)
	}

	local, ok := r.peers.LocalByPeer(envelope.Channel, *envelope.To)
	if !ok {
		return LocalPeer{}, false, nil
	}
	return local, true, nil
}

func (r *Router) markSeen(envelope Envelope, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, expiresAt := range r.seen {
		if !expiresAt.After(now) {
			delete(r.seen, id)
		}
	}
	if expiresAt, ok := r.seen[envelope.ID]; ok && expiresAt.After(now) {
		return true
	}
	r.seen[envelope.ID] = replayDeadline(envelope, now, r.maxReplayAge)
	return false
}

func (r *Router) applyLifecycle(envelope Envelope, now time.Time) (LifecycleResult, error) {
	key := workKey(*envelope.WorkID)

	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.evaluateLifecycleLocked(key, envelope, now)
	if err != nil {
		return LifecycleResult{}, err
	}
	r.works[key] = result.Work
	return result, nil
}

func (r *Router) evaluateLifecycle(envelope Envelope, now time.Time) (LifecycleResult, error) {
	key := workKey(*envelope.WorkID)

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.evaluateLifecycleLocked(key, envelope, now)
}

func (r *Router) evaluateLifecycleLocked(
	key string,
	envelope Envelope,
	now time.Time,
) (LifecycleResult, error) {
	current, ok := r.works[key]
	var currentPtr *Work
	if ok {
		copied := current
		currentPtr = &copied
	}

	return ApplyWorkEnvelope(currentPtr, envelope, now)
}

func (r *Router) validateSendLifecycle(envelope Envelope, now time.Time) error {
	if !shouldTrackSentLifecycle(envelope) {
		return nil
	}

	result, err := r.evaluateLifecycle(envelope, now)
	if err != nil {
		return err
	}
	switch result.Action {
	case LifecycleActionIgnored, LifecycleActionRejectWork:
		return fmt.Errorf(
			"%w: work_id=%q kind=%q",
			ErrWorkClosed,
			result.Work.ID,
			envelope.Kind,
		)
	default:
		return nil
	}
}

func (r *Router) syncSentLifecycle(envelope Envelope, now time.Time) {
	if !shouldTrackSentLifecycle(envelope) {
		return
	}
	if !r.shouldSyncSentLifecycle(envelope) {
		return
	}

	if _, err := r.applyLifecycle(envelope, now); err != nil {
		return
	}
}

func (r *Router) shouldSyncSentLifecycle(envelope Envelope) bool {
	if r == nil || r.peers == nil || !envelope.IsDirected() {
		return true
	}

	switch envelope.Kind {
	case KindReceipt, KindTrace:
		_, ok := r.peers.LocalByPeer(envelope.Channel, *envelope.To)
		return !ok
	default:
		return true
	}
}

func shouldTrackSentLifecycle(envelope Envelope) bool {
	if envelope.WorkID == nil {
		return false
	}

	switch envelope.Kind {
	case KindSay, KindCapability, KindReceipt, KindTrace:
		return true
	default:
		return false
	}
}

func (r *Router) publishEnvelope(ctx context.Context, envelope Envelope) error {
	if r.transport == nil {
		return errors.New("network: router transport is required")
	}

	subject, err := subjectForEnvelope(envelope)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("network: marshal envelope: %w", err)
	}
	if err := r.transport.Publish(ctx, subject, payload); err != nil {
		return err
	}
	return nil
}

func (r *Router) publishGenerated(ctx context.Context, envelopes []Envelope) error {
	for _, envelope := range envelopes {
		if err := r.publishEnvelope(ctx, envelope); err != nil {
			return err
		}
	}
	return nil
}

func subjectForEnvelope(envelope Envelope) (string, error) {
	if envelope.IsDirected() {
		return DirectSubject(envelope.Channel, *envelope.To)
	}
	return BroadcastSubject(envelope.Channel)
}

func replayDeadline(envelope Envelope, now time.Time, maxReplayAge time.Duration) time.Time {
	deadline := time.Unix(envelope.TS, 0).Add(maxReplayAge).UTC()
	if envelope.ExpiresAt != nil {
		expiresAt := time.Unix(*envelope.ExpiresAt, 0).UTC()
		if expiresAt.Before(deadline) {
			deadline = expiresAt
		}
	}
	minDeadline := time.Unix(now.Unix()+1, 0).UTC()
	if deadline.Before(minDeadline) {
		return minDeadline
	}
	return deadline
}

func workKey(workID string) string {
	return strings.TrimSpace(workID)
}

func deliveriesFromLocalPeers(peers []LocalPeer, envelope Envelope) []Delivery {
	if len(peers) == 0 {
		return nil
	}

	deliveries := make([]Delivery, 0, len(peers))
	for _, peer := range peers {
		delivery, ok := deliveryFromLocalPeer(peer, envelope)
		if !ok {
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries
}

func deliveryFromLocalPeer(peer LocalPeer, envelope Envelope) (Delivery, bool) {
	if strings.TrimSpace(peer.PeerID) == "" || isEnvelopeSender(peer, envelope) {
		return Delivery{}, false
	}
	return Delivery{
		SessionID: peer.SessionID,
		PeerID:    peer.PeerID,
		Envelope:  envelope,
	}, true
}

func isEnvelopeSender(peer LocalPeer, envelope Envelope) bool {
	return strings.TrimSpace(peer.PeerID) != "" &&
		strings.TrimSpace(peer.PeerID) == strings.TrimSpace(envelope.From)
}

func buildWorkReceipt(
	local LocalPeer,
	envelope Envelope,
	now time.Time,
	status ReceiptStatus,
	reason ReasonCode,
	detail *string,
) (Envelope, bool, error) {
	if !shouldEmitWorkReceipt(envelope) {
		return Envelope{}, false, nil
	}

	body := ReceiptBody{
		ForID:      envelope.ID,
		Status:     status,
		ReasonCode: &reason,
		Detail:     detail,
	}
	payload, err := marshalEnvelopeBody(body)
	if err != nil {
		return Envelope{}, false, err
	}
	receipt := Envelope{
		Protocol: ProtocolV0,
		ID:       store.NewID("msg"),
		Kind:     KindReceipt,
		Channel:  envelope.Channel,
		Surface:  cloneSurfacePtr(envelope.Surface),
		ThreadID: normalizeOptionalIdentifier(envelope.ThreadID),
		DirectID: normalizeOptionalIdentifier(envelope.DirectID),
		From:     local.PeerID,
		To:       ptrString(envelope.From),
		WorkID:   ptrString(*envelope.WorkID),
		ReplyTo:  ptrString(envelope.ID),
		TS:       now.Unix(),
		Body:     payload,
	}
	return receipt, true, nil
}

func shouldEmitWorkReceipt(envelope Envelope) bool {
	if envelope.WorkID == nil {
		return false
	}
	switch envelope.Kind {
	case KindSay, KindCapability:
		return true
	default:
		return false
	}
}

func marshalEnvelopeBody(body Body) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("network: marshal envelope body: %w", err)
	}
	return payload, nil
}

func ptrString(value string) *string {
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func parseEnvelopeSummary(data []byte) *Envelope {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil
	}

	return &Envelope{
		Protocol:    strings.TrimSpace(env.Protocol),
		ID:          strings.TrimSpace(env.ID),
		Kind:        Kind(strings.TrimSpace(string(env.Kind))),
		Channel:     strings.TrimSpace(env.Channel),
		Surface:     normalizeOptionalSurface(env.Surface),
		ThreadID:    normalizeOptionalIdentifier(env.ThreadID),
		DirectID:    normalizeOptionalIdentifier(env.DirectID),
		From:        strings.TrimSpace(env.From),
		To:          normalizeOptionalIdentifier(env.To),
		WorkID:      normalizeOptionalIdentifier(env.WorkID),
		ReplyTo:     normalizeOptionalIdentifier(env.ReplyTo),
		TraceID:     normalizeOptionalIdentifier(env.TraceID),
		CausationID: normalizeOptionalIdentifier(env.CausationID),
		TS:          env.TS,
		ExpiresAt:   cloneInt64Ptr(env.ExpiresAt),
		Body:        cloneRawMessage(env.Body),
		Proof:       cloneProof(env.Proof),
		Ext:         cloneExtensionMap(env.Ext),
	}
}

func rejectionReceiptStatus(reason ReasonCode) (ReceiptStatus, bool) {
	switch reason {
	case ReasonCodeExpired:
		return ReceiptStatusExpired, true
	case ReasonCodeMalformed:
		return ReceiptStatusRejected, true
	default:
		return "", false
	}
}

func reasonCodeForReceiveError(err error) ReasonCode {
	switch {
	case errors.Is(err, ErrExpired), errors.Is(err, ErrReplayTooOld):
		return ReasonCodeExpired
	case errors.Is(err, ErrInvalidKind):
		return ReasonCodeUnsupportedKind
	case errors.Is(err, ErrVerificationFailed):
		return ReasonCodeVerificationFailed
	default:
		return ReasonCodeMalformed
	}
}
