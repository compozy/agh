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
	SessionID     string
	Space         string
	Kind          Kind
	To            *string
	Body          json.RawMessage
	InteractionID *string
	ReplyTo       *string
	TraceID       *string
	CausationID   *string
	ExpiresAt     *int64
	ID            *string
	Ext           ExtensionMap
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
	cancel func()
	done   chan struct{}
	once   sync.Once
}

// Stop cancels the heartbeat and waits for its goroutine to exit.
func (h *Heartbeat) Stop() {
	if h == nil {
		return
	}

	h.once.Do(func() {
		if h.cancel != nil {
			h.cancel()
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

	mu           sync.Mutex
	seen         map[string]time.Time
	interactions map[string]Interaction
}

// NewRouter constructs the routing runtime on top of a peer registry.
func NewRouter(peers *PeerRegistry, transport RouterTransport, maxReplayAge time.Duration, opts ...RouterOption) (*Router, error) {
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
		interactions: make(map[string]Interaction),
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

// PublishGreet advertises one local peer card to its joined space.
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
		Summary:  strings.TrimSpace(summary),
	})
	if err != nil {
		return SendResult{}, fmt.Errorf("network: marshal greet body: %w", err)
	}

	return r.Send(ctx, SendRequest{
		SessionID: local.SessionID,
		Space:     local.Space,
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

	heartbeatCtx, cancel := context.WithCancel(ctx)
	heartbeat := &Heartbeat{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(heartbeat.done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if _, err := r.PublishGreet(heartbeatCtx, sessionID, summary); err != nil && errors.Is(err, ErrLocalPeerNotFound) {
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
	if envelope.IsDirected() && !r.peers.HasPresence(envelope.Space, *envelope.To, now) {
		return SendResult{}, fmt.Errorf("%w: peer_id=%q space=%q", ErrTargetPeerNotFound, *envelope.To, envelope.Space)
	}

	subject, err := subjectForEnvelope(envelope)
	if err != nil {
		return SendResult{}, err
	}
	if err := r.publishEnvelope(ctx, envelope); err != nil {
		return SendResult{}, err
	}

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
		reason := reasonCodeForReceiveError(err)
		result := RouteResult{
			Rejected:   true,
			ReasonCode: &reason,
		}
		if partial := parseEnvelopeSummary(payload); partial != nil {
			result.Envelope = partial
			if partial.Kind == KindDirect {
				directedTarget, ok, resolveErr := r.resolveDirectedTarget(*partial)
				if resolveErr != nil {
					return RouteResult{}, resolveErr
				}
				if status, emit := rejectionReceiptStatus(reason); emit && ok {
					receipt, built, buildErr := buildDirectReceipt(directedTarget, *partial, now, status, reason, nil)
					if buildErr != nil {
						return RouteResult{}, buildErr
					}
					if built {
						result.Generated = append(result.Generated, receipt)
						if err := r.publishGenerated(ctx, result.Generated); err != nil {
							return RouteResult{}, err
						}
					}
				}
			}
		}
		return result, nil
	}

	result := RouteResult{Envelope: &envelope}
	directedTarget, ok, err := r.resolveDirectedTarget(envelope)
	if err != nil {
		return RouteResult{}, err
	}
	if envelope.IsDirected() && !ok {
		reason := ReasonCodeNotTarget
		result.Rejected = true
		result.ReasonCode = &reason
		return result, nil
	}

	if duplicate := r.markSeen(envelope, now); duplicate {
		reason := ReasonCodeDuplicate
		result.Duplicate = true
		result.Rejected = true
		result.ReasonCode = &reason
		if envelope.Kind == KindDirect && ok {
			receipt, built, buildErr := buildDirectReceipt(directedTarget, envelope, now, ReceiptStatusDuplicate, reason, nil)
			if buildErr != nil {
				return RouteResult{}, buildErr
			}
			if built {
				result.Generated = append(result.Generated, receipt)
				if err := r.publishGenerated(ctx, result.Generated); err != nil {
					return RouteResult{}, err
				}
			}
		}
		return result, nil
	}

	switch envelope.Kind {
	case KindGreet:
		body, decodeErr := envelope.DecodeBody()
		if decodeErr != nil {
			return RouteResult{}, decodeErr
		}
		greet := body.(GreetBody)
		if _, _, refreshErr := r.peers.RefreshRemote(envelope.Space, greet.PeerCard, now); refreshErr != nil {
			return RouteResult{}, refreshErr
		}
		return result, nil
	case KindWhois:
		return r.handleWhois(ctx, envelope, result, directedTarget, ok, now)
	case KindSay:
		result.Deliveries = deliveriesFromLocalPeers(r.peers.LocalPeers(envelope.Space), envelope)
		return result, nil
	case KindRecipe:
		deliver := true
		if envelope.InteractionID != nil && envelope.IsDirected() {
			lifecycleResult, lifecycleErr := r.applyLifecycle(envelope, now)
			switch {
			case lifecycleErr == nil:
				switch lifecycleResult.Action {
				case LifecycleActionIgnored:
					result.Ignored = true
					deliver = false
				case LifecycleActionRejectDirect:
					reason := ReasonCodeInteractionClosed
					result.Rejected = true
					result.ReasonCode = &reason
					deliver = false
				}
			case errors.Is(lifecycleErr, ErrInteractionActorNotAllowed), errors.Is(lifecycleErr, ErrInteractionNotFound):
				result.Ignored = true
				deliver = false
			case errors.Is(lifecycleErr, ErrInvalidStateTransition):
				reason := ReasonCodeInternal
				result.Rejected = true
				result.ReasonCode = &reason
				deliver = false
			default:
				return RouteResult{}, lifecycleErr
			}
		}

		if deliver {
			if envelope.IsDirected() {
				result.Deliveries = []Delivery{deliveryFromLocalPeer(directedTarget, envelope)}
			} else {
				result.Deliveries = deliveriesFromLocalPeers(r.peers.LocalPeers(envelope.Space), envelope)
			}
		}
		return result, nil
	case KindDirect, KindReceipt, KindTrace:
		deliver := true
		if envelope.InteractionID != nil {
			lifecycleResult, lifecycleErr := r.applyLifecycle(envelope, now)
			switch {
			case lifecycleErr == nil:
				switch lifecycleResult.Action {
				case LifecycleActionIgnored:
					result.Ignored = true
					deliver = false
				case LifecycleActionRejectDirect:
					reason := ReasonCodeInteractionClosed
					result.Rejected = true
					result.ReasonCode = &reason
					deliver = false
					if built, ok, buildErr := buildDirectReceipt(directedTarget, envelope, now, ReceiptStatusRejected, reason, nil); buildErr != nil {
						return RouteResult{}, buildErr
					} else if ok {
						result.Generated = append(result.Generated, built)
					}
				}
			case errors.Is(lifecycleErr, ErrInteractionActorNotAllowed), errors.Is(lifecycleErr, ErrInteractionNotFound):
				result.Ignored = true
				deliver = false
			case errors.Is(lifecycleErr, ErrInvalidStateTransition):
				reason := ReasonCodeInternal
				result.Rejected = true
				result.ReasonCode = &reason
				deliver = false
			default:
				return RouteResult{}, lifecycleErr
			}
		}

		if len(result.Generated) > 0 {
			if err := r.publishGenerated(ctx, result.Generated); err != nil {
				return RouteResult{}, err
			}
		}
		if deliver {
			result.Deliveries = []Delivery{deliveryFromLocalPeer(directedTarget, envelope)}
		}
		return result, nil
	default:
		reason := ReasonCodeUnsupportedKind
		result.Rejected = true
		result.ReasonCode = &reason
		return result, nil
	}
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
	whois := body.(WhoisBody)

	switch whois.Type {
	case WhoisTypeResponse:
		if whois.PeerCard != nil {
			if _, _, refreshErr := r.peers.RefreshRemote(envelope.Space, *whois.PeerCard, now); refreshErr != nil {
				return RouteResult{}, refreshErr
			}
		}
		if hasDirectedTarget {
			result.Deliveries = []Delivery{deliveryFromLocalPeer(directedTarget, envelope)}
		}
		return result, nil
	case WhoisTypeRequest:
		var responders []LocalPeer
		if envelope.IsDirected() {
			if hasDirectedTarget {
				responders = append(responders, directedTarget)
			}
		} else {
			responders = r.peers.MatchLocalPeers(envelope.Space, whois.Query)
		}

		for _, responder := range responders {
			payload, marshalErr := marshalEnvelopeBody(WhoisBody{Type: WhoisTypeResponse, PeerCard: &responder.PeerCard})
			if marshalErr != nil {
				return RouteResult{}, marshalErr
			}
			reply := Envelope{
				Protocol: ProtocolV0,
				ID:       store.NewID("msg"),
				Kind:     KindWhois,
				Space:    envelope.Space,
				From:     responder.PeerID,
				To:       ptrString(envelope.From),
				ReplyTo:  ptrString(envelope.ID),
				TS:       now.Unix(),
				Body:     payload,
				Proof:    nil,
			}
			if err := ValidateEnvelope(reply, ValidateOptions{Now: now, MaxReplayAge: r.maxReplayAge}); err != nil {
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
	default:
		reason := ReasonCodeMalformed
		result.Rejected = true
		result.ReasonCode = &reason
		return result, nil
	}
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

	space := strings.TrimSpace(req.Space)
	if space == "" {
		return Envelope{}, fmt.Errorf("%w: space is required", ErrMissingField)
	}
	if local.Space != space {
		return Envelope{}, fmt.Errorf("%w: session=%q space=%q", ErrLocalPeerNotFound, sessionID, space)
	}

	id := normalizeOptionalIdentifier(req.ID)
	if id == nil {
		id = ptrString(store.NewID("msg"))
	}
	envelope := Envelope{
		Protocol:      ProtocolV0,
		ID:            *id,
		Kind:          Kind(strings.TrimSpace(string(req.Kind))),
		Space:         space,
		From:          local.PeerID,
		To:            normalizeOptionalIdentifier(req.To),
		InteractionID: normalizeOptionalIdentifier(req.InteractionID),
		ReplyTo:       normalizeOptionalIdentifier(req.ReplyTo),
		TraceID:       normalizeOptionalIdentifier(req.TraceID),
		CausationID:   normalizeOptionalIdentifier(req.CausationID),
		TS:            now.Unix(),
		ExpiresAt:     cloneInt64Ptr(req.ExpiresAt),
		Body:          cloneRawMessage(req.Body),
		Ext:           cloneExtensionMap(req.Ext),
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

	local, ok := r.peers.LocalByPeer(envelope.Space, *envelope.To)
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
	key := interactionKey(envelope.Space, *envelope.InteractionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.interactions[key]
	var currentPtr *Interaction
	if ok {
		copied := current
		currentPtr = &copied
	}

	result, err := ApplyInteractionEnvelope(currentPtr, envelope, now)
	if err != nil {
		return LifecycleResult{}, err
	}
	r.interactions[key] = result.Interaction
	return result, nil
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
		return DirectSubject(envelope.Space, *envelope.To)
	}
	return BroadcastSubject(envelope.Space)
}

func replayDeadline(envelope Envelope, now time.Time, maxReplayAge time.Duration) time.Time {
	deadline := time.Unix(envelope.TS, 0).Add(maxReplayAge).UTC()
	if envelope.ExpiresAt != nil {
		expiresAt := time.Unix(*envelope.ExpiresAt, 0).UTC()
		if expiresAt.Before(deadline) {
			deadline = expiresAt
		}
	}
	if deadline.Before(now) {
		return now.Add(maxReplayAge).UTC()
	}
	return deadline
}

func interactionKey(space string, interactionID string) string {
	return strings.TrimSpace(space) + "\x00" + strings.TrimSpace(interactionID)
}

func deliveriesFromLocalPeers(peers []LocalPeer, envelope Envelope) []Delivery {
	if len(peers) == 0 {
		return nil
	}

	deliveries := make([]Delivery, 0, len(peers))
	for _, peer := range peers {
		deliveries = append(deliveries, deliveryFromLocalPeer(peer, envelope))
	}
	return deliveries
}

func deliveryFromLocalPeer(peer LocalPeer, envelope Envelope) Delivery {
	return Delivery{
		SessionID: peer.SessionID,
		PeerID:    peer.PeerID,
		Envelope:  envelope,
	}
}

func buildDirectReceipt(local LocalPeer, envelope Envelope, now time.Time, status ReceiptStatus, reason ReasonCode, detail *string) (Envelope, bool, error) {
	if envelope.Kind != KindDirect || envelope.InteractionID == nil {
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
		Protocol:      ProtocolV0,
		ID:            store.NewID("msg"),
		Kind:          KindReceipt,
		Space:         envelope.Space,
		From:          local.PeerID,
		To:            ptrString(envelope.From),
		InteractionID: ptrString(*envelope.InteractionID),
		ReplyTo:       ptrString(envelope.ID),
		TS:            now.Unix(),
		Body:          payload,
	}
	return receipt, true, nil
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
		Protocol:      strings.TrimSpace(env.Protocol),
		ID:            strings.TrimSpace(env.ID),
		Kind:          Kind(strings.TrimSpace(string(env.Kind))),
		Space:         strings.TrimSpace(env.Space),
		From:          strings.TrimSpace(env.From),
		To:            normalizeOptionalIdentifier(env.To),
		InteractionID: normalizeOptionalIdentifier(env.InteractionID),
		ReplyTo:       normalizeOptionalIdentifier(env.ReplyTo),
		TraceID:       normalizeOptionalIdentifier(env.TraceID),
		CausationID:   normalizeOptionalIdentifier(env.CausationID),
		TS:            env.TS,
		ExpiresAt:     cloneInt64Ptr(env.ExpiresAt),
		Body:          cloneRawMessage(env.Body),
		Proof:         cloneProof(env.Proof),
		Ext:           cloneExtensionMap(env.Ext),
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
	default:
		return ReasonCodeMalformed
	}
}
