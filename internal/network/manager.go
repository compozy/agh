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
	sessionpkg "github.com/pedronauck/agh/internal/session"
)

const (
	// StatusDisabled reports that the network runtime is intentionally disabled.
	StatusDisabled = "disabled"
	// StatusRunning reports a connected network runtime.
	StatusRunning = "running"
	// StatusDisconnected reports a network runtime whose transport lost its connection.
	StatusDisconnected = "disconnected"

	defaultListenerHost = "127.0.0.1"
)

// Status is the manager-facing diagnostics snapshot consumed by daemon status
// and later transport surfaces.
type Status struct {
	Enabled              bool
	Status               string
	ListenerHost         string
	ListenerPort         int
	LocalPeers           int
	RemotePeers          int
	Channels             int
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
	tasks   TaskService
}

type managedSession struct {
	sessionID string
	peerID    string
	channel   string
	directSub *nats.Subscription
	heartbeat *Heartbeat
}

type managedChannel struct {
	channel      string
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
	tasks      TaskService
	deliveries *deliveryCoordinator
	stats      *runtimeStats

	mu             sync.Mutex
	sessions       map[string]*managedSession
	channels       map[string]*managedChannel
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
	if err := validateManagerInputs(ctx, cfg, prompter); err != nil {
		return nil, err
	}

	options := resolveManagerOptions(opts...)
	lifecycleCtx, cancel := context.WithCancel(ctx)
	manager := newManagerRuntime(lifecycleCtx, cfg, options, cancel)
	if err := manager.initialize(ctx, cfg, prompter, auditPath, auditStore); err != nil {
		return nil, err
	}
	manager.logStarted()
	return manager, nil
}

func validateManagerInputs(ctx context.Context, cfg aghconfig.NetworkConfig, prompter deliveryPrompter) error {
	if ctx == nil {
		return errors.New("network: manager context is required")
	}
	if prompter == nil {
		return errors.New("network: session prompter is required")
	}
	if !cfg.Enabled {
		return errors.New("network: enabled network config is required")
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("network: validate manager config: %w", err)
	}
	return nil
}

func resolveManagerOptions(opts ...ManagerOption) managerOptions {
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
	return options
}

func newManagerRuntime(
	lifecycleCtx context.Context,
	cfg aghconfig.NetworkConfig,
	options managerOptions,
	cancel context.CancelFunc,
) *Manager {
	return &Manager{
		config:       cfg,
		logger:       options.logger,
		now:          options.now,
		lifecycleCtx: lifecycleCtx,
		cancel:       cancel,
		auditor:      options.auditor,
		sessions:     make(map[string]*managedSession),
		channels:     make(map[string]*managedChannel),
		connected:    true,
		stats:        newRuntimeStats(),
		tasks:        options.tasks,
	}
}

func (m *Manager) initialize(
	ctx context.Context,
	cfg aghconfig.NetworkConfig,
	prompter deliveryPrompter,
	auditPath string,
	auditStore AuditStore,
) error {
	if err := m.initTransport(cfg); err != nil {
		m.cancel()
		return err
	}
	if err := m.initPeers(cfg); err != nil {
		return m.rollbackInit(ctx, err)
	}
	if err := m.initRouter(cfg); err != nil {
		return m.rollbackInit(ctx, err)
	}
	if err := m.initAuditor(auditPath, auditStore); err != nil {
		return m.rollbackInit(ctx, err)
	}
	if err := m.initDeliveries(cfg, prompter); err != nil {
		return m.rollbackInit(ctx, err)
	}
	return nil
}

func (m *Manager) initTransport(cfg aghconfig.NetworkConfig) error {
	transport, err := NewTransport(
		m.lifecycleCtx,
		cfg,
		WithTransportLogger(m.logger),
		WithTransportReconnectHandler(m.handleReconnect),
		WithTransportDisconnectHandler(m.handleDisconnect),
	)
	if err != nil {
		return err
	}
	m.transport = transport
	return nil
}

func (m *Manager) initPeers(cfg aghconfig.NetworkConfig) error {
	peers, err := NewPeerRegistry(cfg.GreetIntervalDuration(), WithPeerRegistryClock(m.now))
	if err != nil {
		return err
	}
	m.peers = peers
	return nil
}

