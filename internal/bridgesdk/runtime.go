package bridgesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

// InitializeHandler runs after the provider runtime receives the negotiated
// initialize request and seeds its Host API client and managed-instance cache.
type InitializeHandler func(context.Context, *Session) error

// DeliveryHandler handles one daemon-originated `bridges/deliver` request.
type DeliveryHandler func(context.Context, *Session, bridgepkg.DeliveryRequest) (bridgepkg.DeliveryAck, error)

// HealthHandler handles one daemon health-check probe.
type HealthHandler func(context.Context, *Session) error

// ShutdownHandler handles one daemon-originated cooperative shutdown request.
type ShutdownHandler func(context.Context, *Session, subprocess.ShutdownRequest) error

// RuntimeConfig configures the shared provider runtime scaffold.
type RuntimeConfig struct {
	ExtensionInfo subprocess.InitializeExtensionInfo
	Initialize    InitializeHandler
	Deliver       DeliveryHandler
	HealthCheck   HealthHandler
	Shutdown      ShutdownHandler
	Now           func() time.Time
}

// Runtime is the shared provider runtime scaffold built on the bridge SDK.
type Runtime struct {
	config RuntimeConfig

	mu           sync.RWMutex
	peer         *Peer
	session      *Session
	initializing bool

	shutdownState runtimeShutdownState
}

type runtimeShutdownState uint8

const (
	runtimeShutdownIdle runtimeShutdownState = iota
	runtimeShutdownRunning
	runtimeShutdownSucceeded
)

// Session captures the negotiated provider runtime session state.
type Session struct {
	request  subprocess.InitializeRequest
	response subprocess.InitializeResponse
	host     *HostAPIClient
	cache    *InstanceCache
	now      func() time.Time
}

// NewRuntime constructs the shared provider runtime scaffold.
func NewRuntime(config RuntimeConfig) (*Runtime, error) {
	if strings.TrimSpace(config.ExtensionInfo.Name) == "" {
		return nil, errors.New("bridgesdk: runtime extension info name is required")
	}
	if strings.TrimSpace(config.ExtensionInfo.Version) == "" {
		return nil, errors.New("bridgesdk: runtime extension info version is required")
	}
	if config.Deliver == nil {
		return nil, errors.New("bridgesdk: runtime deliver handler is required")
	}
	if config.Now == nil {
		config.Now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &Runtime{config: config}, nil
}

// Serve runs the provider runtime over the supplied stdio transport.
func (r *Runtime) Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	if r == nil {
		return errors.New("bridgesdk: runtime is required")
	}
	if ctx == nil {
		return errors.New("bridgesdk: runtime context is required")
	}

	peer := NewPeer(stdin, stdout)
	if err := peer.Handle("initialize", r.handleInitialize); err != nil {
		return err
	}
	if err := peer.Handle("bridges/deliver", r.handleDeliver); err != nil {
		return err
	}
	if err := peer.Handle("health_check", r.handleHealthCheck); err != nil {
		return err
	}
	if err := peer.Handle("shutdown", r.handleShutdown); err != nil {
		return err
	}

	r.mu.Lock()
	r.peer = peer
	r.mu.Unlock()

	return peer.Serve(ctx)
}

// Session returns the negotiated runtime session once initialize succeeds.
func (r *Runtime) Session() *Session {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.session
}

// BridgeRuntime returns the current managed-instance runtime snapshot.
func (s *Session) BridgeRuntime() *subprocess.InitializeBridgeRuntime {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.Snapshot()
}

// InitializeRequest returns a clone of the negotiated initialize request.
func (s *Session) InitializeRequest() subprocess.InitializeRequest {
	if s == nil {
		return subprocess.InitializeRequest{}
	}

	return subprocess.CloneInitializeRequest(s.request)
}

// InitializeResponse returns a copy of the initialize response sent by the runtime.
func (s *Session) InitializeResponse() subprocess.InitializeResponse {
	if s == nil {
		return subprocess.InitializeResponse{}
	}

	response := s.response
	response.AcceptedCapabilities.Provides = append([]string(nil), response.AcceptedCapabilities.Provides...)
	response.AcceptedCapabilities.Actions = append(
		[]extensionprotocol.HostAPIMethod(nil),
		response.AcceptedCapabilities.Actions...,
	)
	response.AcceptedCapabilities.Security = append(
		[]string(nil),
		response.AcceptedCapabilities.Security...,
	)
	response.ImplementedMethods = append([]string(nil), response.ImplementedMethods...)
	response.SupportedHookEvents = append([]string(nil), response.SupportedHookEvents...)
	return response
}

