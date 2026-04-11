package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	// StatusDisabled reports that the network runtime is intentionally disabled.
	StatusDisabled = "disabled"
	// StatusRunning reports a connected network runtime.
	StatusRunning = "running"
	// StatusDisconnected reports a network runtime whose transport lost its connection.
	StatusDisconnected = "disconnected"
)

// NetworkStatus is the manager-facing diagnostics snapshot consumed by daemon
// status and later transport surfaces.
type NetworkStatus struct {
	Enabled              bool
	Status               string
	ListenerHost         string
	ListenerPort         int
	LocalPeers           int
	RemotePeers          int
	Spaces               int
	QueuedMessages       int
	QueuedSessions       int
	DeliveryWorkers      int
	MessagesSent         int64
	MessagesReceived     int64
	MessagesRejected     int64
	MessagesDelivered    int64
	WorkflowTaggedEvents int64
	HandoffTaggedEvents  int64
	LastDisconnect       string
	KindMetrics          []KindMetric
}

// ManagerOption customizes network manager construction.
type ManagerOption func(*managerOptions)

type managerOptions struct {
	logger  *slog.Logger
	now     func() time.Time
	auditor AuditWriter
}

type managedSession struct {
	sessionID string
	peerID    string
	space     string
	directSub *nats.Subscription
	heartbeat *Heartbeat
}

type managedSpace struct {
	space        string
	broadcastSub *nats.Subscription
	refCount     int
}

// Manager owns transport, routing, presence, delivery, and the late-bound
// session lifecycle callbacks required by daemon boot integration.
type Manager struct {
	config       aghconfig.NetworkConfig
	logger       *slog.Logger
	now          func() time.Time
	lifecycleCtx context.Context
	cancel       context.CancelFunc

	transport  *Transport
	peers      *PeerRegistry
	router     *Router
	auditor    AuditWriter
	deliveries *deliveryCoordinator
	stats      *runtimeStats

	mu             sync.Mutex
	sessions       map[string]*managedSession
	spaces         map[string]*managedSpace
	connected      bool
	lastDisconnect string
	closed         bool
}

// WithManagerLogger overrides the logger used by the network manager.
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(opts *managerOptions) {
		opts.logger = logger
	}
}

// WithManagerClock overrides the manager clock, primarily for tests.
func WithManagerClock(now func() time.Time) ManagerOption {
	return func(opts *managerOptions) {
		opts.now = now
	}
}

// WithManagerAuditWriter injects a custom audit sink, primarily for tests.
func WithManagerAuditWriter(auditor AuditWriter) ManagerOption {
	return func(opts *managerOptions) {
		opts.auditor = auditor
	}
}