func (m *Manager) initRouter(cfg aghconfig.NetworkConfig) error {
	router, err := NewRouter(
		m.peers,
		m.transport,
		cfg.MaxReplayAgeDuration(),
		WithRouterClock(m.now),
	)
	if err != nil {
		return err
	}
	m.router = router
	return nil
}

func (m *Manager) initAuditor(auditPath string, auditStore AuditStore) error {
	if m.auditor != nil {
		return nil
	}

	auditor, err := NewAuditWriter(auditPath, auditStore)
	if err != nil {
		return err
	}
	m.auditor = auditor
	return nil
}

func (m *Manager) initDeliveries(cfg aghconfig.NetworkConfig, prompter deliveryPrompter) error {
	deliveries, err := newDeliveryCoordinator(
		m.lifecycleCtx,
		cfg.MaxQueueDepth,
		prompter,
		withDeliveryLogger(m.logger),
		withDeliveryClock(m.now),
		withDeliveryDeliveredHook(m.recordDelivered),
	)
	if err != nil {
		return err
	}
	m.deliveries = deliveries
	return nil
}

func (m *Manager) rollbackInit(ctx context.Context, initErr error) error {
	return rollbackManagerInit(ctx, m.cancel, m.transport, initErr)
}

func (m *Manager) logStarted() {
	host, port := transportListener(m.transport)
	m.logger.Info(
		"network.started",
		"listener_host", host,
		"listener_port", port,
		"connected", true,
	)
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

// JoinChannel registers one daemon-local session as a visible network peer.
func (m *Manager) JoinChannel(ctx context.Context, join sessionpkg.NetworkPeerJoin) error {
	request, err := normalizeJoinChannelRequest(ctx, m, join)
	if err != nil {
		return err
	}
	local, alreadyJoined, err := m.prepareJoinLocalPeer(ctx, request)
	if err != nil {
		return err
	}
	if alreadyJoined {
		return nil
	}
	runtime, err := m.newManagedSession(local)
	if err != nil {
		return err
	}
	m.storeManagedSession(runtime)

	m.logger.Info(
		"network.peer.joined",
		"session_id", local.SessionID,
		"peer_id", local.PeerID,
		"channel", local.Channel,
	)
	return nil
}

type joinChannelRequest struct {
	sessionID    string
	peerID       string
	channel      string
	capabilities []sessionpkg.NetworkPeerCapability
}

func normalizeJoinChannelRequest(
	ctx context.Context,
	manager *Manager,
	join sessionpkg.NetworkPeerJoin,
) (joinChannelRequest, error) {
	if ctx == nil {
		return joinChannelRequest{}, errors.New("network: join context is required")
	}
	if manager == nil {
		return joinChannelRequest{}, errors.New("network: manager is required")
	}
	if err := ctx.Err(); err != nil {
		return joinChannelRequest{}, err
	}
	if err := manager.lifecycleCtx.Err(); err != nil {
		return joinChannelRequest{}, err
	}

	request := joinChannelRequest{
		sessionID:    strings.TrimSpace(join.SessionID),
		peerID:       strings.TrimSpace(join.PeerID),
		channel:      strings.TrimSpace(join.Channel),
		capabilities: cloneJoinCapabilities(join.Capabilities),
	}
	if request.sessionID == "" {
		return joinChannelRequest{}, fmt.Errorf("%w: session id is required", ErrMissingField)
	}
	if request.peerID == "" {
		return joinChannelRequest{}, fmt.Errorf("%w: peer id is required", ErrMissingField)
	}
	if err := ValidateChannel(request.channel); err != nil {
		return joinChannelRequest{}, err
	}
	return request, nil
}

func cloneJoinCapabilities(capabilities []sessionpkg.NetworkPeerCapability) []sessionpkg.NetworkPeerCapability {
	if len(capabilities) == 0 {
		return []sessionpkg.NetworkPeerCapability{}
	}

	cloned := make([]sessionpkg.NetworkPeerCapability, 0, len(capabilities))
	for _, capability := range capabilities {
		cloned = append(cloned, sessionpkg.NetworkPeerCapability{
			ID:      strings.TrimSpace(capability.ID),
			Summary: strings.TrimSpace(capability.Summary),
		})
	}
	return cloned
}

func (m *Manager) prepareJoinLocalPeer(
	ctx context.Context,
	request joinChannelRequest,
) (LocalPeer, bool, error) {
	if current, ok := m.sessionSnapshot(request.sessionID); ok {
		if current.peerID == request.peerID && current.channel == request.channel {
			return LocalPeer{}, true, nil
		}
		if err := m.LeaveChannel(ctx, request.sessionID); err != nil {
			return LocalPeer{}, false, err
		}
	}

	card, err := localPeerCardFromJoinRequest(request)
	if err != nil {
		return LocalPeer{}, false, err
	}
	local, err := m.peers.RegisterLocal(request.sessionID, request.channel, card, m.now())
	if err != nil {
		return LocalPeer{}, false, err
	}
	return local, false, nil
}

func localPeerCardFromJoinRequest(request joinChannelRequest) (PeerCard, error) {
	card, err := DefaultPeerCard(request.peerID)
	if err != nil {
		return PeerCard{}, err
	}
	card.Capabilities = capabilityIDsFromJoinRequest(request.capabilities)
	return card, nil
}

func capabilityIDsFromJoinRequest(capabilities []sessionpkg.NetworkPeerCapability) []string {
	if len(capabilities) == 0 {
		return []string{}
	}

	ids := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		ids = append(ids, strings.TrimSpace(capability.ID))
	}
	return ids
}

