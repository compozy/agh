package bridges

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
		turnIndex:      make(map[turnIndexKey]string),
		sessionIndex:   make(map[string]map[string]struct{}),
		routes:         make(map[string]*routeWorker),
		bridgeRoutes:   make(map[string]map[string]*routeWorker),
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

// RegisterPromptDelivery binds one prompted session turn to a live delivery
// projection and optionally seeds the broker from already-persisted turn events.
func (b *Broker) RegisterPromptDelivery(
	ctx context.Context,
	reg PromptDeliveryRegistration,
) (*DeliverySnapshot, error) {
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
	deliveryKey := newTurnIndexKey(normalized.SessionID, normalized.TurnID)
	if existingID, ok := b.turnIndex[deliveryKey]; ok {
		existing := b.deliveries[existingID]
		b.mu.Unlock()
		if existing == nil {
			return nil, ErrDeliveryNotFound
		}
		return b.Snapshot(ctx, existingID)
	}

	deliveryID, err := b.reserveDeliveryIDLocked(normalized.DeliveryID)
	if err != nil {
		b.mu.Unlock()
		return nil, err
	}
	delivery := newActiveDelivery(deliveryID, routeHash, normalized, b.now())
	b.deliveries[deliveryID] = delivery
	b.turnIndex[deliveryKey] = deliveryID
	if _, ok := b.sessionIndex[normalized.SessionID]; !ok {
		b.sessionIndex[normalized.SessionID] = make(map[string]struct{})
	}
	b.sessionIndex[normalized.SessionID][deliveryID] = struct{}{}
	route := b.ensureRouteLocked(routeHash, normalized.RoutingKey.BridgeInstanceID, normalized.ExtensionName)
	var routeToSignal *routeWorker
	for _, event := range normalized.SeedEvents {
		seedRoute, err := b.projectDeliveryEventLocked(delivery, event)
		if err != nil {
			b.removeDeliveryLocked(route, delivery)
			b.mu.Unlock()
			return nil, err
		}
		if seedRoute != nil {
			routeToSignal = seedRoute
		}
	}
	snapshot := cloneDeliverySnapshot(b.snapshotLocked(delivery))
	b.mu.Unlock()

	if routeToSignal != nil {
		b.signalRoute(routeToSignal)
	}
	return &snapshot, nil
}

func (b *Broker) reserveDeliveryIDLocked(requestedID string) (string, error) {
	if requestedID != "" {
		if _, exists := b.deliveries[requestedID]; exists {
			return "", fmt.Errorf("%w: %s", ErrDeliveryIDConflict, requestedID)
		}
		return requestedID, nil
	}
	for {
		deliveryID := newDeliveryID()
		if _, exists := b.deliveries[deliveryID]; !exists {
			return deliveryID, nil
		}
	}
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
	if delivery.final {
		b.mu.Unlock()
		return nil
	}
	route := b.ensureRouteLocked(routeHash, normalized.BridgeInstanceID, delivery.extensionName)
	err = b.enqueueEventLocked(route, delivery, normalized)
	if err != nil {
		if errors.Is(err, errProgressEventDropped) {
			b.mu.Unlock()
			return nil
		}
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
	deliveryID, ok := b.turnIndex[newTurnIndexKey(sessionID, turnID)]
	if !ok {
		b.mu.Unlock()
		return nil
	}
	delivery := b.deliveries[deliveryID]
	if delivery == nil {
		b.mu.Unlock()
		return nil
	}

	route, err := b.projectDeliveryEventLocked(delivery, event)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()

	if route != nil {
		b.signalRoute(route)
	}
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
			Operation:        delivery.operation,
			Reference:        cloneDeliveryReference(delivery.reference),
			Error:            &DeliveryErrorDetail{Message: reason},
			ProviderMetadata: cloneRawJSON(delivery.providerMetadata),
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
