package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

var (
	// ErrLocalPeerNotFound reports an unknown local session sender.
	ErrLocalPeerNotFound = errors.New("network: local peer not found")
	// ErrTargetPeerNotFound reports a directed send target missing from live presence.
	ErrTargetPeerNotFound = errors.New("network: target peer not found")
)

// ThreadParticipantResolver supplies the pre-message public-thread membership snapshot.
type ThreadParticipantResolver interface {
	ListThreadParticipants(
		ctx context.Context,
		ref store.NetworkChannelRef,
		threadID string,
	) ([]store.NetworkThreadParticipant, error)
}

// SendRequest carries one caller-supplied outbound envelope request.
type SendRequest struct {
	SessionID   string
	WorkspaceID string
	Channel     string
	Surface     *Surface
	ThreadID    *string
	DirectID    *string
	Kind        Kind
	To          *string
	Mentions    []string
	Body        json.RawMessage
	WorkID      *string
	ReplyTo     *string
	TraceID     *string
	CausationID *string
	ExpiresAt   *int64
	ID          *string
	Ext         ExtensionMap
}

// RuntimeSendRequest carries one outbound envelope emitted by the AGH runtime peer.
type RuntimeSendRequest struct {
	WorkspaceID string
	Channel     string
	Surface     *Surface
	ThreadID    *string
	DirectID    *string
	Kind        Kind
	To          *string
	Mentions    []string
	Body        json.RawMessage
	WorkID      *string
	ReplyTo     *string
	TraceID     *string
	CausationID *string
	ExpiresAt   *int64
	ID          *string
	Ext         ExtensionMap
}

// SendResult is one validated envelope ready for atomic acceptance.
type SendResult struct {
	ID       string
	Envelope Envelope
}

// Delivery is one recipient selected from the immutable pre-message snapshot.
type Delivery struct {
	SessionID string
	PeerID    string
	Envelope  Envelope
	Mode      string
}

// RoutingDisposition records one selected or ignored local recipient.
type RoutingDisposition struct {
	SessionID string
	PeerID    string
	Decision  string
}

// RouteResult is the pure routing decision consumed by atomic acceptance.
type RouteResult struct {
	Envelope     Envelope
	Deliveries   []Delivery
	Dispositions []RoutingDisposition
}

// PeerLifecycleKind describes one daemon-local membership transition.
type PeerLifecycleKind string

const (
	PeerLifecycleJoined PeerLifecycleKind = "joined"
	PeerLifecycleLeft   PeerLifecycleKind = "left"
)

// PeerLifecycleEvent is the runtime-local representation of one membership transition.
type PeerLifecycleEvent struct {
	Kind      PeerLifecycleKind
	Peer      PeerInfo
	Timestamp time.Time
}

// RouterOption customizes the pure router.
type RouterOption func(*Router)

// WithRouterClock overrides the clock used for validation decisions.
func WithRouterClock(now func() time.Time) RouterOption {
	return func(router *Router) { router.now = now }
}

// WithRouterThreadParticipantResolver injects the durable participant snapshot reader.
func WithRouterThreadParticipantResolver(resolver ThreadParticipantResolver) RouterOption {
	return func(router *Router) { router.threadParticipants = resolver }
}

// WithRouterLogger overrides the logger used for routing diagnostics.
func WithRouterLogger(logger *slog.Logger) RouterOption {
	return func(router *Router) { router.logger = logger }
}

// Router validates outbound envelopes and computes recipients without side effects.
type Router struct {
	peers              *PeerRegistry
	threadParticipants ThreadParticipantResolver
	logger             *slog.Logger
	maxReplayAge       time.Duration
	now                func() time.Time
}