func (m *Manager) newManagedSession(local LocalPeer) (*managedSession, error) {
	if err := m.acquireBroadcastSubscription(local.Channel); err != nil {
		m.router.Leave(local.SessionID)
		return nil, err
	}

	directSub, err := m.subscribeDirect(local.Channel, local.PeerID)
	if err != nil {
		return nil, m.rollbackManagedSessionJoin(local, nil, err)
	}
	heartbeat, err := m.startAuditedHeartbeat(local.SessionID, "")
	if err != nil {
		return nil, m.rollbackManagedSessionJoin(local, directSub, err)
	}

	return &managedSession{
		sessionID: local.SessionID,
		peerID:    local.PeerID,
		channel:   local.Channel,
		directSub: directSub,
		heartbeat: heartbeat,
	}, nil
}

func (m *Manager) rollbackManagedSessionJoin(
	local LocalPeer,
	directSub *nats.Subscription,
	joinErr error,
) error {
	if directSub != nil {
		if unsubscribeErr := cleanupSubscription(
			directSub.Unsubscribe,
			"network: unsubscribe direct subject for %q: %w",
			local.SessionID,
		); unsubscribeErr != nil {
			joinErr = errors.Join(joinErr, unsubscribeErr)
		}
	}
	if releaseErr := m.releaseBroadcastSubscription(local.Channel); releaseErr != nil {
		joinErr = errors.Join(joinErr, releaseErr)
	}
	m.router.Leave(local.SessionID)
	return joinErr
}

func (m *Manager) storeManagedSession(runtime *managedSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[runtime.sessionID] = runtime
}

// LeaveChannel removes one daemon-local session from the active network runtime.
func (m *Manager) LeaveChannel(ctx context.Context, sessionID string) error {
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
	if err := m.releaseBroadcastSubscription(runtime.channel); err != nil {
		errs = append(errs, err)
	}
	m.router.Leave(targetSession)

	m.logger.Info(
		"network.peer.left",
		"session_id", runtime.sessionID,
		"peer_id", runtime.peerID,
		"channel", runtime.channel,
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

func (m *Manager) startAuditedHeartbeat(sessionID string, summary string) (*Heartbeat, error) {
	if m == nil || m.router == nil || m.peers == nil {
		return nil, errors.New("network: manager heartbeat dependencies are required")
	}

	interval := m.peers.GreetInterval()
	if interval <= 0 {
		return nil, fmt.Errorf("%w: greet interval must be positive", ErrInvalidField)
	}
	if err := m.publishGreetWithAudit(m.lifecycleCtx, sessionID, summary); err != nil {
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
			case <-m.lifecycleCtx.Done():
				return
			case <-ticker.C:
				if err := m.publishGreetWithAudit(m.lifecycleCtx, sessionID, summary); err != nil {
					switch {
					case errors.Is(err, context.Canceled), errors.Is(err, ErrLocalPeerNotFound):
						return
					default:
						m.logger.Warn("network.peer.heartbeat_failed", "session_id", sessionID, "error", err)
					}
				}
			}
		}
	}()

	return heartbeat, nil
}

func (m *Manager) publishGreetWithAudit(ctx context.Context, sessionID string, summary string) error {
	if m == nil || m.router == nil {
		return errors.New("network: manager router is required")
	}

	result, err := m.router.PublishGreet(ctx, sessionID, summary)
	if err != nil {
		return err
	}
	m.recordAuditSent(ctx, sessionID, result.Envelope)
	return nil
}

