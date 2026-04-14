package bridges

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type deliveryQueueKind string

const (
	deliveryQueueKindStart    deliveryQueueKind = "start"
	deliveryQueueKindDelta    deliveryQueueKind = "delta"
	deliveryQueueKindTerminal deliveryQueueKind = "terminal"
	deliveryQueueKindResume   deliveryQueueKind = "resume"
)

type deliveryQueueItem struct {
	deliveryID string
	kind       deliveryQueueKind
}

type routeWorker struct {
	hash             string
	bridgeInstanceID string
	extensionName    string
	queue            []deliveryQueueItem
	wakeCh           chan struct{}
}

type activeDelivery struct {
	deliveryID       string
	sessionID        string
	turnID           string
	bridgeInstanceID string
	extensionName    string
	routingKey       RoutingKey
	target           DeliveryTarget
	routeHash        string

	latestSeq       int64
	lastSentSeq     int64
	lastAckedSeq    int64
	latestEventType string
	currentContent  MessageContent
	final           bool
	errorText       string
	updatedAt       time.Time

	remoteMessageID        string
	replaceRemoteMessageID string
	startDelivered         bool

	pendingStart    *DeliveryEvent
	pendingDelta    *DeliveryEvent
	pendingTerminal *DeliveryEvent
	queuedStart     bool
	queuedDelta     bool
	queuedTerminal  bool
	queuedResume    bool

	seen map[string]struct{}
}

type instanceDeliveryMetrics struct {
	droppedByReason       map[string]int
	deliveryFailuresTotal int
	lastError             string
	lastErrorAt           time.Time
	lastSuccessAt         time.Time
}

// Broker projects session output into ordered delivery requests for one
// bridge-capable extension runtime.
type Broker struct {
	mu sync.Mutex

	transport DeliveryTransport

	now            func() time.Time
	queueCapacity  int
	retryDelay     time.Duration
	requestTimeout time.Duration
	lifecycleCtx   context.Context
	cancel         context.CancelFunc

	wg sync.WaitGroup

	deliveries   map[string]*activeDelivery
	turnIndex    map[string]string
	sessionIndex map[string]map[string]struct{}
	routes       map[string]*routeWorker
	metrics      map[string]*instanceDeliveryMetrics
}

var _ DeliveryBroker = (*Broker)(nil)

// NewBroker constructs a delivery broker with bounded per-route queues and
// background workers for negotiated extension delivery.
func NewBroker(transport DeliveryTransport, opts ...DeliveryBrokerOption) *Broker {
	broker := &Broker{
		transport:      transport,
		now:            func() time.Time { return time.Now().UTC() },
		queueCapacity:  defaultDeliveryQueueCapacity,
		retryDelay:     defaultDeliveryRetryDelay,
		requestTimeout: defaultDeliveryRequestTimeout,
		lifecycleCtx:   context.Background(),
		deliveries:     make(map[string]*activeDelivery),
		turnIndex:      make(map[string]string),
		sessionIndex:   make(map[string]map[string]struct{}),
		routes:         make(map[string]*routeWorker),
		metrics:        make(map[string]*instanceDeliveryMetrics),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(broker)
		}
	}
	if broker.now == nil {
		broker.now = func() time.Time { return time.Now().UTC() }
	}
	if broker.queueCapacity < 2 {
		broker.queueCapacity = 2
	}
	if broker.retryDelay <= 0 {
		broker.retryDelay = defaultDeliveryRetryDelay
	}
	if broker.requestTimeout <= 0 {
		broker.requestTimeout = defaultDeliveryRequestTimeout
	}
	baseCtx := broker.lifecycleCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	broker.lifecycleCtx, broker.cancel = context.WithCancel(baseCtx)
	return broker
}

// SetTransport swaps the negotiated extension-delivery transport used by the broker.
func (b *Broker) SetTransport(transport DeliveryTransport) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.transport = transport
	routes := make([]*routeWorker, 0, len(b.routes))
	for _, route := range b.routes {
		routes = append(routes, route)
	}
	b.mu.Unlock()
	for _, route := range routes {
		b.signalRoute(route)
	}
}

// Close stops every background route worker.
func (b *Broker) Close() {
	if b == nil {
		return
	}
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
}

