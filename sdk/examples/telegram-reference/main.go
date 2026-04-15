package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	adapterHandshakeEnv = "AGH_BRIDGE_ADAPTER_HANDSHAKE_PATH"
	adapterInstanceEnv  = "AGH_BRIDGE_ADAPTER_INSTANCE_PATH"
	adapterStateEnv     = "AGH_BRIDGE_ADAPTER_STATE_PATH"
	adapterDeliveryEnv  = "AGH_BRIDGE_ADAPTER_DELIVERY_PATH"
	adapterIngestEnv    = "AGH_BRIDGE_ADAPTER_INGEST_PATH"
	adapterUpdatesEnv   = "AGH_BRIDGE_ADAPTER_UPDATES_PATH"
	adapterStartsEnv    = "AGH_BRIDGE_ADAPTER_STARTS_PATH"
	adapterShutdownEnv  = "AGH_BRIDGE_ADAPTER_SHUTDOWN_PATH"
	adapterCrashOnceEnv = "AGH_BRIDGE_ADAPTER_CRASH_ONCE_PATH"

	rpcCodeNotInitialized = -32003
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "serve" {
		return runServe(stdin, stdout, stderr)
	}
	return fmt.Errorf("telegram-reference: unsupported command %q", strings.TrimSpace(args[0]))
}

func runServe(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	reportSideEffectError(stderr, "write start marker", appendMarkerLine(os.Getenv(adapterStartsEnv), fmt.Sprintf("pid=%d", os.Getpid())))

	peer := newRPCPeer(stdin, stdout)
	runtime := newTelegramReferenceRuntime(stderr, peer)

	peer.handle("initialize", runtime.handleInitialize)
	peer.handle("bridges/deliver", runtime.handleBridgesDeliver)
	peer.handle("health_check", runtime.handleHealthCheck)
	peer.handle("shutdown", runtime.handleShutdown)

	return peer.serve()
}

type adapterEnv struct {
	handshakePath string
	instancePath  string
	statePath     string
	deliveryPath  string
	ingestPath    string
	updatesPath   string
	startsPath    string
	shutdownPath  string
	crashOncePath string
}

func adapterEnvFromProcess() adapterEnv {
	return adapterEnv{
		handshakePath: strings.TrimSpace(os.Getenv(adapterHandshakeEnv)),
		instancePath:  strings.TrimSpace(os.Getenv(adapterInstanceEnv)),
		statePath:     strings.TrimSpace(os.Getenv(adapterStateEnv)),
		deliveryPath:  strings.TrimSpace(os.Getenv(adapterDeliveryEnv)),
		ingestPath:    strings.TrimSpace(os.Getenv(adapterIngestEnv)),
		updatesPath:   strings.TrimSpace(os.Getenv(adapterUpdatesEnv)),
		startsPath:    strings.TrimSpace(os.Getenv(adapterStartsEnv)),
		shutdownPath:  strings.TrimSpace(os.Getenv(adapterShutdownEnv)),
		crashOncePath: strings.TrimSpace(os.Getenv(adapterCrashOnceEnv)),
	}
}

type telegramReferenceRuntime struct {
	stderr io.Writer
	peer   *rpcPeer
	now    func() time.Time
	env    adapterEnv

	mu          sync.RWMutex
	initialized bool
	lastError   string
	session     runtimeSession
	deliveries  map[string]deliveryState

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type runtimeSession struct {
	request     subprocess.InitializeRequest
	response    subprocess.InitializeResponse
	bridge      subprocess.InitializeBridgeManagedInstance
	boundSecret map[string]subprocess.InitializeBridgeBoundSecret
}

type deliveryState struct {
	LastSeq         int64
	RemoteMessageID string
}

type initializeMarker struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
}