// NewRouter constructs a broker-free router over daemon-local live presence.
func NewRouter(peers *PeerRegistry, maxReplayAge time.Duration, opts ...RouterOption) (*Router, error) {
	if peers == nil {
		return nil, fmt.Errorf("%w: peer registry is required", ErrInvalidField)
	}
	if maxReplayAge <= 0 {
		maxReplayAge = DefaultMaxReplayAge
	}
	router := &Router{
		peers:        peers,
		logger:       slog.Default(),
		maxReplayAge: maxReplayAge,
		now:          func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		if opt != nil {
			optsApply := opt
			optsApply(router)
		}
	}
	if router.now == nil {
		router.now = func() time.Time { return time.Now().UTC() }
	}
	if router.logger == nil {
		router.logger = slog.Default()
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

// PrepareSend validates a session-originated envelope before persistence.
func (r *Router) PrepareSend(ctx context.Context, req SendRequest) (SendResult, error) {
	if ctx == nil {
		return SendResult{}, errors.New("network: send context is required")
	}
	if r == nil || r.peers == nil {
		return SendResult{}, errors.New("network: router is required")
	}
	if err := ctx.Err(); err != nil {
		return SendResult{}, err
	}
	envelope, err := r.buildSessionEnvelope(req, r.now().UTC())
	if err != nil {
		return SendResult{}, err
	}
	if err := r.requireDirectedTarget(envelope); err != nil {
		return SendResult{}, err
	}
	return SendResult{ID: envelope.ID, Envelope: envelope}, nil
}

// PrepareRuntimeSend validates an AGH-runtime-originated envelope before persistence.
func (r *Router) PrepareRuntimeSend(ctx context.Context, req RuntimeSendRequest) (SendResult, error) {
	if ctx == nil {
		return SendResult{}, errors.New("network: runtime send context is required")
	}
	if r == nil || r.peers == nil {
		return SendResult{}, errors.New("network: router is required")
	}
	if err := ctx.Err(); err != nil {
		return SendResult{}, err
	}
	envelope, err := r.buildRuntimeEnvelope(req, r.now().UTC())
	if err != nil {
		return SendResult{}, err
	}
	if err := r.requireDirectedTarget(envelope); err != nil {
		return SendResult{}, err
	}
	return SendResult{ID: envelope.ID, Envelope: envelope}, nil
}

// RoutePrepared computes the exact local recipient set from pre-message state.
func (r *Router) RoutePrepared(ctx context.Context, prepared SendResult) (RouteResult, error) {
	if ctx == nil {
		return RouteResult{}, errors.New("network: route context is required")
	}
	if r == nil || r.peers == nil {
		return RouteResult{}, errors.New("network: router is required")
	}
	if err := ctx.Err(); err != nil {
		return RouteResult{}, err
	}
	envelope := prepared.Envelope
	if err := ValidateEnvelope(envelope, ValidateOptions{
		Now: r.now().UTC(), MaxReplayAge: r.maxReplayAge,
	}); err != nil {
		return RouteResult{}, err
	}
	selected, err := r.selectedSessionSet(ctx, envelope)
	if err != nil {
		return RouteResult{}, err
	}
	locals := r.peers.LocalPeers(envelope.WorkspaceID, envelope.Channel)
	result := RouteResult{
		Envelope:     envelope,
		Deliveries:   make([]Delivery, 0, len(selected)),
		Dispositions: make([]RoutingDisposition, 0, len(locals)),
	}
	for _, peer := range locals {
		if isEnvelopeSender(peer, envelope) {
			continue
		}
		decision := store.NetworkDispositionIgnore
		if selected[peer.SessionID] && localPeerMatchesConversation(peer, envelope) {
			decision = store.NetworkDispositionDeliver
			result.Deliveries = append(result.Deliveries, Delivery{
				SessionID: peer.SessionID,
				PeerID:    peer.PeerID,
				Envelope:  envelope,
				Mode:      store.NetworkSubscriptionModeFull,
			})
		}
		result.Dispositions = append(result.Dispositions, RoutingDisposition{
			SessionID: peer.SessionID,
			PeerID:    peer.PeerID,
			Decision:  decision,
		})
	}
	sort.SliceStable(result.Deliveries, func(i, j int) bool {
		return result.Deliveries[i].SessionID < result.Deliveries[j].SessionID
	})
	sort.SliceStable(result.Dispositions, func(i, j int) bool {
		return result.Dispositions[i].SessionID < result.Dispositions[j].SessionID
	})
	return result, nil
}

func (r *Router) selectedSessionSet(ctx context.Context, envelope Envelope) (map[string]bool, error) {
	if envelope.IsDirected() {
		target, ok := r.peers.LocalByPeer(envelope.WorkspaceID, envelope.Channel, *envelope.To)
		if !ok {
			return nil, fmt.Errorf(
				"%w: peer_id=%q channel=%q",
				ErrTargetPeerNotFound,
				strings.TrimSpace(*envelope.To),
				envelope.Channel,
			)
		}
		return map[string]bool{target.SessionID: true}, nil
	}
	locals := r.peers.LocalPeers(envelope.WorkspaceID, envelope.Channel)
	selected := make(map[string]bool, len(locals))
	if !envelopeRequiresThreadSnapshot(envelope) {
		selectAllLocalRecipients(selected, locals, envelope)
		return selected, nil
	}
	participants, err := r.listThreadParticipants(ctx, envelope)
	if err != nil {
		return nil, err
	}
	if len(participants) == 0 {
		selectAllLocalRecipients(selected, locals, envelope)
	} else {
		for _, participant := range participants {
			selected[strings.TrimSpace(participant.SessionID)] = true
		}
	}
	for _, peer := range locals {
		if slicesContainsString(envelope.Mentions, peer.PeerID) {
			selected[peer.SessionID] = true
		}
	}
	return selected, nil
}

func (r *Router) listThreadParticipants(
	ctx context.Context,
	envelope Envelope,
) ([]store.NetworkThreadParticipant, error) {
	if r.threadParticipants == nil {
		return nil, nil
	}
	participants, err := r.threadParticipants.ListThreadParticipants(
		ctx,
		store.NetworkChannelRef{WorkspaceID: envelope.WorkspaceID, Channel: envelope.Channel},
		strings.TrimSpace(*envelope.ThreadID),
	)
	if err != nil {
		return nil, fmt.Errorf("network: list pre-message thread participants: %w", err)
	}
	return participants, nil
}

func selectAllLocalRecipients(selected map[string]bool, peers []LocalPeer, envelope Envelope) {
	for _, peer := range peers {
		if !isEnvelopeSender(peer, envelope) {
			selected[peer.SessionID] = true
		}
	}
}

func envelopeRequiresThreadSnapshot(envelope Envelope) bool {
	return envelope.Surface != nil && *envelope.Surface == SurfaceThread && envelope.ThreadID != nil
}

func slicesContainsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func (r *Router) requireDirectedTarget(envelope Envelope) error {
	if !envelope.IsDirected() {
		return nil
	}
	if _, ok := r.peers.LocalByPeer(envelope.WorkspaceID, envelope.Channel, *envelope.To); !ok {
		return fmt.Errorf(
			"%w: peer_id=%q channel=%q",
			ErrTargetPeerNotFound,
			strings.TrimSpace(*envelope.To),
			envelope.Channel,
		)
	}
	return nil
}

func localPeerMatchesConversation(peer LocalPeer, envelope Envelope) bool {
	if envelope.Surface == nil || *envelope.Surface != SurfaceDirect {
		return true
	}
	if envelope.To == nil {
		return false
	}
	return strings.TrimSpace(peer.PeerID) == strings.TrimSpace(*envelope.To)
}

func isEnvelopeSender(peer LocalPeer, envelope Envelope) bool {
	return strings.TrimSpace(peer.PeerID) == strings.TrimSpace(envelope.From)
}

func (r *Router) buildSessionEnvelope(req SendRequest, now time.Time) (Envelope, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Envelope{}, fmt.Errorf("%w: session is required", ErrMissingField)
	}
	local, ok := r.peers.LocalBySession(sessionID)
	if !ok {
		return Envelope{}, fmt.Errorf("%w: session=%q", ErrLocalPeerNotFound, sessionID)
	}
	channel := strings.TrimSpace(req.Channel)
	if channel == "" || channel != local.Channel {
		return Envelope{}, fmt.Errorf("%w: session=%q channel=%q", ErrLocalPeerNotFound, sessionID, channel)
	}
	if workspaceID := strings.TrimSpace(req.WorkspaceID); workspaceID != "" && workspaceID != local.WorkspaceID {
		return Envelope{}, fmt.Errorf("%w: session=%q workspace_id=%q", ErrLocalPeerNotFound, sessionID, workspaceID)
	}
	return buildEnvelope(envelopeInput{
		workspaceID: local.WorkspaceID, channel: channel, from: local.PeerID,
		directIdentityFrom: local.SessionID,
		directIdentityTo:   r.directTargetSessionID(local.WorkspaceID, channel, req.To),
		surface:            req.Surface, threadID: req.ThreadID, directID: req.DirectID, kind: req.Kind,
		to: req.To, mentions: req.Mentions, body: req.Body, workID: req.WorkID, replyTo: req.ReplyTo,
		traceID: req.TraceID, causationID: req.CausationID, expiresAt: req.ExpiresAt, id: req.ID, ext: req.Ext,
	}, now, r.maxReplayAge)
}

func (r *Router) buildRuntimeEnvelope(req RuntimeSendRequest, now time.Time) (Envelope, error) {
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if err := ValidateWorkspaceID(workspaceID); err != nil {
		return Envelope{}, err
	}
	channel := strings.TrimSpace(req.Channel)
	if err := ValidateChannel(channel); err != nil {
		return Envelope{}, err
	}
	return buildEnvelope(envelopeInput{
		workspaceID: workspaceID, channel: channel, from: RuntimePeerID,
		directIdentityFrom: runtimePeerSessionID,
		directIdentityTo:   r.directTargetSessionID(workspaceID, channel, req.To),
		surface:            req.Surface, threadID: req.ThreadID, directID: req.DirectID, kind: req.Kind,
		to: req.To, mentions: req.Mentions, body: req.Body, workID: req.WorkID, replyTo: req.ReplyTo,
		traceID: req.TraceID, causationID: req.CausationID, expiresAt: req.ExpiresAt, id: req.ID, ext: req.Ext,
	}, now, r.maxReplayAge)
}

func (r *Router) directTargetSessionID(workspaceID string, channel string, peerID *string) string {
	if peerID == nil || r == nil || r.peers == nil {
		return ""
	}
	target := strings.TrimSpace(*peerID)
	if target == "" {
		return ""
	}
	for _, peer := range r.peers.LocalPeers(workspaceID, channel) {
		if strings.TrimSpace(peer.PeerID) == target {
			return peer.SessionID
		}
	}
	return ""
}