// DeliveryMetrics returns a point-in-time snapshot of per-instance broker
// telemetry used by health and observability surfaces.
func (b *Broker) DeliveryMetrics() map[string]BridgeDeliveryMetrics {
	if b == nil {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	snapshot := make(map[string]BridgeDeliveryMetrics, len(b.metrics))
	for bridgeInstanceID, metrics := range b.metrics {
		if metrics == nil {
			continue
		}

		clonedReasons := make(map[string]int, len(metrics.droppedByReason))
		totalDropped := 0
		for reason, count := range metrics.droppedByReason {
			clonedReasons[reason] = count
			totalDropped += count
		}

		snapshot[bridgeInstanceID] = BridgeDeliveryMetrics{
			BridgeInstanceID:        bridgeInstanceID,
			DeliveryDroppedTotal:    totalDropped,
			DeliveryDroppedByReason: clonedReasons,
			DeliveryFailuresTotal:   metrics.deliveryFailuresTotal,
			LastError:               metrics.lastError,
			LastErrorAt:             metrics.lastErrorAt,
			LastSuccessAt:           metrics.lastSuccessAt,
		}
	}

	for _, route := range b.routes {
		if route == nil || route.bridgeInstanceID == "" {
			continue
		}
		entry := snapshot[route.bridgeInstanceID]
		entry.BridgeInstanceID = route.bridgeInstanceID
		entry.DeliveryBacklog += len(route.queue)
		snapshot[route.bridgeInstanceID] = entry
	}

	return snapshot
}

// RegisterPromptDelivery binds one prompted session turn to a live delivery
// projection and optionally seeds the broker from already-persisted turn events.
func (b *Broker) RegisterPromptDelivery(ctx context.Context, reg PromptDeliveryRegistration) (*DeliverySnapshot, error) {
	if b == nil {
		return nil, errors.New("bridges: delivery broker is required")
	}
	if ctx == nil {
		return nil, errors.New("bridges: delivery registration context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	normalized := reg.normalize()
	if err := normalized.Validate(); err != nil {
		return nil, err
	}

	routeHash, err := normalized.RoutingKey.Hash()
	if err != nil {
		return nil, fmt.Errorf("bridges: hash delivery routing key: %w", err)
	}

	b.mu.Lock()
	deliveryKey := turnLookupKey(normalized.SessionID, normalized.TurnID)
	if existingID, ok := b.turnIndex[deliveryKey]; ok {
		existing := b.deliveries[existingID]
		b.mu.Unlock()
		if existing == nil {
			return nil, ErrDeliveryNotFound
		}
		return b.Snapshot(ctx, existingID)
	}

	deliveryID := normalized.DeliveryID
	if deliveryID == "" {
		deliveryID = newDeliveryID()
	}
	now := b.now()
	delivery := &activeDelivery{
		deliveryID:       deliveryID,
		sessionID:        normalized.SessionID,
		turnID:           normalized.TurnID,
		bridgeInstanceID: normalized.RoutingKey.BridgeInstanceID,
		extensionName:    normalized.ExtensionName,
		routingKey:       normalized.RoutingKey,
		target:           normalized.DeliveryTarget,
		routeHash:        routeHash,
		updatedAt:        now,
		seen:             make(map[string]struct{}),
	}
	b.deliveries[deliveryID] = delivery
	b.turnIndex[deliveryKey] = deliveryID
	if _, ok := b.sessionIndex[normalized.SessionID]; !ok {
		b.sessionIndex[normalized.SessionID] = make(map[string]struct{})
	}
	b.sessionIndex[normalized.SessionID][deliveryID] = struct{}{}
	b.ensureRouteLocked(routeHash, normalized.RoutingKey.BridgeInstanceID, normalized.ExtensionName)
	b.mu.Unlock()

	for _, event := range normalized.SeedEvents {
		if err := b.ProjectEvent(ctx, normalized.SessionID, event); err != nil {
			return nil, err
		}
	}

	return b.Snapshot(ctx, deliveryID)
}

// Deliver enqueues one already-projected delivery event for ordered extension delivery.
func (b *Broker) Deliver(ctx context.Context, evt DeliveryEvent) error {
	if b == nil {
		return errors.New("bridges: delivery broker is required")
	}
	if ctx == nil {
		return errors.New("bridges: delivery context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	normalized := evt.normalize()
	if err := normalized.Validate(); err != nil {
		return err
	}

	routeHash, err := normalized.RoutingKey.Hash()
	if err != nil {
		return fmt.Errorf("bridges: hash delivery routing key: %w", err)
	}

	b.mu.Lock()
	delivery, ok := b.deliveries[normalized.DeliveryID]
	if !ok {
		b.mu.Unlock()
		return ErrDeliveryNotFound
	}
	if delivery.routeHash != routeHash {
		b.mu.Unlock()
		return errors.New("bridges: delivery event routing key does not match registered delivery")
	}
	route := b.ensureRouteLocked(routeHash, normalized.BridgeInstanceID, delivery.extensionName)
	err = b.enqueueEventLocked(route, delivery, normalized)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	b.applyQueuedEventLocked(delivery, normalized)
	b.mu.Unlock()

	b.signalRoute(route)
	return nil
}

// Snapshot returns the current resumable state for one active delivery.
func (b *Broker) Snapshot(ctx context.Context, deliveryID string) (*DeliverySnapshot, error) {
	if b == nil {
		return nil, errors.New("bridges: delivery broker is required")
	}
	if ctx == nil {
		return nil, errors.New("bridges: delivery snapshot context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(deliveryID)
	if trimmed == "" {
		return nil, errors.New("bridges: delivery snapshot id is required")
	}

	b.mu.Lock()
	delivery := b.deliveries[trimmed]
	if delivery == nil {
		b.mu.Unlock()
		return nil, ErrDeliveryNotFound
	}
	snapshot := cloneDeliverySnapshot(b.snapshotLocked(delivery))
	b.mu.Unlock()
	return &snapshot, nil
}

// ProjectEvent converts one live or persisted session output event into the
// delivery-oriented stream for the registered prompt turn.
func (b *Broker) ProjectEvent(ctx context.Context, sessionID string, event DeliveryProjectionEvent) error {
	if b == nil {
		return errors.New("bridges: delivery broker is required")
	}
	if ctx == nil {
		return errors.New("bridges: delivery projection context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	sessionID = strings.TrimSpace(sessionID)
	turnID := strings.TrimSpace(event.TurnID)
	if sessionID == "" || turnID == "" {
		return nil
	}

	b.mu.Lock()
	deliveryID, ok := b.turnIndex[turnLookupKey(sessionID, turnID)]
	if !ok {
		b.mu.Unlock()
		return nil
	}
	delivery := b.deliveries[deliveryID]
	if delivery == nil {
		b.mu.Unlock()
		return nil
	}

	fingerprint := agentEventFingerprint(event)
	if fingerprint != "" {
		if _, seen := delivery.seen[fingerprint]; seen {
			b.mu.Unlock()
			return nil
		}
	}

	projected, ok, err := b.projectEventLocked(delivery, event)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if !ok {
		b.mu.Unlock()
		return nil
	}

	route := b.ensureRouteLocked(delivery.routeHash, delivery.bridgeInstanceID, delivery.extensionName)
	err = b.enqueueEventLocked(route, delivery, projected)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if fingerprint != "" {
		delivery.seen[fingerprint] = struct{}{}
	}
	b.applyQueuedEventLocked(delivery, projected)
	b.mu.Unlock()

	b.signalRoute(route)
	return nil
}

// FailSession marks every unfinished delivery for the stopped session as a
// terminal error so adapters do not silently orphan bridge responses.
func (b *Broker) FailSession(ctx context.Context, sessionID string, reason string) error {
	if b == nil {
		return errors.New("bridges: delivery broker is required")
	}
	if ctx == nil {
		return errors.New("bridges: delivery fail context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "session stopped before delivery completed"
	}

	type pendingSignal struct {
		route *routeWorker
	}
	signals := make([]pendingSignal, 0)

	b.mu.Lock()
	deliveryIDs := b.sessionDeliveriesLocked(sessionID)
	for _, deliveryID := range deliveryIDs {
		delivery := b.deliveries[deliveryID]
		if delivery == nil || delivery.final {
			continue
		}

		projected := DeliveryEvent{
			DeliveryID:       delivery.deliveryID,
			BridgeInstanceID: delivery.bridgeInstanceID,
			RoutingKey:       delivery.routingKey,
			DeliveryTarget:   delivery.target,
			Seq:              delivery.latestSeq + 1,
			EventType:        DeliveryEventTypeError,
			Content:          delivery.currentContent,
			Final:            true,
			Metadata: deliveryMetadataJSON(map[string]string{
				"error": reason,
			}),
		}
		route := b.ensureRouteLocked(delivery.routeHash, delivery.bridgeInstanceID, delivery.extensionName)
		if err := b.enqueueEventLocked(route, delivery, projected); err != nil {
			b.mu.Unlock()
			return err
		}
		b.applyQueuedEventLocked(delivery, projected)
		signals = append(signals, pendingSignal{route: route})
	}
	b.mu.Unlock()

	for _, signal := range signals {
		b.signalRoute(signal.route)
	}
	return nil
}

func (b *Broker) ensureRouteLocked(hash string, bridgeInstanceID string, extensionName string) *routeWorker {
	if route := b.routes[hash]; route != nil {
		return route
	}

	route := &routeWorker{
		hash:             hash,
		bridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
		extensionName:    strings.TrimSpace(extensionName),
		wakeCh:           make(chan struct{}, 1),
	}
	b.routes[hash] = route

	b.wg.Add(1)
	go b.runRouteWorker(route)
	return route
}

func (b *Broker) runRouteWorker(route *routeWorker) {
	defer b.wg.Done()

	for {
		item, ok := b.popQueueItem(route)
		if !ok {
			select {
			case <-route.wakeCh:
				continue
			case <-b.lifecycleCtx.Done():
				return
			}
		}

		retry := b.processQueueItem(route, item)
		if retry {
			timer := time.NewTimer(b.retryDelay)
			select {
			case <-timer.C:
			case <-b.lifecycleCtx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			}
		}
	}
}

func (b *Broker) processQueueItem(route *routeWorker, item deliveryQueueItem) bool {
	req, eventType, eventSeq, deliveryID, ok := b.prepareRequest(route, item)
	if !ok {
		return false
	}

	transport := b.currentTransport()
	if transport == nil {
		b.handleSendFailure(route, deliveryID, ErrDeliveryTransportUnavailable)
		return true
	}

	callCtx, cancel := context.WithTimeout(b.lifecycleCtx, b.requestTimeout)
	ack, err := transport.DeliverBridge(callCtx, route.extensionName, req)
	cancel()
	if err != nil {
		b.handleSendFailure(route, deliveryID, err)
		return true
	}
	if err := ack.ValidateFor(req.Event); err != nil {
		b.handleSendFailure(route, deliveryID, err)
		return true
	}

	b.handleSendSuccess(route, deliveryID, eventType, eventSeq, ack)
	return false
}

func (b *Broker) prepareRequest(route *routeWorker, item deliveryQueueItem) (DeliveryRequest, string, int64, string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delivery := b.deliveries[item.deliveryID]
	if delivery == nil {
		return DeliveryRequest{}, "", 0, "", false
	}

	switch item.kind {
	case deliveryQueueKindStart:
		if delivery.pendingStart == nil {
			return DeliveryRequest{}, "", 0, "", false
		}
		event := cloneDeliveryEvent(*delivery.pendingStart)
		delivery.pendingStart = nil
		delivery.queuedStart = false
		if event.Seq > delivery.lastSentSeq {
			delivery.lastSentSeq = event.Seq
		}
		delivery.updatedAt = b.now()
		return DeliveryRequest{Event: event}, event.EventType, event.Seq, delivery.deliveryID, true
	case deliveryQueueKindDelta:
		if delivery.pendingDelta == nil {
			return DeliveryRequest{}, "", 0, "", false
		}
		event := cloneDeliveryEvent(*delivery.pendingDelta)
		delivery.pendingDelta = nil
		delivery.queuedDelta = false
		if event.Seq > delivery.lastSentSeq {
			delivery.lastSentSeq = event.Seq
		}
		delivery.updatedAt = b.now()
		return DeliveryRequest{Event: event}, event.EventType, event.Seq, delivery.deliveryID, true
	case deliveryQueueKindTerminal:
		if delivery.pendingTerminal == nil {
			return DeliveryRequest{}, "", 0, "", false
		}
		event := cloneDeliveryEvent(*delivery.pendingTerminal)
		delivery.pendingTerminal = nil
		delivery.queuedTerminal = false
		if event.Seq > delivery.lastSentSeq {
			delivery.lastSentSeq = event.Seq
		}
		delivery.updatedAt = b.now()
		return DeliveryRequest{Event: event}, event.EventType, event.Seq, delivery.deliveryID, true
	case deliveryQueueKindResume:
		delivery.queuedResume = false
		snapshot := cloneDeliverySnapshot(b.snapshotLocked(delivery))
		event := DeliveryEvent{
			DeliveryID:       delivery.deliveryID,
			BridgeInstanceID: delivery.bridgeInstanceID,
			RoutingKey:       delivery.routingKey,
			DeliveryTarget:   delivery.target,
			Seq:              delivery.latestSeq,
			EventType:        DeliveryEventTypeResume,
			Content:          delivery.currentContent,
			Final:            delivery.final,
			Metadata: deliveryMetadataJSON(map[string]string{
				"latest_event_type": delivery.latestEventType,
			}),
		}
		if event.Seq > delivery.lastSentSeq {
			delivery.lastSentSeq = event.Seq
		}
		delivery.updatedAt = b.now()
		return DeliveryRequest{
			Event:    event,
			Snapshot: &snapshot,
		}, event.EventType, event.Seq, delivery.deliveryID, true
	default:
		return DeliveryRequest{}, "", 0, "", false
	}
}

func (b *Broker) handleSendFailure(route *routeWorker, deliveryID string, reason error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delivery := b.deliveries[deliveryID]
	if delivery == nil {
		return
	}
	b.removeQueuedSlotLocked(route, deliveryID, deliveryQueueKindStart)
	b.removeQueuedSlotLocked(route, deliveryID, deliveryQueueKindDelta)
	b.removeQueuedSlotLocked(route, deliveryID, deliveryQueueKindTerminal)
	delivery.pendingStart = nil
	delivery.pendingDelta = nil
	delivery.pendingTerminal = nil

	if !delivery.queuedResume {
		route.queue = append([]deliveryQueueItem{{
			deliveryID: deliveryID,
			kind:       deliveryQueueKindResume,
		}}, route.queue...)
		delivery.queuedResume = true
	}
	delivery.updatedAt = b.now()
	if reason != nil {
		// Preserve terminal delivery errors as the operator-visible failure
		// signal; resume retries may still fail, but they should not replace a
		// concrete delivery error with a generic transport issue.
		if delivery.latestEventType == DeliveryEventTypeError && strings.TrimSpace(delivery.errorText) != "" {
			return
		}
		b.recordDeliveryIssueLocked(delivery.bridgeInstanceID, reason.Error())
	}
}

func (b *Broker) handleSendSuccess(route *routeWorker, deliveryID string, eventType string, eventSeq int64, ack DeliveryAck) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delivery := b.deliveries[deliveryID]
	if delivery == nil {
		return
	}

	if eventSeq > delivery.lastSentSeq {
		delivery.lastSentSeq = eventSeq
	}
	if eventSeq > delivery.lastAckedSeq {
		delivery.lastAckedSeq = eventSeq
	}
	if ack.RemoteMessageID != "" {
		delivery.remoteMessageID = strings.TrimSpace(ack.RemoteMessageID)
	}
	if ack.ReplaceRemoteMessageID != "" {
		delivery.replaceRemoteMessageID = strings.TrimSpace(ack.ReplaceRemoteMessageID)
	}
	if normalizeDeliveryEventType(eventType) == DeliveryEventTypeStart || normalizeDeliveryEventType(eventType) == DeliveryEventTypeResume {
		if delivery.latestSeq > 0 && delivery.latestEventType != DeliveryEventTypeError {
			delivery.startDelivered = true
		}
	}
	delivery.updatedAt = b.now()
	b.recordDeliverySuccessLocked(delivery.bridgeInstanceID, delivery.updatedAt)

	if delivery.final && !delivery.hasQueuedItems() {
		b.removeDeliveryLocked(route, delivery)
	}
}

func (b *Broker) popQueueItem(route *routeWorker) (deliveryQueueItem, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	current := b.routes[route.hash]
	if current == nil || len(current.queue) == 0 {
		return deliveryQueueItem{}, false
	}

	item := current.queue[0]
	current.queue = current.queue[1:]
	return item, true
}

func (b *Broker) currentTransport() DeliveryTransport {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.transport
}

func (b *Broker) enqueueEventLocked(route *routeWorker, delivery *activeDelivery, event DeliveryEvent) error {
	switch normalizeDeliveryEventType(event.EventType) {
	case DeliveryEventTypeStart:
		if delivery.queuedStart {
			start := cloneDeliveryEvent(*delivery.pendingStart)
			start.Content = event.Content
			start.Seq = event.Seq
			start.Metadata = cloneRawJSON(event.Metadata)
			delivery.pendingStart = &start
			return nil
		}
		if len(route.queue) >= b.queueCapacity && !b.dropQueuedDeltaLocked(route) {
			b.recordDeliveryDropLocked(route.bridgeInstanceID, "queue_saturated")
			return ErrDeliveryQueueSaturated
		}
		cloned := cloneDeliveryEvent(event)
		delivery.pendingStart = &cloned
		delivery.queuedStart = true
		route.queue = append(route.queue, deliveryQueueItem{deliveryID: delivery.deliveryID, kind: deliveryQueueKindStart})
		return nil
	case DeliveryEventTypeDelta:
		if !delivery.startDelivered && delivery.queuedStart && delivery.pendingStart != nil {
			if len(route.queue) >= b.queueCapacity && !b.dropQueuedDeltaLocked(route) {
				start := cloneDeliveryEvent(event)
				start.EventType = DeliveryEventTypeStart
				start.Final = false
				delivery.pendingStart = &start
				return nil
			}
		}
		if delivery.queuedDelta {
			cloned := cloneDeliveryEvent(event)
			delivery.pendingDelta = &cloned
			return nil
		}
		if len(route.queue) >= b.queueCapacity && !b.dropQueuedDeltaLocked(route) {
			b.recordDeliveryDropLocked(route.bridgeInstanceID, "queue_saturated")
			return ErrDeliveryQueueSaturated
		}
		cloned := cloneDeliveryEvent(event)
		delivery.pendingDelta = &cloned
		delivery.queuedDelta = true
		route.queue = append(route.queue, deliveryQueueItem{deliveryID: delivery.deliveryID, kind: deliveryQueueKindDelta})
		return nil
	case DeliveryEventTypeFinal, DeliveryEventTypeError:
		b.removeQueuedSlotLocked(route, delivery.deliveryID, deliveryQueueKindDelta)
		delivery.pendingDelta = nil
		if delivery.queuedTerminal {
			cloned := cloneDeliveryEvent(event)
			delivery.pendingTerminal = &cloned
			return nil
		}
		if len(route.queue) >= b.queueCapacity && !b.dropQueuedDeltaLocked(route) {
			b.recordDeliveryDropLocked(route.bridgeInstanceID, "queue_saturated")
			return ErrDeliveryQueueSaturated
		}
		cloned := cloneDeliveryEvent(event)
		delivery.pendingTerminal = &cloned
		delivery.queuedTerminal = true
		route.queue = append(route.queue, deliveryQueueItem{deliveryID: delivery.deliveryID, kind: deliveryQueueKindTerminal})
		return nil
	default:
		return fmt.Errorf("bridges: unsupported projected delivery event type %q", event.EventType)
	}
}

func (b *Broker) projectEventLocked(delivery *activeDelivery, event DeliveryProjectionEvent) (DeliveryEvent, bool, error) {
	if delivery == nil {
		return DeliveryEvent{}, false, ErrDeliveryNotFound
	}

	switch strings.TrimSpace(event.Type) {
	case "agent_message":
		if event.Text == "" {
			return DeliveryEvent{}, false, nil
		}
		nextContent := MessageContent{Text: delivery.currentContent.Text + event.Text}
		nextType := DeliveryEventTypeDelta
		if delivery.latestSeq == 0 {
			nextType = DeliveryEventTypeStart
		}
		return DeliveryEvent{
			DeliveryID:       delivery.deliveryID,
			BridgeInstanceID: delivery.bridgeInstanceID,
			RoutingKey:       delivery.routingKey,
			DeliveryTarget:   delivery.target,
			Seq:              delivery.latestSeq + 1,
			EventType:        nextType,
			Content:          nextContent,
			Final:            false,
		}, true, nil
	case "done":
		if delivery.latestSeq == 0 && strings.TrimSpace(delivery.currentContent.Text) == "" {
			return DeliveryEvent{}, false, nil
		}
		return DeliveryEvent{
			DeliveryID:       delivery.deliveryID,
			BridgeInstanceID: delivery.bridgeInstanceID,
			RoutingKey:       delivery.routingKey,
			DeliveryTarget:   delivery.target,
			Seq:              delivery.latestSeq + 1,
			EventType:        DeliveryEventTypeFinal,
			Content:          delivery.currentContent,
			Final:            true,
		}, true, nil
	case "error":
		errorText := strings.TrimSpace(event.Error)
		return DeliveryEvent{
			DeliveryID:       delivery.deliveryID,
			BridgeInstanceID: delivery.bridgeInstanceID,
			RoutingKey:       delivery.routingKey,
			DeliveryTarget:   delivery.target,
			Seq:              delivery.latestSeq + 1,
			EventType:        DeliveryEventTypeError,
			Content:          delivery.currentContent,
			Final:            true,
			Metadata: deliveryMetadataJSON(map[string]string{
				"error": errorText,
			}),
		}, true, nil
	default:
		return DeliveryEvent{}, false, nil
	}
}

func (b *Broker) applyQueuedEventLocked(delivery *activeDelivery, event DeliveryEvent) {
	if delivery == nil {
		return
	}

	normalizedType := normalizeDeliveryEventType(event.EventType)
	if event.Seq > delivery.latestSeq {
		delivery.latestSeq = event.Seq
	}
	delivery.latestEventType = normalizedType
	delivery.currentContent = event.Content
	delivery.final = event.Final
	delivery.updatedAt = b.now()

	if normalizedType == DeliveryEventTypeError {
		delivery.errorText = deliveryErrorText(event.Metadata)
		b.recordDeliveryFailureLocked(delivery.bridgeInstanceID, delivery.errorText)
	} else if normalizedType != DeliveryEventTypeResume {
		delivery.errorText = ""
	}
}

func (b *Broker) snapshotLocked(delivery *activeDelivery) DeliverySnapshot {
	return DeliverySnapshot{
		DeliveryID:             delivery.deliveryID,
		SessionID:              delivery.sessionID,
		TurnID:                 delivery.turnID,
		BridgeInstanceID:       delivery.bridgeInstanceID,
		RoutingKey:             delivery.routingKey,
		DeliveryTarget:         delivery.target,
		LatestSeq:              delivery.latestSeq,
		LatestEventType:        delivery.latestEventType,
		CurrentContent:         delivery.currentContent,
		LastSentSeq:            delivery.lastSentSeq,
		LastAckedSeq:           delivery.lastAckedSeq,
		RemoteMessageID:        delivery.remoteMessageID,
		ReplaceRemoteMessageID: delivery.replaceRemoteMessageID,
		Final:                  delivery.final,
		Error:                  delivery.errorText,
		UpdatedAt:              delivery.updatedAt,
	}
}

func (b *Broker) removeDeliveryLocked(route *routeWorker, delivery *activeDelivery) {
	if delivery == nil {
		return
	}
	delete(b.deliveries, delivery.deliveryID)
	delete(b.turnIndex, turnLookupKey(delivery.sessionID, delivery.turnID))
	if deliverySet := b.sessionIndex[delivery.sessionID]; deliverySet != nil {
		delete(deliverySet, delivery.deliveryID)
		if len(deliverySet) == 0 {
			delete(b.sessionIndex, delivery.sessionID)
		}
	}
	if route != nil {
		b.removeQueuedSlotLocked(route, delivery.deliveryID, deliveryQueueKindStart)
		b.removeQueuedSlotLocked(route, delivery.deliveryID, deliveryQueueKindDelta)
		b.removeQueuedSlotLocked(route, delivery.deliveryID, deliveryQueueKindTerminal)
		b.removeQueuedSlotLocked(route, delivery.deliveryID, deliveryQueueKindResume)
	}
}

func (b *Broker) dropQueuedDeltaLocked(route *routeWorker) bool {
	if route == nil {
		return false
	}
	for idx, item := range route.queue {
		if item.kind != deliveryQueueKindDelta {
			continue
		}
		delivery := b.deliveries[item.deliveryID]
		if delivery != nil {
			delivery.queuedDelta = false
			delivery.pendingDelta = nil
		}
		route.queue = append(route.queue[:idx], route.queue[idx+1:]...)
		b.recordDeliveryDropLocked(route.bridgeInstanceID, "coalesced")
		return true
	}
	return false
}

func (b *Broker) metricsLocked(bridgeInstanceID string) *instanceDeliveryMetrics {
	if b.metrics == nil {
		b.metrics = make(map[string]*instanceDeliveryMetrics)
	}
	trimmedID := strings.TrimSpace(bridgeInstanceID)
	if trimmedID == "" {
		return nil
	}
	metrics := b.metrics[trimmedID]
	if metrics == nil {
		metrics = &instanceDeliveryMetrics{
			droppedByReason: make(map[string]int),
		}
		b.metrics[trimmedID] = metrics
	}
	return metrics
}

func (b *Broker) recordDeliveryDropLocked(bridgeInstanceID string, reason string) {
	metrics := b.metricsLocked(bridgeInstanceID)
	if metrics == nil {
		return
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "unknown"
	}
	metrics.droppedByReason[trimmedReason]++
}

func (b *Broker) recordDeliveryIssueLocked(bridgeInstanceID string, message string) {
	metrics := b.metricsLocked(bridgeInstanceID)
	if metrics == nil {
		return
	}
	metrics.lastError = strings.TrimSpace(message)
	metrics.lastErrorAt = b.now()
}

func (b *Broker) recordDeliveryFailureLocked(bridgeInstanceID string, message string) {
	metrics := b.metricsLocked(bridgeInstanceID)
	if metrics == nil {
		return
	}
	metrics.deliveryFailuresTotal++
	metrics.lastError = strings.TrimSpace(message)
	metrics.lastErrorAt = b.now()
}

func (b *Broker) recordDeliverySuccessLocked(bridgeInstanceID string, deliveredAt time.Time) {
	metrics := b.metricsLocked(bridgeInstanceID)
	if metrics == nil {
		return
	}
	metrics.lastSuccessAt = deliveredAt.UTC()
}

func (b *Broker) removeQueuedSlotLocked(route *routeWorker, deliveryID string, kind deliveryQueueKind) bool {
	if route == nil {
		return false
	}
	removed := false
	next := route.queue[:0]
	for _, item := range route.queue {
		if item.deliveryID == deliveryID && item.kind == kind {
			removed = true
			continue
		}
		next = append(next, item)
	}
	route.queue = next

	delivery := b.deliveries[deliveryID]
	if delivery == nil {
		return removed
	}
	switch kind {
	case deliveryQueueKindStart:
		delivery.queuedStart = false
	case deliveryQueueKindDelta:
		delivery.queuedDelta = false
	case deliveryQueueKindTerminal:
		delivery.queuedTerminal = false
	case deliveryQueueKindResume:
		delivery.queuedResume = false
	}
	return removed
}

func (b *Broker) sessionDeliveriesLocked(sessionID string) []string {
	set := b.sessionIndex[sessionID]
	if len(set) == 0 {
		return nil
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

func (b *Broker) signalRoute(route *routeWorker) {
	if route == nil {
		return
	}
	select {
	case route.wakeCh <- struct{}{}:
	default:
	}
}

func (d *activeDelivery) hasQueuedItems() bool {
	return d.queuedStart || d.queuedDelta || d.queuedTerminal || d.queuedResume
}

func turnLookupKey(sessionID string, turnID string) string {
	return strings.TrimSpace(sessionID) + "\x00" + strings.TrimSpace(turnID)
}

func newDeliveryID() string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("del-%d", time.Now().UTC().UnixNano())
	}
	return "del-" + hex.EncodeToString(random[:])
}

func agentEventFingerprint(event DeliveryProjectionEvent) string {
	if fingerprint := strings.TrimSpace(event.Fingerprint); fingerprint != "" {
		return fingerprint
	}
	return strings.TrimSpace(event.Type) + "|" + strings.TrimSpace(event.TurnID) + "|" + event.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + event.Text + "|" + event.Error
}

func deliveryErrorText(raw []byte) string {
	type errorEnvelope struct {
		Error string `json:"error"`
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return ""
	}

	var payload errorEnvelope
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Error)
}