// ListPeers returns the current visible local+remote peer snapshot.
func (m *Manager) ListPeers(ctx context.Context, channel string) ([]PeerInfo, error) {
	if ctx == nil {
		return nil, errors.New("network: list peers context is required")
	}
	if m == nil || m.peers == nil {
		return nil, errors.New("network: peer registry is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.peers.ListPeers(strings.TrimSpace(channel), m.now()), nil
}

// ListChannels returns the currently active runtime channels.
func (m *Manager) ListChannels(ctx context.Context) ([]ChannelInfo, error) {
	if ctx == nil {
		return nil, errors.New("network: list channels context is required")
	}
	if m == nil || m.peers == nil {
		return nil, errors.New("network: peer registry is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.peers.ListChannels(m.now()), nil
}

// Status returns a safe diagnostics snapshot without exposing transport credentials.
func (m *Manager) Status(ctx context.Context) (*Status, error) {
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
	channels := m.peers.ListChannels(m.now())
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

	return &Status{
		Enabled:              true,
		Status:               status,
		ListenerHost:         host,
		ListenerPort:         port,
		LocalPeers:           localPeers,
		RemotePeers:          len(peers) - localPeers,
		Channels:             len(channels),
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
	channels := make([]*managedChannel, 0, len(m.channels))
	for _, runtime := range m.channels {
		channels = append(channels, runtime)
	}
	m.sessions = make(map[string]*managedSession)
	m.channels = make(map[string]*managedChannel)
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
				errs = append(
					errs,
					fmt.Errorf("network: unsubscribe direct subject for %q: %w", runtime.sessionID, err),
				)
			}
		}
		m.router.Leave(runtime.sessionID)
	}
	for _, runtime := range channels {
		if runtime == nil || runtime.broadcastSub == nil {
			continue
		}
		if err := runtime.broadcastSub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			errs = append(errs, fmt.Errorf("network: unsubscribe broadcast subject for %q: %w", runtime.channel, err))
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
	recordedReceivers := make(map[string]struct{})
	if result.Envelope != nil && result.Rejected {
		sessionID := ""
		if result.Envelope.IsDirected() {
			if target, ok := m.peers.LocalByPeer(result.Envelope.Channel, *result.Envelope.To); ok {
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
		recordedReceivers[delivery.SessionID] = struct{}{}
	}
	for _, sessionID := range m.controlMessageReceivers(result) {
		if _, ok := recordedReceivers[sessionID]; ok {
			continue
		}
		m.recordAuditReceived(m.lifecycleCtx, sessionID, *result.Envelope)
	}
	for _, envelope := range result.Generated {
		local, ok := m.peers.LocalByPeer(envelope.Channel, envelope.From)
		if !ok {
			continue
		}
		m.recordAuditSent(m.lifecycleCtx, local.SessionID, envelope)
	}
}

func (m *Manager) controlMessageReceivers(result RouteResult) []string {
	if m == nil || m.peers == nil || result.Envelope == nil || result.Rejected || result.Ignored {
		return nil
	}

	envelope := *result.Envelope
	switch envelope.Kind {
	case KindGreet:
		locals := m.peers.LocalPeers(envelope.Channel)
		receivers := make([]string, 0, len(locals))
		for _, local := range locals {
			receivers = append(receivers, local.SessionID)
		}
		return receivers
	case KindWhois:
		body, err := envelope.DecodeBody()
		if err != nil {
			return nil
		}
		whois, ok := body.(WhoisBody)
		if !ok || whois.Type != WhoisTypeRequest {
			return nil
		}
		if envelope.IsDirected() {
			if target, ok := m.peers.LocalByPeer(envelope.Channel, *envelope.To); ok {
				return []string{target.SessionID}
			}
			return nil
		}

		seen := make(map[string]struct{})
		receivers := make([]string, 0, len(result.Generated))
		for _, generated := range result.Generated {
			local, ok := m.peers.LocalByPeer(generated.Channel, generated.From)
			if !ok {
				continue
			}
			if _, exists := seen[local.SessionID]; exists {
				continue
			}
			seen[local.SessionID] = struct{}{}
			receivers = append(receivers, local.SessionID)
		}
		return receivers
	default:
		return nil
	}
}

func (m *Manager) acquireBroadcastSubscription(channel string) error {
	targetChannel := strings.TrimSpace(channel)

	m.mu.Lock()
	if runtime, ok := m.channels[targetChannel]; ok {
		runtime.refCount++
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	subject, err := BroadcastSubject(targetChannel)
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
	if runtime, ok := m.channels[targetChannel]; ok {
		runtime.refCount++
		if err := cleanupDuplicateBroadcastSubscription(targetChannel, runtime, subscription.Unsubscribe); err != nil {
			return err
		}
		return nil
	}
	m.channels[targetChannel] = &managedChannel{
		channel:      targetChannel,
		broadcastSub: subscription,
		refCount:     1,
	}
	return nil
}

func (m *Manager) releaseBroadcastSubscription(channel string) error {
	targetChannel := strings.TrimSpace(channel)

	m.mu.Lock()
	runtime, ok := m.channels[targetChannel]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	runtime.refCount--
	if runtime.refCount > 0 {
		m.mu.Unlock()
		return nil
	}
	delete(m.channels, targetChannel)
	m.mu.Unlock()

	if runtime.broadcastSub == nil {
		return nil
	}
	if err := runtime.broadcastSub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return fmt.Errorf("network: unsubscribe broadcast subject for %q: %w", targetChannel, err)
	}
	return nil
}

func (m *Manager) subscribeDirect(channel string, peerID string) (*nats.Subscription, error) {
	subject, err := DirectSubject(channel, peerID)
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

func cleanupDuplicateBroadcastSubscription(channel string, runtime *managedChannel, unsubscribe func() error) error {
	if err := cleanupSubscription(
		unsubscribe,
		"network: unsubscribe duplicate broadcast subject for %q: %w",
		channel,
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
		if err := m.publishGreetWithAudit(m.lifecycleCtx, sessionID, ""); err != nil {
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
		m.logger.Warn(
			"network.audit.record_sent_failed",
			"session_id",
			sessionID,
			"envelope_id",
			envelope.ID,
			"error",
			err,
		)
		return
	}
	if m.stats != nil {
		m.stats.recordSent(envelope)
	}
	m.logger.Info("network.message.sent", networkLogFields(envelope, "session_id", sessionID)...)
}

func (m *Manager) recordAuditReceived(ctx context.Context, sessionID string, envelope Envelope) {
	if m == nil || m.auditor == nil {
		return
	}
	if err := m.auditor.RecordReceived(ctx, sessionID, envelope); err != nil {
		m.logger.Warn(
			"network.audit.record_received_failed",
			"session_id",
			sessionID,
			"envelope_id",
			envelope.ID,
			"error",
			err,
		)
		return
	}
	if m.stats != nil {
		m.stats.recordReceived(envelope)
	}
	m.logger.Info("network.message.received", networkLogFields(envelope, "session_id", sessionID)...)
}

func (m *Manager) recordAuditRejected(ctx context.Context, sessionID string, envelope Envelope, reason string) {
	if m == nil || m.auditor == nil {
		return
	}
	if err := m.auditor.RecordRejected(ctx, sessionID, envelope, reason); err != nil {
		m.logger.Warn(
			"network.audit.record_rejected_failed",
			"session_id",
			sessionID,
			"envelope_id",
			envelope.ID,
			"error",
			err,
		)
		return
	}
	if m.stats != nil {
		m.stats.recordRejected(envelope)
	}
	fields := networkLogFields(envelope, "session_id", sessionID)
	fields = append(fields, "reason", strings.TrimSpace(reason))
	m.logger.Info("network.message.rejected", fields...)
}

func (m *Manager) recordDelivered(sessionID string, envelope Envelope, _ string, _ time.Duration) {
	if m == nil {
		return
	}
	if m.auditor != nil {
		if err := m.auditor.RecordDelivered(m.lifecycleCtx, sessionID, envelope); err != nil {
			m.logger.Warn(
				"network.audit.record_delivered_failed",
				"session_id",
				sessionID,
				"envelope_id",
				envelope.ID,
				"error",
				err,
			)
		}
	}
	if m.stats == nil {
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
		return defaultListenerHost, port
	}

	parsed, err := url.Parse(clientURL)
	if err != nil {
		return defaultListenerHost, port
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		host = defaultListenerHost
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
		"channel", strings.TrimSpace(envelope.Channel),
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
	compacted := compactJSON(raw)
	return compacted, compacted != ""
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
