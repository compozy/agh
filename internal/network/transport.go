package network

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	defaultTransportReadyTimeout    = 5 * time.Second
	defaultTransportShutdownTimeout = 5 * time.Second
	defaultTransportPublishTimeout  = 5 * time.Second
	subjectPrefix                   = "agh.network.v0"
	maxNATSPayloadBytes             = int(^uint32(0) >> 1)
)

// TransportOption customizes embedded transport startup behavior.
type TransportOption func(*transportOptions)

type transportOptions struct {
	logger         *slog.Logger
	readyTimeout   time.Duration
	publishTimeout time.Duration
	onReconnect    func()
	onDisconnect   func(error)
}

// WithTransportLogger overrides the logger used by the transport.
func WithTransportLogger(logger *slog.Logger) TransportOption {
	return func(opts *transportOptions) {
		opts.logger = logger
	}
}

// WithTransportReadyTimeout overrides the server readiness timeout.
func WithTransportReadyTimeout(timeout time.Duration) TransportOption {
	return func(opts *transportOptions) {
		opts.readyTimeout = timeout
	}
}

// WithTransportPublishTimeout overrides the publish flush timeout when the
// caller does not provide a deadline.
func WithTransportPublishTimeout(timeout time.Duration) TransportOption {
	return func(opts *transportOptions) {
		opts.publishTimeout = timeout
	}
}

// WithTransportReconnectHandler registers a reconnect callback for later
// manager wiring.
func WithTransportReconnectHandler(handler func()) TransportOption {
	return func(opts *transportOptions) {
		opts.onReconnect = handler
	}
}

// WithTransportDisconnectHandler registers a disconnect callback for later
// manager wiring.
func WithTransportDisconnectHandler(handler func(error)) TransportOption {
	return func(opts *transportOptions) {
		opts.onDisconnect = handler
	}
}

// Transport owns the embedded NATS server plus the daemon's in-process
// connection.
type Transport struct {
	logger         *slog.Logger
	server         *server.Server
	conn           *nats.Conn
	token          string
	port           int
	publishTimeout time.Duration
	closedCh       chan struct{}

	mu      sync.Mutex
	drained bool
	stopped bool
}

// NewTransport starts an embedded NATS server and connects the daemon to it
// using an in-process, token-authenticated connection.
func NewTransport(ctx context.Context, cfg aghconfig.NetworkConfig, opts ...TransportOption) (*Transport, error) {
	if ctx == nil {
		return nil, errors.New("network: transport context is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("network: validate transport config: %w", err)
	}

	options := resolveTransportOptions(opts...)

	token := rand.Text()
	serverOptions, err := newEmbeddedServerOptions(cfg, token)
	if err != nil {
		return nil, err
	}

	natsServer, err := server.NewServer(serverOptions)
	if err != nil {
		return nil, fmt.Errorf("network: create embedded nats server: %w", err)
	}
	natsServer.Start()
	if !natsServer.ReadyForConnections(options.readyTimeout) {
		natsServer.Shutdown()
		natsServer.WaitForShutdown()
		return nil, fmt.Errorf("network: embedded nats server did not become ready within %s", options.readyTimeout)
	}

	closedCh := make(chan struct{})
	closeOnce := sync.Once{}
	conn, err := nats.Connect(
		natsServer.ClientURL(),
		nats.InProcessServer(natsServer),
		nats.Token(token),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, disconnectErr error) {
			if options.onDisconnect != nil {
				options.onDisconnect(disconnectErr)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			if options.onReconnect != nil {
				options.onReconnect()
			}
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			closeOnce.Do(func() {
				close(closedCh)
			})
		}),
	)
	if err != nil {
		natsServer.Shutdown()
		natsServer.WaitForShutdown()
		return nil, fmt.Errorf("network: connect embedded nats client: %w", err)
	}

	transport := &Transport{
		logger:         options.logger,
		server:         natsServer,
		conn:           conn,
		token:          token,
		port:           resolvedTransportPort(natsServer),
		publishTimeout: options.publishTimeout,
		closedCh:       closedCh,
	}

	return transport, nil
}