// NewManager constructs the top-level network runtime and starts the embedded
// transport it owns.
func NewManager(
	ctx context.Context,
	cfg aghconfig.NetworkConfig,
	prompter deliveryPrompter,
	auditPath string,
	auditStore AuditStore,
	opts ...ManagerOption,
) (*Manager, error) {
	if ctx == nil {
		return nil, errors.New("network: manager context is required")
	}
	if prompter == nil {
		return nil, errors.New("network: session prompter is required")
	}
	if !cfg.Enabled {
		return nil, errors.New("network: enabled network config is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("network: validate manager config: %w", err)
	}

	options := managerOptions{
		logger: slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.logger == nil {
		options.logger = slog.Default()
	}
	if options.now == nil {
		options.now = func() time.Time {
			return time.Now().UTC()
		}
	}

	lifecycleCtx, cancel := context.WithCancel(ctx)
	manager := &Manager{
		config:       cfg,
		logger:       options.logger,
		now:          options.now,
		lifecycleCtx: lifecycleCtx,
		cancel:       cancel,
		sessions:     make(map[string]*managedSession),
		spaces:       make(map[string]*managedSpace),
		connected:    true,
		stats:        newRuntimeStats(),
	}

	transport, err := NewTransport(
		lifecycleCtx,
		cfg,
		WithTransportLogger(manager.logger),
		WithTransportReconnectHandler(manager.handleReconnect),
		WithTransportDisconnectHandler(manager.handleDisconnect),
	)
	if err != nil {
		cancel()
		return nil, err
	}
	manager.transport = transport

	peers, err := NewPeerRegistry(cfg.GreetIntervalDuration(), WithPeerRegistryClock(manager.now))
	if err != nil {
		return nil, rollbackManagerInit(ctx, cancel, transport, err)
	}
	manager.peers = peers

	router, err := NewRouter(
		peers,
		transport,
		cfg.MaxReplayAgeDuration(),
		WithRouterClock(manager.now),
	)
	if err != nil {
		return nil, rollbackManagerInit(ctx, cancel, transport, err)
	}
	manager.router = router

	auditor := options.auditor
	if auditor == nil {
		auditor, err = NewAuditWriter(auditPath, auditStore)
		if err != nil {
			return nil, rollbackManagerInit(ctx, cancel, transport, err)
		}
	}
	manager.auditor = auditor

	deliveries, err := newDeliveryCoordinator(
		lifecycleCtx,
		cfg.MaxQueueDepth,
		prompter,
		withDeliveryLogger(manager.logger),
		withDeliveryClock(manager.now),
		withDeliveryDeliveredHook(manager.recordDelivered),
	)
	if err != nil {
		return nil, rollbackManagerInit(ctx, cancel, transport, err)
	}
	manager.deliveries = deliveries
	host, port := transportListener(manager.transport)
	manager.logger.Info(
		"network.started",
		"listener_host", host,
		"listener_port", port,
		"connected", true,
	)

	return manager, nil
}

func rollbackManagerInit(ctx context.Context, cancel context.CancelFunc, transport *Transport, initErr error) error {
	if cancel != nil {
		cancel()
	}
	if initErr == nil {
		return nil
	}
	if transport == nil {
		return initErr
	}

	if shutdownErr := transport.Shutdown(ctx); shutdownErr != nil {
		return errors.Join(initErr, fmt.Errorf("network: shutdown transport during manager setup: %w", shutdownErr))
	}
	return initErr
}

// JoinSpace registers one daemon-local session as a visible network peer.
func (m *Manager) JoinSpace(ctx context.Context, sessionID string, peerID string, space string) error {
	if ctx == nil {
		return errors.New("network: join context is required")
	}
	if m == nil {
		return errors.New("network: manager is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := m.lifecycleCtx.Err(); err != nil {
		return err
	}

	targetSession := strings.TrimSpace(sessionID)
	targetPeer := strings.TrimSpace(peerID)
	targetSpace := strings.TrimSpace(space)
	if targetSession == "" {
		return fmt.Errorf("%w: session id is required", ErrMissingField)
	}
	if targetPeer == "" {
		return fmt.Errorf("%w: peer id is required", ErrMissingField)
	}
	if err := ValidateSpace(targetSpace); err != nil {
		return err
	}

	if current, ok := m.sessionSnapshot(targetSession); ok {
		if current.peerID == targetPeer && current.space == targetSpace {
			return nil
		}
		if err := m.LeaveSpace(ctx, targetSession); err != nil {
			return err
		}
	}

	card, err := DefaultPeerCard(targetPeer)
	if err != nil {
		return err
	}
	local, err := m.peers.RegisterLocal(targetSession, targetSpace, card, m.now())
	if err != nil {
		return err
	}

	if err := m.acquireBroadcastSubscription(local.Space); err != nil {
		m.router.Leave(local.SessionID)
		return err
	}

	directSub, err := m.subscribeDirect(local.Space, local.PeerID)
	if err != nil {
		if releaseErr := m.releaseBroadcastSubscription(local.Space); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
		m.router.Leave(local.SessionID)
		return err
	}

	heartbeat, err := m.router.StartHeartbeat(m.lifecycleCtx, local.SessionID, "")
	if err != nil {
		if unsubscribeErr := cleanupSubscription(
			directSub.Unsubscribe,
			"network: unsubscribe direct subject for %q: %w",
			local.SessionID,
		); unsubscribeErr != nil {
			err = errors.Join(err, unsubscribeErr)
		}
		if releaseErr := m.releaseBroadcastSubscription(local.Space); releaseErr != nil {
			err = errors.Join(err, releaseErr)
		}
		m.router.Leave(local.SessionID)
		return err
	}

	m.mu.Lock()
	m.sessions[local.SessionID] = &managedSession{
		sessionID: local.SessionID,
		peerID:    local.PeerID,
		space:     local.Space,
		directSub: directSub,
		heartbeat: heartbeat,
	}
	m.mu.Unlock()

	m.logger.Info(
		"network.peer.joined",
		"session_id", local.SessionID,
		"peer_id", local.PeerID,
		"space", local.Space,
	)
	return nil
}

// LeaveSpace removes one daemon-local session from the active network runtime.
func (m *Manager) LeaveSpace(ctx context.Context, sessionID string) error {
	if ctx == nil {
		return errors.New("network: leave context is required")
	}
	if m == nil {
		return errors.New("network: manager is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	targetSession := strings.TrimSpace(sessionID)
	if targetSession == "" {
		return fmt.Errorf("%w: session id is required", ErrMissingField)
	}

	runtime, ok := m.removeSessionRuntime(targetSession)
	if !ok {
		m.deliveries.dropSession(targetSession)
		m.router.Leave(targetSession)
		return nil
	}

	var errs []error
	m.deliveries.dropSession(targetSession)

	if runtime.heartbeat != nil {
		runtime.heartbeat.Stop()
	}
	if runtime.directSub != nil {
		if err := runtime.directSub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			errs = append(errs, fmt.Errorf("network: unsubscribe direct subject for %q: %w", targetSession, err))
		}
	}
	if err := m.releaseBroadcastSubscription(runtime.space); err != nil {
		errs = append(errs, err)
	}
	m.router.Leave(targetSession)

	m.logger.Info(
		"network.peer.left",
		"session_id", runtime.sessionID,
		"peer_id", runtime.peerID,
		"space", runtime.space,
	)
	return errors.Join(errs...)
}

// OnTurnEnd wakes the per-session delivery worker after a prompt turn finishes.
func (m *Manager) OnTurnEnd(sessionID string) {
	if m == nil || m.deliveries == nil {
		return
	}
	m.deliveries.onTurnEnd(sessionID)
}

// Send publishes one outbound envelope through the owned router/transport.
func (m *Manager) Send(ctx context.Context, req SendRequest) (string, error) {
	if ctx == nil {
		return "", errors.New("network: send context is required")
	}
	if m == nil || m.router == nil {
		return "", errors.New("network: manager router is required")
	}

	result, err := m.router.Send(ctx, req)
	if err != nil {
		return "", err
	}
	m.recordAuditSent(ctx, req.SessionID, result.Envelope)
	return result.ID, nil
}

// ListPeers returns the current visible local+remote peer snapshot.
func (m *Manager) ListPeers(ctx context.Context, space string) ([]PeerInfo, error) {
	if ctx == nil {
		return nil, errors.New("network: list peers context is required")
	}
	if m == nil || m.peers == nil {
		return nil, errors.New("network: peer registry is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.peers.ListPeers(strings.TrimSpace(space), m.now()), nil
}

// ListSpaces returns the currently active runtime spaces.
func (m *Manager) ListSpaces(ctx context.Context) ([]SpaceInfo, error) {
	if ctx == nil {
		return nil, errors.New("network: list spaces context is required")
	}
	if m == nil || m.peers == nil {
		return nil, errors.New("network: peer registry is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.peers.ListSpaces(m.now()), nil
}

// Status returns a safe diagnostics snapshot without exposing transport credentials.
func (m *Manager) Status(ctx context.Context) (*NetworkStatus, error) {
	if ctx == nil {
		return nil, errors.New("network: status context is required")
	}
	if m == nil {
		return nil, errors.New("network: manager is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	peers := m.peers.ListPeers("", m.now())
	spaces := m.peers.ListSpaces(m.now())
	localPeers := 0
	for _, peer := range peers {
		if peer.Local {
			localPeers++
		}
	}
	host, port := transportListener(m.transport)
	status := StatusRunning
	connected, lastDisconnect := m.connectionState()
	if !connected {
		status = StatusDisconnected
	}
	deliveryStats := m.deliveries.stats()
	stats := m.stats.snapshot()

	return &NetworkStatus{
		Enabled:              true,
		Status:               status,
		ListenerHost:         host,
		ListenerPort:         port,
		LocalPeers:           localPeers,
		RemotePeers:          len(peers) - localPeers,
		Spaces:               len(spaces),
		QueuedMessages:       deliveryStats.QueuedMessages,
		QueuedSessions:       deliveryStats.QueuedSessions,
		DeliveryWorkers:      deliveryStats.DeliveryWorkers,
		MessagesSent:         stats.MessagesSent,
		MessagesReceived:     stats.MessagesReceived,
		MessagesRejected:     stats.MessagesRejected,
		MessagesDelivered:    stats.MessagesDelivered,
		WorkflowTaggedEvents: stats.WorkflowTaggedEvents,
		HandoffTaggedEvents:  stats.HandoffTaggedEvents,
		LastDisconnect:       lastDisconnect,
		KindMetrics:          stats.KindMetrics,
	}, nil
}

// Inbox returns the queued inbound envelopes for one local session.
func (m *Manager) Inbox(ctx context.Context, sessionID string) ([]Envelope, error) {
	if ctx == nil {
		return nil, errors.New("network: inbox context is required")
	}
	if m == nil || m.deliveries == nil {
		return nil, errors.New("network: delivery coordinator is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.deliveries.inbox(strings.TrimSpace(sessionID)), nil
}

// Shutdown drains all background work and stops the owned transport.
func (m *Manager) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("network: shutdown context is required")
	}
	if m == nil {
		return nil
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true

	sessions := make([]*managedSession, 0, len(m.sessions))
	for _, runtime := range m.sessions {
		sessions = append(sessions, runtime)
	}
	spaces := make([]*managedSpace, 0, len(m.spaces))
	for _, runtime := range m.spaces {
		spaces = append(spaces, runtime)
	}
	m.sessions = make(map[string]*managedSession)
	m.spaces = make(map[string]*managedSpace)
	m.mu.Unlock()

	deliveryStats := m.deliveries.stats()
	m.cancel()

	var errs []error
	for _, runtime := range sessions {
		if runtime == nil {
			continue
		}
		m.deliveries.dropSession(runtime.sessionID)
		if runtime.heartbeat != nil {
			runtime.heartbeat.Stop()
		}
		if runtime.directSub != nil {
			if err := runtime.directSub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
				errs = append(errs, fmt.Errorf("network: unsubscribe direct subject for %q: %w", runtime.sessionID, err))
			}
		}
		m.router.Leave(runtime.sessionID)
	}
	for _, runtime := range spaces {
		if runtime == nil || runtime.broadcastSub == nil {
			continue
		}
		if err := runtime.broadcastSub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			errs = append(errs, fmt.Errorf("network: unsubscribe broadcast subject for %q: %w", runtime.space, err))
		}
	}

	m.deliveries.wait()
	if m.transport != nil {
		if err := m.transport.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	m.logger.Info(
		"network.stopped",
		"pending_messages", deliveryStats.QueuedMessages+deliveryStats.InFlightMessages,
		"queued_messages", deliveryStats.QueuedMessages,
		"inflight_messages", deliveryStats.InFlightMessages,
		"delivery_workers", deliveryStats.DeliveryWorkers,
	)

	return errors.Join(errs...)
}

func (m *Manager) handleInboundMessage(payload []byte) {
	if m == nil || m.router == nil {
		return
	}
	if err := m.lifecycleCtx.Err(); err != nil {
		return
	}

	result, err := m.router.Receive(m.lifecycleCtx, payload)
	if err != nil {
		m.logger.Warn("network.message.receive_failed", "error", err)
		return
	}
	m.recordInboundAudit(result)

	if len(result.Deliveries) == 0 {
		return
	}
	if err := m.deliveries.accept(m.lifecycleCtx, result.Deliveries); err != nil {
		m.logger.Warn("network.message.accept_failed", "error", err)
	}
}

func (m *Manager) recordInboundAudit(result RouteResult) {
	if m == nil || m.auditor == nil {
		return
	}
	if result.Envelope != nil && result.Rejected {
		sessionID := ""
		if result.Envelope.IsDirected() {
			if target, ok := m.peers.LocalByPeer(result.Envelope.Space, *result.Envelope.To); ok {
				sessionID = target.SessionID
			}
		}
		reason := ""
		if result.ReasonCode != nil {
			reason = strings.TrimSpace(string(*result.ReasonCode))
		}
		m.recordAuditRejected(m.lifecycleCtx, sessionID, *result.Envelope, reason)
	}

	for _, delivery := range result.Deliveries {
		m.recordAuditReceived(m.lifecycleCtx, delivery.SessionID, delivery.Envelope)
	}
	for _, envelope := range result.Generated {
		local, ok := m.peers.LocalByPeer(envelope.Space, envelope.From)
		if !ok {
			continue
		}
		m.recordAuditSent(m.lifecycleCtx, local.SessionID, envelope)
	}
}

func (m *Manager) acquireBroadcastSubscription(space string) error {
	targetSpace := strings.TrimSpace(space)

	m.mu.Lock()
	if runtime, ok := m.spaces[targetSpace]; ok {
		runtime.refCount++
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	subject, err := BroadcastSubject(targetSpace)
	if err != nil {
		return err
	}
	subscription, err := m.transport.Subscribe(subject, func(msg *nats.Msg) {
		m.handleInboundMessage(msg.Data)
	})
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if runtime, ok := m.spaces[targetSpace]; ok {
		runtime.refCount++
		if err := cleanupDuplicateBroadcastSubscription(targetSpace, runtime, subscription.Unsubscribe); err != nil {
			return err
		}
		return nil
	}
	m.spaces[targetSpace] = &managedSpace{
		space:        targetSpace,
		broadcastSub: subscription,
		refCount:     1,
	}
	return nil
}

func (m *Manager) releaseBroadcastSubscription(space string) error {
	targetSpace := strings.TrimSpace(space)

	m.mu.Lock()
	runtime, ok := m.spaces[targetSpace]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	runtime.refCount--
	if runtime.refCount > 0 {
		m.mu.Unlock()
		return nil
	}
	delete(m.spaces, targetSpace)
	m.mu.Unlock()

	if runtime.broadcastSub == nil {
		return nil
	}
	if err := runtime.broadcastSub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return fmt.Errorf("network: unsubscribe broadcast subject for %q: %w", targetSpace, err)
	}
	return nil
}

func (m *Manager) subscribeDirect(space string, peerID string) (*nats.Subscription, error) {
	subject, err := DirectSubject(space, peerID)
	if err != nil {
		return nil, err
	}
	return m.transport.Subscribe(subject, func(msg *nats.Msg) {
		m.handleInboundMessage(msg.Data)
	})
}

func (m *Manager) sessionSnapshot(sessionID string) (managedSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime, ok := m.sessions[strings.TrimSpace(sessionID)]
	if !ok || runtime == nil {
		return managedSession{}, false
	}
	return *runtime, true
}

func cleanupSubscription(unsubscribe func() error, format string, value string) error {
	if unsubscribe == nil {
		return nil
	}

	if err := unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return fmt.Errorf(format, value, err)
	}
	return nil
}

func cleanupDuplicateBroadcastSubscription(space string, runtime *managedSpace, unsubscribe func() error) error {
	if err := cleanupSubscription(
		unsubscribe,
		"network: unsubscribe duplicate broadcast subject for %q: %w",
		space,
	); err != nil {
		if runtime != nil {
			runtime.refCount--
		}
		return err
	}
	return nil
}

func (m *Manager) removeSessionRuntime(sessionID string) (managedSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	target := strings.TrimSpace(sessionID)
	runtime, ok := m.sessions[target]
	if !ok || runtime == nil {
		return managedSession{}, false
	}
	delete(m.sessions, target)
	return *runtime, true
}

func (m *Manager) handleDisconnect(err error) {
	if m == nil {
		return
	}

	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}

	m.mu.Lock()
	m.connected = false
	m.lastDisconnect = message
	m.mu.Unlock()
	m.logger.Warn("network.disconnected", "error", message)
}

func (m *Manager) handleReconnect() {
	if m == nil {
		return
	}

	sessionIDs := make([]string, 0)

	m.mu.Lock()
	m.connected = true
	m.lastDisconnect = ""
	for sessionID := range m.sessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	m.mu.Unlock()

	for _, sessionID := range sessionIDs {
		if _, err := m.router.PublishGreet(m.lifecycleCtx, sessionID, ""); err != nil {
			m.logger.Warn("network.peer.regreet_failed", "session_id", sessionID, "error", err)
		}
	}
	m.logger.Info("network.reconnected", "sessions", len(sessionIDs))
}

func (m *Manager) connectionState() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected, m.lastDisconnect
}

func (m *Manager) recordAuditSent(ctx context.Context, sessionID string, envelope Envelope) {
	if m == nil || m.auditor == nil {
		return
	}
	if err := m.auditor.RecordSent(ctx, sessionID, envelope); err != nil {
		m.logger.Warn("network.audit.record_sent_failed", "session_id", sessionID, "envelope_id", envelope.ID, "error", err)
		return
	}
	m.stats.recordSent(envelope)
	m.logger.Info("network.message.sent", networkLogFields(envelope, "session_id", sessionID)...)
}

func (m *Manager) recordAuditReceived(ctx context.Context, sessionID string, envelope Envelope) {
	if m == nil || m.auditor == nil {
		return
	}
	if err := m.auditor.RecordReceived(ctx, sessionID, envelope); err != nil {
		m.logger.Warn("network.audit.record_received_failed", "session_id", sessionID, "envelope_id", envelope.ID, "error", err)
		return
	}
	m.stats.recordReceived(envelope)
	m.logger.Info("network.message.received", networkLogFields(envelope, "session_id", sessionID)...)
}

func (m *Manager) recordAuditRejected(ctx context.Context, sessionID string, envelope Envelope, reason string) {
	if m == nil || m.auditor == nil {
		return
	}
	if err := m.auditor.RecordRejected(ctx, sessionID, envelope, reason); err != nil {
		m.logger.Warn("network.audit.record_rejected_failed", "session_id", sessionID, "envelope_id", envelope.ID, "error", err)
		return
	}
	m.stats.recordRejected(envelope)
	fields := networkLogFields(envelope, "session_id", sessionID)
	fields = append(fields, "reason", strings.TrimSpace(reason))
	m.logger.Info("network.message.rejected", fields...)
}

func (m *Manager) recordDelivered(sessionID string, envelope Envelope, _ string, _ time.Duration) {
	if m == nil || m.stats == nil {
		return
	}
	m.stats.recordDelivered(envelope)
}

func transportListener(transport *Transport) (string, int) {
	if transport == nil {
		return "", 0
	}

	port := transport.Port()
	clientURL := strings.TrimSpace(transport.ClientURL())
	if clientURL == "" {
		return "127.0.0.1", port
	}

	parsed, err := url.Parse(clientURL)
	if err != nil {
		return "127.0.0.1", port
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		host = "127.0.0.1"
	}
	if parsedPort := strings.TrimSpace(parsed.Port()); parsedPort != "" {
		if value, convErr := strconv.Atoi(parsedPort); convErr == nil {
			port = value
		}
	}
	return host, port
}

func networkLogFields(envelope Envelope, extra ...any) []any {
	fields := []any{
		"message_id", strings.TrimSpace(envelope.ID),
		"kind", string(envelope.Kind),
		"space", strings.TrimSpace(envelope.Space),
		"from", strings.TrimSpace(envelope.From),
	}
	if envelope.To != nil {
		fields = append(fields, "to", strings.TrimSpace(*envelope.To))
	}
	if envelope.ReplyTo != nil {
		fields = append(fields, "reply_to", strings.TrimSpace(*envelope.ReplyTo))
	}
	if envelope.TraceID != nil {
		fields = append(fields, "trace_id", strings.TrimSpace(*envelope.TraceID))
	}
	if envelope.CausationID != nil {
		fields = append(fields, "causation_id", strings.TrimSpace(*envelope.CausationID))
	}
	if value, ok := extensionLogValue(envelope.Ext, "agh.workflow_id"); ok {
		fields = append(fields, "agh.workflow_id", value)
	}
	if value, ok := extensionLogValue(envelope.Ext, "agh.handoff_version"); ok {
		fields = append(fields, "agh.handoff_version", value)
	}
	if value, ok := extensionLogValue(envelope.Ext, "agh.handoff_digest"); ok {
		fields = append(fields, "agh.handoff_digest", value)
	}
	if value, ok := extensionLogValue(envelope.Ext, "agh.handoff_source"); ok {
		fields = append(fields, "agh.handoff_source", value)
	}
	return append(fields, extra...)
}

func extensionLogValue(ext ExtensionMap, key string) (string, bool) {
	if len(ext) == 0 {
		return "", false
	}
	raw, ok := ext[key]
	if !ok || len(raw) == 0 {
		return "", false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		text = strings.TrimSpace(text)
		if text != "" {
			return text, true
		}
	}
	return compactJSON(raw), strings.TrimSpace(compactJSON(raw)) != ""
}

func hasWorkflowID(ext ExtensionMap) bool {
	_, ok := extensionLogValue(ext, "agh.workflow_id")
	return ok
}

func hasHandoffVersion(ext ExtensionMap) bool {
	_, ok := extensionLogValue(ext, "agh.handoff_version")
	return ok
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return strings.TrimSpace(compacted.String())
}