type deliveryMarker struct {
	PID     int                       `json:"pid"`
	Request bridgepkg.DeliveryRequest `json:"request"`
	Ack     *bridgepkg.DeliveryAck    `json:"ack,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

type stateMarker struct {
	Status   bridgepkg.BridgeStatus   `json:"status"`
	Instance bridgepkg.BridgeInstance `json:"instance,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

type ingestMarker struct {
	Envelope bridgepkg.InboundMessageEnvelope              `json:"envelope"`
	Result   extensioncontract.BridgesMessagesIngestResult `json:"result,omitempty"`
	Error    string                                        `json:"error,omitempty"`
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message,omitempty"`
}

type telegramMessage struct {
	MessageID       int64        `json:"message_id"`
	MessageThreadID int64        `json:"message_thread_id,omitempty"`
	Date            int64        `json:"date"`
	Chat            telegramChat `json:"chat"`
	From            telegramUser `json:"from"`
	Text            string       `json:"text,omitempty"`
	Caption         string       `json:"caption,omitempty"`
}

type telegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type rpcEnvelope struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      json.RawMessage  `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *rpcErrorPayload `json:"error,omitempty"`
}

type rpcErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type runtimeRPCError struct {
	Code    int
	Message string
}

func (e *runtimeRPCError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type rpcPeer struct {
	scanner *bufio.Scanner
	encoder *json.Encoder

	writeMu sync.Mutex
	pending sync.Map
	wg      sync.WaitGroup
	nextID  int64

	handlers map[string]func(json.RawMessage) (any, error)
}

func newRPCPeer(stdin io.Reader, stdout io.Writer) *rpcPeer {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)

	return &rpcPeer{
		scanner:  scanner,
		encoder:  encoder,
		handlers: make(map[string]func(json.RawMessage) (any, error)),
	}
}

func (p *rpcPeer) handle(method string, handler func(json.RawMessage) (any, error)) {
	p.handlers[strings.TrimSpace(method)] = handler
}

func (p *rpcPeer) serve() error {
	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())
		if line == "" {
			continue
		}

		var envelope rpcEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			return fmt.Errorf("telegram-reference: decode rpc frame: %w", err)
		}

		if strings.TrimSpace(envelope.Method) != "" {
			p.wg.Add(1)
			go func(env rpcEnvelope) {
				defer p.wg.Done()
				p.dispatchRequest(env)
			}(envelope)
			continue
		}

		idKey := string(bytesTrim(envelope.ID))
		if idKey == "" {
			continue
		}
		if pending, ok := p.pending.LoadAndDelete(idKey); ok {
			pending.(chan rpcEnvelope) <- envelope
		}
	}

	p.wg.Wait()
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("telegram-reference: read rpc frame: %w", err)
	}
	return nil
}

func (p *rpcPeer) dispatchRequest(envelope rpcEnvelope) {
	handler, ok := p.handlers[strings.TrimSpace(envelope.Method)]
	if !ok {
		_ = p.sendError(envelope.ID, rpcErrorPayload{Code: -32601, Message: "Method not found"})
		return
	}

	result, err := handler(envelope.Params)
	if err != nil {
		var rpcErr *runtimeRPCError
		if errors.As(err, &rpcErr) {
			_ = p.sendError(envelope.ID, rpcErrorPayload{Code: rpcErr.Code, Message: rpcErr.Message})
			return
		}
		_ = p.sendError(envelope.ID, rpcErrorPayload{Code: -32603, Message: err.Error()})
		return
	}

	_ = p.sendResult(envelope.ID, result)
}

func (p *rpcPeer) call(ctx context.Context, method string, params any, result any) error {
	idValue := fmt.Sprintf("telegram-reference-%d", atomic.AddInt64(&p.nextID, 1))
	idBytes, err := json.Marshal(idValue)
	if err != nil {
		return err
	}

	responseCh := make(chan rpcEnvelope, 1)
	p.pending.Store(string(idBytes), responseCh)
	if err := p.writeFrame(rpcEnvelope{
		JSONRPC: "2.0",
		ID:      idBytes,
		Method:  method,
		Params:  mustRawJSON(params),
	}); err != nil {
		p.pending.Delete(string(idBytes))
		return err
	}

	select {
	case <-ctx.Done():
		p.pending.Delete(string(idBytes))
		return ctx.Err()
	case response := <-responseCh:
		if response.Error != nil {
			return &runtimeRPCError{Code: response.Error.Code, Message: response.Error.Message}
		}
		if result == nil {
			return nil
		}
		payload, err := json.Marshal(response.Result)
		if err != nil {
			return err
		}
		if len(payload) == 0 || string(payload) == "null" {
			return nil
		}
		return json.Unmarshal(payload, result)
	}
}

func (p *rpcPeer) sendResult(id json.RawMessage, result any) error {
	return p.writeFrame(rpcEnvelope{
		JSONRPC: "2.0",
		ID:      append(json.RawMessage(nil), id...),
		Result:  result,
	})
}

func (p *rpcPeer) sendError(id json.RawMessage, rpcErr rpcErrorPayload) error {
	return p.writeFrame(rpcEnvelope{
		JSONRPC: "2.0",
		ID:      append(json.RawMessage(nil), id...),
		Error:   &rpcErr,
	})
}

func (p *rpcPeer) writeFrame(frame rpcEnvelope) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return p.encoder.Encode(frame)
}

func newTelegramReferenceRuntime(stderr io.Writer, peer *rpcPeer) *telegramReferenceRuntime {
	if stderr == nil {
		stderr = io.Discard
	}
	return &telegramReferenceRuntime{
		stderr:     stderr,
		peer:       peer,
		now:        func() time.Time { return time.Now().UTC() },
		env:        adapterEnvFromProcess(),
		deliveries: make(map[string]deliveryState),
		stopCh:     make(chan struct{}),
	}
}

func (r *telegramReferenceRuntime) handleInitialize(params json.RawMessage) (any, error) {
	var request subprocess.InitializeRequest
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, fmt.Errorf("telegram-reference: decode initialize request: %w", err)
	}
	if request.Runtime.Bridge == nil {
		return nil, errors.New("telegram-reference: initialize runtime bridge is required")
	}
	managedBridge, err := request.Runtime.Bridge.SingleManagedInstance()
	if err != nil {
		return nil, fmt.Errorf("telegram-reference: select managed bridge instance: %w", err)
	}

	response := subprocess.InitializeResponse{
		ProtocolVersion: request.ProtocolVersion,
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "telegram-reference",
			Version: "0.1.0",
		},
		AcceptedCapabilities: subprocess.AcceptedCapabilities{
			Provides: append([]string(nil), request.Capabilities.Provides...),
			Actions:  append([]extensionprotocol.HostAPIMethod(nil), request.Capabilities.GrantedActions...),
			Security: append([]string(nil), request.Capabilities.GrantedSecurity...),
		},
		ImplementedMethods: []string{
			"bridges/deliver",
			"health_check",
			"shutdown",
		},
		Supports: subprocess.InitializeSupports{
			HealthCheck: true,
		},
	}

	r.mu.Lock()
	r.initialized = true
	r.session = runtimeSession{
		request:     request,
		response:    response,
		bridge:      *managedBridge,
		boundSecret: indexBoundSecrets(managedBridge.BoundSecrets),
	}
	r.lastError = ""
	r.mu.Unlock()

	r.reportSideEffectError("write initialize marker", writeJSONFile(r.env.handshakePath, initializeMarker{
		Request:  request,
		Response: response,
	}))

	r.wg.Add(2)
	go func() {
		defer r.wg.Done()
		r.afterInitialize()
	}()
	go func() {
		defer r.wg.Done()
		r.pollInboundUpdates()
	}()

	return response, nil
}

func (r *telegramReferenceRuntime) afterInitialize() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	instance, err := r.hostInstance(ctx)
	if err != nil {
		r.setLastError(err)
		r.reportSideEffectError("write error state marker", appendJSONLine(r.env.statePath, stateMarker{
			Status: bridgepkg.BridgeStatusError,
			Error:  err.Error(),
		}))
		return
	}
	r.reportSideEffectError("write instance marker", writeJSONFile(r.env.instancePath, instance))

	status := bridgepkg.BridgeStatusReady
	if _, ok := boundSecretValue(r.sessionSnapshot().bridge, "bot_token"); !ok {
		status = bridgepkg.BridgeStatusAuthRequired
	}

	if _, err := r.reportState(ctx, status); err != nil {
		r.setLastError(err)
		r.reportSideEffectError("write failed state marker", appendJSONLine(r.env.statePath, stateMarker{
			Status: status,
			Error:  err.Error(),
		}))
		return
	}
	r.clearLastError()
}

func (r *telegramReferenceRuntime) handleBridgesDeliver(params json.RawMessage) (any, error) {
	var request bridgepkg.DeliveryRequest
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, fmt.Errorf("telegram-reference: decode delivery request: %w", err)
	}

	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	if shouldCrashOnce(r.env.crashOncePath) {
		r.reportSideEffectError("write pre-crash delivery marker", appendJSONLine(r.env.deliveryPath, marker))
		r.reportSideEffectError("write crash marker", writeJSONFile(r.env.crashOncePath, map[string]any{
			"crashed":     true,
			"pid":         os.Getpid(),
			"delivery_id": strings.TrimSpace(request.Event.DeliveryID),
		}))
		os.Exit(23)
	}

	ack, err := r.ackDelivery(request)
	if err != nil {
		r.setLastError(err)
		marker.Error = err.Error()
		r.reportSideEffectError("write failed delivery marker", appendJSONLine(r.env.deliveryPath, marker))
		return nil, err
	}
	marker.Ack = &ack
	r.reportSideEffectError("write delivery marker", appendJSONLine(r.env.deliveryPath, marker))
	return ack, nil
}

func (r *telegramReferenceRuntime) handleHealthCheck(json.RawMessage) (any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	message := strings.TrimSpace(r.lastError)
	return subprocess.HealthCheckResponse{
		Healthy: r.initialized && message == "",
		Message: message,
	}, nil
}

func (r *telegramReferenceRuntime) handleShutdown(params json.RawMessage) (any, error) {
	var request subprocess.ShutdownRequest
	if len(params) > 0 {
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, fmt.Errorf("telegram-reference: decode shutdown request: %w", err)
		}
	}

	r.mu.Lock()
	alreadyStopped := !r.initialized
	r.initialized = false
	r.mu.Unlock()

	if !alreadyStopped {
		close(r.stopCh)
	}

	deadline := time.Now().Add(5 * time.Second)
	if request.DeadlineMS > 0 {
		deadline = time.Now().Add(time.Duration(request.DeadlineMS) * time.Millisecond)
	}
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Until(deadline)):
	}

	r.reportSideEffectError("write shutdown marker", appendMarkerLine(r.env.shutdownPath, fmt.Sprintf("pid=%d", os.Getpid())))
	return subprocess.ShutdownResponse{Acknowledged: true}, nil
}

func (r *telegramReferenceRuntime) pollInboundUpdates() {
	updatesPath := strings.TrimSpace(r.env.updatesPath)
	if updatesPath == "" {
		return
	}

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	processed := 0
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			payload, err := os.ReadFile(updatesPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				r.setLastError(err)
				continue
			}

			lines := nonEmptyLines(string(payload))
			for processed < len(lines) {
				var update telegramUpdate
				if err := json.Unmarshal([]byte(lines[processed]), &update); err != nil {
					break
				}
				if err := r.ingestTelegramUpdate(update); err != nil {
					r.setLastError(err)
				} else {
					r.clearLastError()
				}
				processed++
			}
		}
	}
}

func (r *telegramReferenceRuntime) ingestTelegramUpdate(update telegramUpdate) error {
	session := r.sessionSnapshot()
	envelope, err := mapTelegramUpdate(update, session.bridge, r.now)
	if err != nil {
		r.reportSideEffectError("write failed ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
			Envelope: envelope,
			Error:    err.Error(),
		}))
		return err
	}

	var result extensioncontract.BridgesMessagesIngestResult
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = r.callHost(ctx, string(extensionprotocol.HostAPIMethodBridgesMessagesIngest), envelope, &result)
	if err != nil {
		r.reportSideEffectError("write failed ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
			Envelope: envelope,
			Error:    err.Error(),
		}))
		return err
	}

	r.reportSideEffectError("write ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
		Envelope: envelope,
		Result:   result,
	}))
	return nil
}

func (r *telegramReferenceRuntime) hostInstance(ctx context.Context) (bridgepkg.BridgeInstance, error) {
	var instance bridgepkg.BridgeInstance
	err := r.callHost(ctx, string(extensionprotocol.HostAPIMethodBridgesInstancesGet), map[string]any{}, &instance)
	return instance, err
}

func (r *telegramReferenceRuntime) reportState(
	ctx context.Context,
	status bridgepkg.BridgeStatus,
) (*bridgepkg.BridgeInstance, error) {
	var instance bridgepkg.BridgeInstance
	err := r.callHost(
		ctx,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		extensioncontract.BridgesInstancesReportStateParams{Status: status},
		&instance,
	)
	if err != nil {
		return nil, err
	}
	r.reportSideEffectError("write state marker", appendJSONLine(r.env.statePath, stateMarker{
		Status:   instance.Status,
		Instance: instance,
	}))
	return &instance, nil
}

func (r *telegramReferenceRuntime) callHost(ctx context.Context, method string, params any, result any) error {
	delay := 10 * time.Millisecond
	var lastErr error

	for attempt := 0; attempt < 6; attempt++ {
		err := r.peer.call(ctx, method, params, result)
		if err == nil {
			return nil
		}
		if !isNotInitializedRPCError(err) {
			return err
		}

		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.stopCh:
			return err
		case <-time.After(delay):
		}

		if delay < 100*time.Millisecond {
			delay *= 2
			if delay > 100*time.Millisecond {
				delay = 100 * time.Millisecond
			}
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return nil
}

func (r *telegramReferenceRuntime) ackDelivery(request bridgepkg.DeliveryRequest) (bridgepkg.DeliveryAck, error) {
	if err := request.Validate(); err != nil {
		return bridgepkg.DeliveryAck{}, err
	}

	event := request.Event
	deliveryID := strings.TrimSpace(event.DeliveryID)

	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.deliveries[deliveryID]
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		state.LastSeq = request.Snapshot.LastAckedSeq
		state.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
	}

	if normalizeDeliveryEventType(event.EventType) != bridgepkg.DeliveryEventTypeResume && event.Seq <= state.LastSeq {
		return bridgepkg.DeliveryAck{}, fmt.Errorf(
			"telegram-reference: out-of-order delivery seq %d after %d",
			event.Seq,
			state.LastSeq,
		)
	}

	remoteID := state.RemoteMessageID
	if normalizeDeliveryEventType(event.EventType) != bridgepkg.DeliveryEventTypeResume || remoteID == "" {
		remoteID = remoteMessageID(deliveryID, event.Seq)
	}

	ack := bridgepkg.DeliveryAck{
		DeliveryID:      deliveryID,
		Seq:             event.Seq,
		RemoteMessageID: remoteID,
	}
	if state.RemoteMessageID != "" && state.RemoteMessageID != remoteID {
		ack.ReplaceRemoteMessageID = state.RemoteMessageID
	}
	if normalizeDeliveryEventType(event.EventType) == bridgepkg.DeliveryEventTypeResume && request.Snapshot != nil {
		if ack.RemoteMessageID == "" {
			ack.RemoteMessageID = strings.TrimSpace(request.Snapshot.RemoteMessageID)
		}
		if ack.ReplaceRemoteMessageID == "" {
			ack.ReplaceRemoteMessageID = strings.TrimSpace(request.Snapshot.ReplaceRemoteMessageID)
		}
	}

	state.LastSeq = event.Seq
	if ack.RemoteMessageID != "" {
		state.RemoteMessageID = ack.RemoteMessageID
	}
	r.deliveries[deliveryID] = state

	return ack, nil
}

func (r *telegramReferenceRuntime) sessionSnapshot() runtimeSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.session
}

func (r *telegramReferenceRuntime) setLastError(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastError = err.Error()
}

func (r *telegramReferenceRuntime) clearLastError() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastError = ""
}

func (r *telegramReferenceRuntime) reportSideEffectError(action string, err error) {
	reportSideEffectError(r.stderr, action, err)
}

func mapTelegramUpdate(
	update telegramUpdate,
	bridgeRuntime subprocess.InitializeBridgeManagedInstance,
	now func() time.Time,
) (bridgepkg.InboundMessageEnvelope, error) {
	if update.Message == nil {
		return bridgepkg.InboundMessageEnvelope{}, errors.New("telegram-reference: message update is required")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	message := update.Message
	receivedAt := now().UTC()
	if message.Date > 0 {
		receivedAt = time.Unix(message.Date, 0).UTC()
	}

	text := strings.TrimSpace(message.Text)
	if text == "" {
		text = strings.TrimSpace(message.Caption)
	}

	senderName := strings.TrimSpace(strings.Join([]string{message.From.FirstName, message.From.LastName}, " "))
	return bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  bridgeRuntime.Instance.ID,
		Scope:             bridgeRuntime.Instance.Scope,
		WorkspaceID:       bridgeRuntime.Instance.WorkspaceID,
		PeerID:            strconv.FormatInt(message.Chat.ID, 10),
		ThreadID:          optionalTelegramID(message.MessageThreadID),
		PlatformMessageID: strconv.FormatInt(message.MessageID, 10),
		ReceivedAt:        receivedAt,
		Sender: bridgepkg.MessageSender{
			ID:          optionalTelegramID(message.From.ID),
			Username:    strings.TrimSpace(message.From.Username),
			DisplayName: senderName,
		},
		Content: bridgepkg.MessageContent{
			Text: text,
		},
		IdempotencyKey: fmt.Sprintf("telegram:%s:%d", bridgeRuntime.Instance.ID, update.UpdateID),
	}, nil
}

func boundSecretValue(bridgeRuntime subprocess.InitializeBridgeManagedInstance, bindingName string) (string, bool) {
	trimmed := strings.TrimSpace(bindingName)
	if trimmed == "" {
		return "", false
	}
	for _, secret := range bridgeRuntime.BoundSecrets {
		if strings.TrimSpace(secret.BindingName) == trimmed {
			return strings.TrimSpace(secret.Value), strings.TrimSpace(secret.Value) != ""
		}
	}
	return "", false
}

func indexBoundSecrets(secrets []subprocess.InitializeBridgeBoundSecret) map[string]subprocess.InitializeBridgeBoundSecret {
	if len(secrets) == 0 {
		return nil
	}
	indexed := make(map[string]subprocess.InitializeBridgeBoundSecret, len(secrets))
	for _, secret := range secrets {
		indexed[strings.TrimSpace(secret.BindingName)] = secret
	}
	return indexed
}

func remoteMessageID(deliveryID string, seq int64) string {
	return fmt.Sprintf("telegram:%s:%d", strings.TrimSpace(deliveryID), seq)
}

func optionalTelegramID(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func normalizeDeliveryEventType(eventType string) string {
	return strings.ToLower(strings.TrimSpace(eventType))
}

func isNotInitializedRPCError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *runtimeRPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	return rpcErr.Code == rpcCodeNotInitialized || strings.EqualFold(strings.TrimSpace(rpcErr.Message), "Not initialized")
}

func shouldCrashOnce(path string) bool {
	target := strings.TrimSpace(path)
	if target == "" {
		return false
	}
	_, err := os.Stat(target)
	return errors.Is(err, os.ErrNotExist)
}

func appendMarkerLine(path string, line string) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	_, err = fmt.Fprintf(file, "%s\n", strings.TrimSpace(line))
	return err
}

func appendJSONLine(path string, value any) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func writeJSONFile(path string, value any) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(target, append(payload, '\n'), 0o600)
}

func reportSideEffectError(stderr io.Writer, action string, err error) {
	if err == nil {
		return
	}
	writer := stderr
	if writer == nil {
		writer = io.Discard
	}
	_, _ = fmt.Fprintf(writer, "telegram-reference: %s: %v\n", strings.TrimSpace(action), err)
}

func mustRawJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func bytesTrim(value []byte) []byte {
	return []byte(strings.TrimSpace(string(value)))
}

func nonEmptyLines(input string) []string {
	lines := strings.Split(input, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}
