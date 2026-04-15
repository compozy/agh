package main

import (
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
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	adapterHandshakeEnv = "AGH_BRIDGE_ADAPTER_HANDSHAKE_PATH"
	adapterOwnershipEnv = "AGH_BRIDGE_ADAPTER_OWNERSHIP_PATH"
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

	runtime, err := newTelegramReferenceRuntime(stderr)
	if err != nil {
		return err
	}
	return runtime.serve(context.Background(), stdin, stdout)
}

type adapterEnv struct {
	handshakePath string
	ownershipPath string
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
		ownershipPath: strings.TrimSpace(os.Getenv(adapterOwnershipEnv)),
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
	sdk    *bridgesdk.Runtime
	stderr io.Writer
	now    func() time.Time
	env    adapterEnv

	mu         sync.RWMutex
	lastError  string
	deliveries map[string]deliveryState

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type deliveryState struct {
	LastSeq         int64
	RemoteMessageID string
}

type initializeMarker struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
}

type ownershipMarker struct {
	Listed  []bridgepkg.BridgeInstance `json:"listed,omitempty"`
	Fetched []bridgepkg.BridgeInstance `json:"fetched,omitempty"`
	Error   string                     `json:"error,omitempty"`
}

type deliveryMarker struct {
	PID     int                       `json:"pid"`
	Request bridgepkg.DeliveryRequest `json:"request"`
	Ack     *bridgepkg.DeliveryAck    `json:"ack,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

type stateMarker struct {
	BridgeInstanceID string                   `json:"bridge_instance_id,omitempty"`
	Status           bridgepkg.BridgeStatus   `json:"status"`
	Instance         bridgepkg.BridgeInstance `json:"instance,omitempty"`
	Error            string                   `json:"error,omitempty"`
}

type ingestMarker struct {
	Envelope bridgepkg.InboundMessageEnvelope              `json:"envelope"`
	Result   extensioncontract.BridgesMessagesIngestResult `json:"result,omitempty"`
	Error    string                                        `json:"error,omitempty"`
}

type telegramUpdate struct {
	BridgeInstanceID string           `json:"bridge_instance_id,omitempty"`
	UpdateID         int64            `json:"update_id"`
	Message          *telegramMessage `json:"message,omitempty"`
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

func newTelegramReferenceRuntime(stderr io.Writer) (*telegramReferenceRuntime, error) {
	if stderr == nil {
		stderr = io.Discard
	}

	runtime := &telegramReferenceRuntime{
		stderr:     stderr,
		now:        func() time.Time { return time.Now().UTC() },
		env:        adapterEnvFromProcess(),
		deliveries: make(map[string]deliveryState),
		stopCh:     make(chan struct{}),
	}

	sdkRuntime, err := bridgesdk.NewRuntime(bridgesdk.RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "telegram-reference",
			Version: "0.1.0",
			SDKName: "bridgesdk",
		},
		Initialize: runtime.handleInitialize,
		Deliver:    runtime.handleBridgesDeliver,
		HealthCheck: func(context.Context, *bridgesdk.Session) error {
			return runtime.healthCheck()
		},
		Shutdown: runtime.handleShutdown,
		Now: func() time.Time {
			return runtime.now()
		},
	})
	if err != nil {
		return nil, err
	}

	runtime.sdk = sdkRuntime
	return runtime, nil
}

func (r *telegramReferenceRuntime) serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return r.sdk.Serve(ctx, stdin, stdout)
}

func (r *telegramReferenceRuntime) handleInitialize(_ context.Context, session *bridgesdk.Session) error {
	marker := initializeMarker{
		Request:  session.InitializeRequest(),
		Response: session.InitializeResponse(),
	}
	r.reportSideEffectError("write initialize marker", writeJSONFile(r.env.handshakePath, marker))
	r.clearLastError()

	r.wg.Add(2)
	go func() {
		defer r.wg.Done()
		r.afterInitialize(session)
	}()
	go func() {
		defer r.wg.Done()
		r.pollInboundUpdates(session)
	}()

	return nil
}

func (r *telegramReferenceRuntime) afterInitialize(session *bridgesdk.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listed, err := r.syncOwnedInstances(ctx, session)
	fetched := make([]bridgepkg.BridgeInstance, 0)
	ownershipErr := err
	if ownershipErr == nil {
		for _, managed := range listed {
			instance, getErr := r.getOwnedInstance(ctx, session, managed.Instance.ID)
			if getErr != nil {
				ownershipErr = getErr
				break
			}
			fetched = append(fetched, *instance)
		}
	}
	if len(listed) == 0 {
		listed = session.Cache().List()
	}
	ownership := ownershipMarker{
		Listed:  managedInstancesToInstances(listed),
		Fetched: fetched,
	}
	if ownershipErr != nil {
		ownership.Error = ownershipErr.Error()
		r.setLastError(ownershipErr)
	}
	r.reportSideEffectError("write ownership marker", writeJSONFile(r.env.ownershipPath, ownership))

	var lastErr error
	for _, managed := range session.Cache().List() {
		status := bridgeStatusForManaged(session, managed.Instance.ID)
		if _, reportErr := r.reportState(ctx, session, managed.Instance.ID, status); reportErr != nil {
			lastErr = reportErr
		}
	}

	if lastErr != nil {
		r.setLastError(lastErr)
		return
	}
	if ownershipErr == nil {
		r.clearLastError()
	}
}

func (r *telegramReferenceRuntime) handleBridgesDeliver(
	_ context.Context,
	session *bridgesdk.Session,
	request bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	marker := deliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	}

	instanceID := strings.TrimSpace(request.Event.BridgeInstanceID)
	if _, ok := session.Cache().Get(instanceID); !ok {
		err := fmt.Errorf("telegram-reference: delivery targeted unmanaged instance %q", instanceID)
		marker.Error = err.Error()
		r.reportSideEffectError("write failed delivery marker", appendJSONLine(r.env.deliveryPath, marker))
		r.setLastError(err)
		return bridgepkg.DeliveryAck{}, err
	}

	if shouldCrashOnce(r.env.crashOncePath) {
		r.reportSideEffectError("write pre-crash delivery marker", appendJSONLine(r.env.deliveryPath, marker))
		r.reportSideEffectError("write crash marker", writeJSONFile(r.env.crashOncePath, map[string]any{
			"crashed":            true,
			"pid":                os.Getpid(),
			"delivery_id":        strings.TrimSpace(request.Event.DeliveryID),
			"bridge_instance_id": instanceID,
		}))
		os.Exit(23)
	}

	ack, err := r.ackDelivery(request)
	if err != nil {
		r.setLastError(err)
		marker.Error = err.Error()
		r.reportSideEffectError("write failed delivery marker", appendJSONLine(r.env.deliveryPath, marker))
		return bridgepkg.DeliveryAck{}, err
	}
	marker.Ack = &ack
	r.reportSideEffectError("write delivery marker", appendJSONLine(r.env.deliveryPath, marker))
	r.clearLastError()
	return ack, nil
}

func (r *telegramReferenceRuntime) healthCheck() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if strings.TrimSpace(r.lastError) == "" {
		return nil
	}
	return errors.New(strings.TrimSpace(r.lastError))
}

func (r *telegramReferenceRuntime) handleShutdown(
	_ context.Context,
	_ *bridgesdk.Session,
	request subprocess.ShutdownRequest,
) error {
	r.stop()

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
	return nil
}

func (r *telegramReferenceRuntime) stop() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

func (r *telegramReferenceRuntime) pollInboundUpdates(session *bridgesdk.Session) {
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
				if err := r.ingestTelegramUpdate(session, update); err != nil {
					r.setLastError(err)
				} else {
					r.clearLastError()
				}
				processed++
			}
		}
	}
}

func (r *telegramReferenceRuntime) ingestTelegramUpdate(session *bridgesdk.Session, update telegramUpdate) error {
	managed, err := resolveManagedInstance(session, update.BridgeInstanceID)
	if err != nil {
		r.reportSideEffectError("write failed ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
			Envelope: bridgepkg.InboundMessageEnvelope{BridgeInstanceID: strings.TrimSpace(update.BridgeInstanceID)},
			Error:    err.Error(),
		}))
		return err
	}

	envelope, err := mapTelegramUpdate(update, *managed, r.now)
	if err != nil {
		r.reportSideEffectError("write failed ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
			Envelope: bridgepkg.InboundMessageEnvelope{BridgeInstanceID: managed.Instance.ID},
			Error:    err.Error(),
		}))
		return err
	}

	result, err := r.ingestBridgeMessage(context.Background(), session, envelope)
	if err != nil {
		r.reportSideEffectError("write failed ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
			Envelope: envelope,
			Error:    err.Error(),
		}))
		return err
	}

	r.reportSideEffectError("write ingest marker", appendJSONLine(r.env.ingestPath, ingestMarker{
		Envelope: envelope,
		Result:   *result,
	}))
	return nil
}

func (r *telegramReferenceRuntime) syncOwnedInstances(
	ctx context.Context,
	session *bridgesdk.Session,
) ([]subprocess.InitializeBridgeManagedInstance, error) {
	var result []subprocess.InitializeBridgeManagedInstance
	err := r.retryHostCall(ctx, func(callCtx context.Context) error {
		items, callErr := session.SyncInstances(callCtx)
		if callErr == nil {
			result = items
		}
		return callErr
	})
	return result, err
}

func (r *telegramReferenceRuntime) getOwnedInstance(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
) (*bridgepkg.BridgeInstance, error) {
	var result *bridgepkg.BridgeInstance
	err := r.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().GetBridgeInstance(callCtx, bridgeInstanceID)
		if callErr == nil {
			result = instance
		}
		return callErr
	})
	return result, err
}

func (r *telegramReferenceRuntime) reportState(
	ctx context.Context,
	session *bridgesdk.Session,
	bridgeInstanceID string,
	status bridgepkg.BridgeStatus,
) (*bridgepkg.BridgeInstance, error) {
	var result *bridgepkg.BridgeInstance
	err := r.retryHostCall(ctx, func(callCtx context.Context) error {
		instance, callErr := session.HostAPI().ReportBridgeInstanceState(callCtx, extensioncontract.BridgesInstancesReportStateParams{
			BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
			Status:           status,
		})
		if callErr == nil {
			result = instance
		}
		return callErr
	})
	if err != nil {
		r.reportSideEffectError("write failed state marker", appendJSONLine(r.env.statePath, stateMarker{
			BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
			Status:           status,
			Error:            err.Error(),
		}))
		return nil, err
	}

	r.reportSideEffectError("write state marker", appendJSONLine(r.env.statePath, stateMarker{
		BridgeInstanceID: result.ID,
		Status:           result.Status,
		Instance:         *result,
	}))
	return result, nil
}

func (r *telegramReferenceRuntime) ingestBridgeMessage(
	ctx context.Context,
	session *bridgesdk.Session,
	envelope bridgepkg.InboundMessageEnvelope,
) (*extensioncontract.BridgesMessagesIngestResult, error) {
	var result *extensioncontract.BridgesMessagesIngestResult
	err := r.retryHostCall(ctx, func(callCtx context.Context) error {
		ingestResult, callErr := session.HostAPI().IngestBridgeMessage(callCtx, envelope)
		if callErr == nil {
			result = ingestResult
		}
		return callErr
	})
	return result, err
}

func (r *telegramReferenceRuntime) retryHostCall(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	delay := 10 * time.Millisecond
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if !isNotInitializedRPCError(err) {
			return err
		}

		lastErr = err
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-r.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return err
		case <-timer.C:
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
	instanceID := strings.TrimSpace(event.BridgeInstanceID)
	deliveryID := strings.TrimSpace(event.DeliveryID)
	key := deliveryStateKey(instanceID, deliveryID)

	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.deliveries[key]
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
		remoteID = remoteMessageID(instanceID, deliveryID, event.Seq)
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
	r.deliveries[key] = state

	return ack, nil
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

func resolveManagedInstance(
	session *bridgesdk.Session,
	bridgeInstanceID string,
) (*subprocess.InitializeBridgeManagedInstance, error) {
	if session == nil || session.Cache() == nil {
		return nil, errors.New("telegram-reference: managed bridge cache is required")
	}

	trimmedID := strings.TrimSpace(bridgeInstanceID)
	if trimmedID != "" {
		managed, ok := session.Cache().Get(trimmedID)
		if !ok || managed == nil {
			return nil, fmt.Errorf("telegram-reference: managed bridge instance %q is not owned by this runtime", trimmedID)
		}
		return managed, nil
	}

	managed := session.Cache().List()
	switch len(managed) {
	case 0:
		return nil, errors.New("telegram-reference: provider runtime does not own any bridge instances")
	case 1:
		only := managed[0]
		return &only, nil
	default:
		return nil, errors.New("telegram-reference: bridge_instance_id is required when provider owns multiple bridge instances")
	}
}

func bridgeStatusForManaged(session *bridgesdk.Session, bridgeInstanceID string) bridgepkg.BridgeStatus {
	if session == nil || session.Cache() == nil {
		return bridgepkg.BridgeStatusError
	}
	if _, ok := session.Cache().BoundSecretValue(bridgeInstanceID, "bot_token"); !ok {
		return bridgepkg.BridgeStatusAuthRequired
	}
	return bridgepkg.BridgeStatusReady
}

func managedInstancesToInstances(items []subprocess.InitializeBridgeManagedInstance) []bridgepkg.BridgeInstance {
	instances := make([]bridgepkg.BridgeInstance, 0, len(items))
	for _, item := range items {
		instances = append(instances, item.Instance)
	}
	return instances
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

func deliveryStateKey(instanceID string, deliveryID string) string {
	return strings.TrimSpace(instanceID) + ":" + strings.TrimSpace(deliveryID)
}

func remoteMessageID(instanceID string, deliveryID string, seq int64) string {
	return fmt.Sprintf("telegram:%s:%s:%d", strings.TrimSpace(instanceID), strings.TrimSpace(deliveryID), seq)
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
	var rpcErr *subprocess.RPCError
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
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	if _, err := fmt.Fprintln(file, strings.TrimSpace(line)); err != nil {
		return err
	}
	return nil
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
	return os.WriteFile(target, payload, 0o600)
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

func reportSideEffectError(writer io.Writer, action string, err error) {
	if err == nil || writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, "telegram-reference: %s: %v\n", strings.TrimSpace(action), err)
}