// HostAPI returns the typed bridge Host API client.
func (s *Session) HostAPI() *HostAPIClient {
	if s == nil {
		return nil
	}
	return s.host
}

// Cache returns the provider-owned managed-instance cache.
func (s *Session) Cache() *InstanceCache {
	if s == nil {
		return nil
	}
	return s.cache
}

// SyncInstances refreshes the managed-instance cache from the Host API.
func (s *Session) SyncInstances(ctx context.Context) ([]subprocess.InitializeBridgeManagedInstance, error) {
	if s == nil || s.cache == nil {
		return nil, errors.New("bridgesdk: runtime session cache is required")
	}
	return s.cache.Sync(ctx, s.host)
}

// AckDelivery builds and validates one delivery acknowledgement for the request.
func (s *Session) AckDelivery(
	req bridgepkg.DeliveryRequest,
	remoteMessageID string,
	replaceRemoteMessageID string,
) (bridgepkg.DeliveryAck, error) {
	ack := bridgepkg.DeliveryAck{
		DeliveryID:             req.Event.DeliveryID,
		Seq:                    req.Event.Seq,
		RemoteMessageID:        strings.TrimSpace(remoteMessageID),
		ReplaceRemoteMessageID: strings.TrimSpace(replaceRemoteMessageID),
	}
	if err := ack.ValidateFor(req.Event); err != nil {
		return bridgepkg.DeliveryAck{}, err
	}
	return ack, nil
}

// ReportClassifiedError applies the recovery mapping for one provider failure
// and reports the resulting instance status transition through the Host API.
func (s *Session) ReportClassifiedError(
	ctx context.Context,
	bridgeInstanceID string,
	classified ClassifiedError,
) (*bridgepkg.BridgeInstance, RecoveryDecision, error) {
	if s == nil || s.host == nil {
		return nil, RecoveryDecision{}, errors.New("bridgesdk: runtime session host api is required")
	}

	recovery := classified.Recovery()
	if recovery.Status == "" {
		return nil, recovery, nil
	}

	updated, err := s.host.ReportBridgeInstanceState(ctx, extensioncontract.BridgesInstancesReportStateParams{
		BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
		Status:           recovery.Status,
		Degradation:      recovery.Degradation,
	})
	if err != nil {
		return nil, recovery, err
	}
	return updated, recovery, nil
}

func (r *Runtime) handleInitialize(ctx context.Context, raw json.RawMessage) (any, error) {
	if r == nil {
		return nil, errors.New("bridgesdk: runtime is required")
	}

	var request subprocess.InitializeRequest
	if err := decodeParams(raw, &request); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, subprocess.NewRPCError(bridgeSDKRPCCodeInvalidParams, "Invalid params", map[string]string{
			"error": err.Error(),
		})
	}
	if request.Runtime.Bridge == nil {
		return nil, subprocess.NewRPCError(bridgeSDKRPCCodeInvalidParams, "Invalid params", map[string]string{
			"error": "initialize bridge runtime is required",
		})
	}

	r.mu.Lock()
	if r.session != nil || r.initializing {
		r.mu.Unlock()
		return nil, subprocess.NewRPCError(bridgeSDKRPCCodeInternal, "Internal error", map[string]string{
			"error": "provider runtime already initialized",
		})
	}
	peer := r.peer
	r.initializing = true
	r.mu.Unlock()

	host := NewHostAPIClient(peer)
	cache := NewInstanceCache(request.Runtime.Bridge)
	response := r.initializeResponse(request)
	session := &Session{
		request:  request,
		response: response,
		host:     host,
		cache:    cache,
		now:      r.config.Now,
	}

	if r.config.Initialize != nil {
		if err := r.config.Initialize(ctx, session); err != nil {
			r.mu.Lock()
			r.initializing = false
			r.mu.Unlock()
			return nil, err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		r.initializing = false
		return nil, err
	}
	r.session = session
	r.initializing = false
	return response, nil
}