func resolveTransportOptions(opts ...TransportOption) transportOptions {
	options := transportOptions{
		logger:         slog.Default(),
		readyTimeout:   defaultTransportReadyTimeout,
		publishTimeout: defaultTransportPublishTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.logger == nil {
		options.logger = slog.Default()
	}
	if options.readyTimeout <= 0 {
		options.readyTimeout = defaultTransportReadyTimeout
	}
	if options.publishTimeout <= 0 {
		options.publishTimeout = defaultTransportPublishTimeout
	}
	return options
}

func newEmbeddedServerOptions(cfg aghconfig.NetworkConfig, token string) (*server.Options, error) {
	maxPayload, err := natsMaxPayload(cfg.MaxPayload)
	if err != nil {
		return nil, err
	}

	return &server.Options{
		Authorization: token,
		Host:          "127.0.0.1",
		Port:          cfg.Port,
		NoSigs:        true,
		MaxPayload:    maxPayload,
	}, nil
}

func natsMaxPayload(maxPayload int) (int32, error) {
	if maxPayload < 0 {
		return 0, fmt.Errorf("%w: network max payload must be non-negative", ErrInvalidField)
	}
	if maxPayload > maxNATSPayloadBytes {
		return 0, fmt.Errorf(
			"%w: network max payload %d exceeds supported limit %d",
			ErrInvalidField,
			maxPayload,
			maxNATSPayloadBytes,
		)
	}
	return int32(maxPayload), nil
}

// Port reports the resolved listener port. Random-port servers return the
// actual chosen port after startup.
func (t *Transport) Port() int {
	if t == nil {
		return 0
	}
	return t.port
}

// ClientURL reports the internal NATS client URL used by the transport.
func (t *Transport) ClientURL() string {
	if t == nil || t.server == nil {
		return ""
	}
	return t.server.ClientURL()
}

// Publish sends one payload to a NATS subject and flushes the connection.
func (t *Transport) Publish(ctx context.Context, subject string, payload []byte) error {
	if ctx == nil {
		return errors.New("network: publish context is required")
	}
	if t == nil || t.conn == nil {
		return errors.New("network: transport connection is required")
	}

	trimmedSubject := strings.TrimSpace(subject)
	if trimmedSubject == "" {
		return errors.New("network: publish subject is required")
	}
	if err := t.conn.Publish(trimmedSubject, payload); err != nil {
		return fmt.Errorf("network: publish to subject %q: %w", trimmedSubject, err)
	}

	flushCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		flushCtx, cancel = context.WithTimeout(ctx, t.publishTimeout)
		defer cancel()
	}
	if err := t.conn.FlushWithContext(flushCtx); err != nil {
		return fmt.Errorf("network: flush publish to subject %q: %w", trimmedSubject, err)
	}

	return nil
}

// Subscribe registers a callback for one subject.
func (t *Transport) Subscribe(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	if t == nil || t.conn == nil {
		return nil, errors.New("network: transport connection is required")
	}
	trimmedSubject := strings.TrimSpace(subject)
	if trimmedSubject == "" {
		return nil, errors.New("network: subscription subject is required")
	}
	if handler == nil {
		return nil, errors.New("network: subscription handler is required")
	}

	subscription, err := t.conn.Subscribe(trimmedSubject, handler)
	if err != nil {
		return nil, fmt.Errorf("network: subscribe to subject %q: %w", trimmedSubject, err)
	}
	return subscription, nil
}

// Drain closes the daemon connection cleanly before the server is shut down.
func (t *Transport) Drain(ctx context.Context) error {
	if ctx == nil {
		return errors.New("network: drain context is required")
	}
	if t == nil || t.conn == nil {
		return errors.New("network: transport connection is required")
	}

	t.mu.Lock()
	if t.drained {
		t.mu.Unlock()
		return nil
	}
	t.drained = true
	t.mu.Unlock()

	if err := t.conn.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) &&
		!errors.Is(err, nats.ErrConnectionDraining) {
		return fmt.Errorf("network: drain embedded nats client: %w", err)
	}

	select {
	case <-t.closedCh:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("network: wait for drained connection: %w", ctx.Err())
	}
}

// Shutdown drains the daemon connection and stops the embedded server.
func (t *Transport) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("network: shutdown context is required")
	}
	if t == nil {
		return nil
	}

	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return nil
	}
	t.stopped = true
	server := t.server
	t.mu.Unlock()

	drainErr := t.Drain(ctx)

	shutdownErr := shutdownEmbeddedNATSServer(server)
	return errors.Join(drainErr, shutdownErr)
}

func shutdownEmbeddedNATSServer(natsServer *server.Server) error {
	if natsServer == nil {
		return nil
	}

	natsServer.Shutdown()
	done := make(chan struct{})
	go func() {
		defer close(done)
		natsServer.WaitForShutdown()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(defaultTransportShutdownTimeout):
		return fmt.Errorf(
			"network: embedded nats server did not shut down within %s",
			defaultTransportShutdownTimeout,
		)
	}
}

// BroadcastSubject builds the workspace-qualified broadcast subject for one channel.
func BroadcastSubject(workspaceID string, channel string) (string, error) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := ValidateWorkspaceID(trimmedWorkspaceID); err != nil {
		return "", err
	}
	trimmedChannel := strings.TrimSpace(channel)
	if err := ValidateChannel(trimmedChannel); err != nil {
		return "", err
	}
	return subjectPrefix + "." + trimmedWorkspaceID + "." + trimmedChannel + ".broadcast", nil
}

// DirectSubject builds the workspace-qualified direct subject for one target peer.
func DirectSubject(workspaceID string, channel string, peerID string) (string, error) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := ValidateWorkspaceID(trimmedWorkspaceID); err != nil {
		return "", err
	}
	trimmedChannel := strings.TrimSpace(channel)
	if err := ValidateChannel(trimmedChannel); err != nil {
		return "", err
	}
	routeToken, err := RouteToken(peerID)
	if err != nil {
		return "", err
	}
	return subjectPrefix + "." + trimmedWorkspaceID + "." + trimmedChannel + ".peer." + routeToken, nil
}

func resolvedTransportPort(natsServer *server.Server) int {
	if natsServer == nil {
		return 0
	}

	clientURL := strings.TrimSpace(natsServer.ClientURL())
	if clientURL == "" {
		return 0
	}

	parsed, err := url.Parse(clientURL)
	if err != nil {
		return 0
	}
	if port := strings.TrimSpace(parsed.Port()); port != "" {
		value, convErr := strconv.Atoi(port)
		if convErr == nil {
			return value
		}
	}
	return 0
}