func (r *Runtime) handleDeliver(ctx context.Context, raw json.RawMessage) (any, error) {
	session, err := r.requireSession()
	if err != nil {
		return nil, err
	}

	var request bridgepkg.DeliveryRequest
	if err := decodeParams(raw, &request); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, subprocess.NewRPCError(bridgeSDKRPCCodeInvalidParams, "Invalid params", map[string]string{
			"error": err.Error(),
		})
	}

	ack, err := r.config.Deliver(ctx, session, request)
	if err != nil {
		return nil, err
	}
	if err := ack.ValidateFor(request.Event); err != nil {
		return nil, err
	}
	return ack, nil
}

func (r *Runtime) handleHealthCheck(ctx context.Context, _ json.RawMessage) (any, error) {
	session, err := r.requireSession()
	if err != nil {
		return nil, err
	}
	if r.config.HealthCheck != nil {
		if err := r.config.HealthCheck(ctx, session); err != nil {
			return nil, err
		}
	}
	return struct {
		OK bool `json:"ok"`
	}{OK: true}, nil
}

func (r *Runtime) handleShutdown(ctx context.Context, raw json.RawMessage) (any, error) {
	session, err := r.requireSession()
	if err != nil {
		return nil, err
	}

	var request subprocess.ShutdownRequest
	trimmedRaw := bytes.TrimSpace(raw)
	if len(trimmedRaw) > 0 && !bytes.Equal(trimmedRaw, []byte("null")) {
		if err := decodeParams(raw, &request); err != nil {
			return nil, err
		}
	}

	shouldRun, err := r.beginShutdown()
	if err != nil {
		return nil, err
	}
	if shouldRun {
		var shutdownErr error
		if r.config.Shutdown != nil {
			shutdownErr = r.config.Shutdown(ctx, session, request)
		}
		r.completeShutdown(shutdownErr)
		if shutdownErr != nil {
			return nil, shutdownErr
		}
	}

	return subprocess.ShutdownResponse{Acknowledged: true}, nil
}

func (r *Runtime) beginShutdown() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.shutdownState {
	case runtimeShutdownSucceeded:
		return false, nil
	case runtimeShutdownRunning:
		return false, subprocess.NewRPCError(bridgeSDKRPCCodeShutdownRunning, "Shutdown running", nil)
	default:
		r.shutdownState = runtimeShutdownRunning
		return true, nil
	}
}

func (r *Runtime) completeShutdown(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.shutdownState = runtimeShutdownIdle
		return
	}
	r.shutdownState = runtimeShutdownSucceeded
}

func (r *Runtime) requireSession() (*Session, error) {
	if r == nil {
		return nil, errors.New("bridgesdk: runtime is required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.session == nil {
		return nil, subprocess.NewRPCError(bridgeSDKRPCCodeNotInitialized, "Not initialized", nil)
	}
	return r.session, nil
}

func (r *Runtime) initializeResponse(request subprocess.InitializeRequest) subprocess.InitializeResponse {
	implemented := []string{
		string(extensionprotocol.ExtensionServiceMethodBridgesDeliver),
		"health_check",
		"shutdown",
	}
	slices.Sort(implemented)

	return subprocess.InitializeResponse{
		ProtocolVersion: request.ProtocolVersion,
		ExtensionInfo:   r.config.ExtensionInfo,
		AcceptedCapabilities: subprocess.AcceptedCapabilities{
			Provides: append([]string(nil), request.Capabilities.Provides...),
			Actions:  append([]extensionprotocol.HostAPIMethod(nil), request.Capabilities.GrantedActions...),
			Security: append([]string(nil), request.Capabilities.GrantedSecurity...),
		},
		ImplementedMethods: implemented,
		Supports: subprocess.InitializeSupports{
			HealthCheck: true,
		},
	}
}

func decodeParams(raw json.RawMessage, dest any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		raw = json.RawMessage("{}")
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return subprocess.NewRPCError(bridgeSDKRPCCodeInvalidParams, "Invalid params", map[string]string{
			"error": err.Error(),
		})
	}
	return nil
}
