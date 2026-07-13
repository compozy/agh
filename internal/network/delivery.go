package network

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
)

var (
	xmlEscapeReplacer = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	protocolGuidanceText = [...]string{
		"If you send a protocol `receipt`, the body must include `for_id` and a valid `status`. " +
			`Use ` + "`status:\"accepted\"`" + ` for normal admission. Use ` + "`status:\"rejected\"`" +
			`, ` + "`\"duplicate\"`" + `, ` + "`\"expired\"`" + `, or ` + "`\"unsupported\"`" +
			` only with a matching ` + "`reason_code`" + ".",
		"If you send a protocol `trace`, the body must include a valid `state` such as " +
			"`working`, `needs_input`, `completed`, `failed`, or `canceled`.",
		"If you send a protocol `capability`, the body must be nested as `{\"capability\":{...}}` and " +
			"include `capability.id`, `capability.summary`, `capability.outcome`, and a canonical `capability.digest`.",
		protocolGuidanceDirectRoomText,
	}
)

const capabilityBodyExample = `  --body '{"capability":{"id":"reply-workflow","summary":"Compact inline checklist.","outcome":"A reusable reply workflow.","version":"1.0.0","digest":"sha256:replace-me","execution_outline":["Inspect request","Draft response"],"requirements":["workspace-write"]}}' \`

const (
	defaultDeliveryRetryBaseDelay         = 250 * time.Millisecond
	defaultDeliveryRetryMaxDelay          = 5 * time.Second
	defaultDeliveryStructuredBodyMaxBytes = 4 * 1024
	deliveryDropReasonQueueFull           = "queue_overflow"
	networkMessageTrustUntrusted          = "untrusted"
	protocolGuidanceDirectRoomText        = "Direct-room chat uses `--kind say --surface direct`."
	compactReplyGuidanceText              = "Full protocol examples were already provided earlier in this session; run `agh network --help` for command details."
)

type deliveryPrompter interface {
	PromptNetwork(
		ctx context.Context,
		sessionID string,
		message string,
		meta ...acp.PromptNetworkMeta,
	) (<-chan acp.AgentEvent, error)
	IsPrompting(sessionID string) bool
}

type deliveryOption func(*deliveryCoordinator)

type deliveryRetryScheduler func(context.Context, time.Duration, func())

type deliveredPromptCost struct {
	PromptSizeBytes       int64
	EstimatedPromptTokens int64
}

type deliveryCoordinator struct {
	lifecycleCtx        context.Context
	prompter            deliveryPrompter
	maxQueueDepth       int
	digestFlushInterval time.Duration
	digestMaxEnvelopes  int
	logger              *slog.Logger
	now                 func() time.Time
	retryBaseDelay      time.Duration
	retryMaxDelay       time.Duration
	scheduleRetry       deliveryRetryScheduler

	mu            sync.Mutex
	queues        map[string]*inboundQueue
	inFlight      map[string]queuedEnvelope
	waiters       map[string][]chan struct{}
	guidance      map[string]deliveryGuidanceState
	sessionTokens atomic.Uint64

	deliveries sync.Map
	wg         sync.WaitGroup

	onDelivered func(
		sessionID string,
		peerID string,
		envelope Envelope,
		mode string,
		latency time.Duration,
		cost deliveredPromptCost,
	)
	onDropped func(sessionID string, envelope Envelope, reason string)
}

type deliveryState struct {
	done chan struct{}
}

type inboundQueue struct {
	mu       sync.Mutex
	maxDepth int
	token    uint64
	items    []queuedEnvelope
}

type enqueueResult struct {
	Depth        int
	DeliveryMode string
	Dropped      *Envelope
}

type queuedEnvelope struct {
	Envelope     Envelope
	PeerID       string
	AcceptedAt   time.Time
	DeliveryMode string
	PromptMode   string
	RetryAttempt int
	SessionToken uint64
}

type deliveryGuidanceState struct {
	replyDelivered    bool
	protocolDelivered bool
	loaded            bool
}

type networkMessageGuidanceMode int

const (
	networkMessageGuidanceVerbose networkMessageGuidanceMode = iota
	networkMessageGuidanceCompact
)

type deliveryCoordinatorStats struct {
	QueuedMessages   int
	QueuedSessions   int
	DeliveryWorkers  int
	InFlightMessages int
}

func withDeliveryLogger(logger *slog.Logger) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.logger = logger
	}
}

func withDeliveryClock(now func() time.Time) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.now = now
	}
}

func withDeliveryDeliveredHook(
	hook func(
		sessionID string,
		peerID string,
		envelope Envelope,
		mode string,
		latency time.Duration,
		cost deliveredPromptCost,
	),
) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.onDelivered = hook
	}
}

func withDeliveryDroppedHook(hook func(sessionID string, envelope Envelope, reason string)) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.onDropped = hook
	}
}

func withDeliveryRetryScheduler(scheduler deliveryRetryScheduler) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.scheduleRetry = scheduler
	}
}

func withDeliveryDigestCoalescing(flushInterval time.Duration, maxEnvelopes int) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.digestFlushInterval = flushInterval
		coordinator.digestMaxEnvelopes = maxEnvelopes
	}
}

func newDeliveryCoordinator(
	ctx context.Context,
	maxQueueDepth int,
	prompter deliveryPrompter,
	opts ...deliveryOption,
) (*deliveryCoordinator, error) {
	if ctx == nil {
		return nil, errors.New("network: delivery context is required")
	}
	if prompter == nil {
		return nil, errors.New("network: delivery prompter is required")
	}
	if maxQueueDepth <= 0 {
		return nil, fmt.Errorf("%w: delivery queue depth must be positive", ErrInvalidField)
	}

	coordinator := &deliveryCoordinator{
		lifecycleCtx:       ctx,
		prompter:           prompter,
		maxQueueDepth:      maxQueueDepth,
		digestMaxEnvelopes: 1,
		logger:             slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
		retryBaseDelay: defaultDeliveryRetryBaseDelay,
		retryMaxDelay:  defaultDeliveryRetryMaxDelay,
		scheduleRetry:  scheduleDeliveryRetry,
		queues:         make(map[string]*inboundQueue),
		inFlight:       make(map[string]queuedEnvelope),
		waiters:        make(map[string][]chan struct{}),
		guidance:       make(map[string]deliveryGuidanceState),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(coordinator)
		}
	}
	if coordinator.logger == nil {
		coordinator.logger = slog.Default()
	}
	if coordinator.now == nil {
		coordinator.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if coordinator.retryBaseDelay <= 0 {
		coordinator.retryBaseDelay = defaultDeliveryRetryBaseDelay
	}
	if coordinator.retryMaxDelay < coordinator.retryBaseDelay {
		coordinator.retryMaxDelay = defaultDeliveryRetryMaxDelay
	}
	if coordinator.scheduleRetry == nil {
		coordinator.scheduleRetry = scheduleDeliveryRetry
	}
	if coordinator.digestMaxEnvelopes <= 0 {
		coordinator.digestMaxEnvelopes = 1
	}
	if coordinator.digestFlushInterval < 0 {
		coordinator.digestFlushInterval = 0
	}
	return coordinator, nil
}

func (c *deliveryCoordinator) accept(ctx context.Context, deliveries []Delivery) error {
	if ctx == nil {
		return errors.New("network: accept context is required")
	}

	for _, delivery := range deliveries {
		if err := c.acceptOne(ctx, delivery); err != nil {
			return err
		}
	}
	return nil
}

func (c *deliveryCoordinator) acceptOne(ctx context.Context, delivery Delivery) error {
	if ctx == nil {
		return errors.New("network: accept context is required")
	}
	if c == nil {
		return errors.New("network: delivery coordinator is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	sessionID := strings.TrimSpace(delivery.SessionID)
	if sessionID == "" {
		return fmt.Errorf("%w: delivery session id is required", ErrMissingField)
	}

	queue := c.queueForSession(sessionID)
	result := queue.enqueue(
		delivery.PeerID,
		delivery.Envelope,
		delivery.Mode,
		c.now(),
		c.prompter.IsPrompting(sessionID),
	)
	if result.Dropped != nil {
		c.logger.Warn(
			"network.message.queue_overflow",
			"session_id", sessionID,
			"dropped_envelope_id", result.Dropped.ID,
			"queue_depth", result.Depth,
		)
		if c.onDropped != nil {
			c.onDropped(sessionID, cloneEnvelope(*result.Dropped), deliveryDropReasonQueueFull)
		}
	}
	if result.DeliveryMode == "queued" {
		c.logger.Info(
			"network.message.queued",
			"session_id", sessionID,
			"message_id", delivery.Envelope.ID,
			"kind", string(delivery.Envelope.Kind),
			"channel", delivery.Envelope.Channel,
			"queue_depth", result.Depth,
		)
	}
	c.notifyWaiters(sessionID)

	if !c.prompter.IsPrompting(sessionID) {
		c.trigger(sessionID)
	}
	return nil
}

func (c *deliveryCoordinator) onTurnEnd(sessionID string) {
	if c == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}
	c.trigger(target)
}

func (c *deliveryCoordinator) inbox(sessionID string) []Envelope {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil {
		return nil
	}
	return queue.snapshot()
}

func (c *deliveryCoordinator) waitInbox(
	ctx context.Context,
	sessionID string,
	channel string,
) ([]Envelope, error) {
	if ctx == nil {
		return nil, errors.New("network: wait inbox context is required")
	}
	if c == nil {
		return nil, errors.New("network: delivery coordinator is required")
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return nil, fmt.Errorf("%w: delivery session id is required", ErrMissingField)
	}
	channel = strings.TrimSpace(channel)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := c.lifecycleCtx.Err(); err != nil {
			return nil, err
		}
		if messages := filterInboxByChannel(c.inbox(target), channel); len(messages) > 0 {
			return messages, nil
		}

		waiter := c.registerWaiter(target)
		if messages := filterInboxByChannel(c.inbox(target), channel); len(messages) > 0 {
			c.unregisterWaiter(target, waiter)
			return messages, nil
		}

		select {
		case <-ctx.Done():
			c.unregisterWaiter(target, waiter)
			return nil, ctx.Err()
		case <-c.lifecycleCtx.Done():
			c.unregisterWaiter(target, waiter)
			return nil, c.lifecycleCtx.Err()
		case <-waiter:
		}
	}
}

func (c *deliveryCoordinator) queueDepth(sessionID string) int {
	if c == nil {
		return 0
	}

	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil {
		return 0
	}
	return queue.len()
}

func filterInboxByChannel(envelopes []Envelope, channel string) []Envelope {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return envelopes
	}
	filtered := make([]Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if strings.TrimSpace(envelope.Channel) == channel {
			filtered = append(filtered, envelope)
		}
	}
	return filtered
}

func (c *deliveryCoordinator) dropSession(sessionID string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	target := strings.TrimSpace(sessionID)
	delete(c.queues, target)
	delete(c.inFlight, target)
	delete(c.guidance, target)
	waiters := c.waiters[target]
	delete(c.waiters, target)
	for _, waiter := range waiters {
		close(waiter)
	}
}

func (c *deliveryCoordinator) registerWaiter(sessionID string) chan struct{} {
	waiter := make(chan struct{})
	c.mu.Lock()
	defer c.mu.Unlock()
	target := strings.TrimSpace(sessionID)
	c.waiters[target] = append(c.waiters[target], waiter)
	return waiter
}

func (c *deliveryCoordinator) unregisterWaiter(sessionID string, waiter chan struct{}) {
	if waiter == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	target := strings.TrimSpace(sessionID)
	waiters := c.waiters[target]
	for index, candidate := range waiters {
		if candidate != waiter {
			continue
		}
		c.waiters[target] = append(waiters[:index], waiters[index+1:]...)
		if len(c.waiters[target]) == 0 {
			delete(c.waiters, target)
		}
		return
	}
}

func (c *deliveryCoordinator) notifyWaiters(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	target := strings.TrimSpace(sessionID)
	waiters := c.waiters[target]
	delete(c.waiters, target)
	for _, waiter := range waiters {
		close(waiter)
	}
}

func (c *deliveryCoordinator) wait() {
	if c == nil {
		return
	}
	c.wg.Wait()
}

func (c *deliveryCoordinator) queueForSession(sessionID string) *inboundQueue {
	target := strings.TrimSpace(sessionID)

	c.mu.Lock()
	defer c.mu.Unlock()

	if queue, ok := c.queues[target]; ok {
		return queue
	}

	queue := newInboundQueueWithToken(c.maxQueueDepth, c.sessionTokens.Add(1))
	c.queues[target] = queue
	return queue
}

func (c *deliveryCoordinator) guidanceModeForDelivery(
	sessionID string,
	envelope Envelope,
) networkMessageGuidanceMode {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return networkMessageGuidanceVerbose
	}

	c.mu.Lock()
	state := c.guidance[target]
	c.mu.Unlock()
	if !state.replyDelivered {
		return networkMessageGuidanceVerbose
	}
	if lifecycleWorkID(envelope) != "" && !state.protocolDelivered {
		return networkMessageGuidanceVerbose
	}
	return networkMessageGuidanceCompact
}

func (c *deliveryCoordinator) markGuidanceDelivered(sessionID string, item queuedEnvelope) {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	c.mu.Lock()
	queue := c.queues[target]
	if queue == nil || queue.token != item.SessionToken {
		c.mu.Unlock()
		return
	}

	state := c.guidance[target]
	state.replyDelivered = true
	state.loaded = true
	if lifecycleWorkID(item.Envelope) != "" {
		state.protocolDelivered = true
	}
	c.guidance[target] = state
	c.mu.Unlock()
}

func (c *deliveryCoordinator) trigger(sessionID string) {
	if c == nil {
		return
	}
	if err := c.lifecycleCtx.Err(); err != nil {
		return
	}
	if c.queueDepth(sessionID) == 0 {
		return
	}

	state := &deliveryState{done: make(chan struct{})}
	if _, loaded := c.deliveries.LoadOrStore(sessionID, state); loaded {
		return
	}

	c.wg.Add(1)
	go c.runWorker(sessionID, state)
}

func (c *deliveryCoordinator) runWorker(sessionID string, state *deliveryState) {
	defer c.wg.Done()
	defer close(state.done)
	defer c.deliveries.Delete(sessionID)

	target := strings.TrimSpace(sessionID)
	for {
		if err := c.lifecycleCtx.Err(); err != nil {
			return
		}
		if c.prompter.IsPrompting(target) {
			return
		}

		item, ok := c.dequeue(target)
		if !ok {
			return
		}
		if !c.processQueuedItem(target, item, state) {
			return
		}
	}
}

func (c *deliveryCoordinator) processQueuedItem(sessionID string, item queuedEnvelope, state *deliveryState) bool {
	c.markInFlight(sessionID, item)
	batch := c.collectDigestBatch(sessionID, item)
	envelope := batch[0].Envelope

	message, err := c.formatQueuedDeliveryBatch(sessionID, batch)
	if err != nil {
		c.handleRenderFailure(sessionID, batch, state, err)
		return false
	}

	cost := deliveredPromptCostForMessage(message)
	events, err := c.prompter.PromptNetwork(
		c.lifecycleCtx,
		sessionID,
		message,
		promptNetworkMeta(envelope, cost, batch[0].PromptMode),
	)
	if err != nil {
		c.handleDeliveryFailure(sessionID, batch, state, err)
		return false
	}
	if !c.drainPromptEvents(events) {
		c.handleInterruptedDelivery(sessionID, batch)
		return false
	}

	c.finishDeliveredMessages(sessionID, batch, cost)
	return true
}

func (c *deliveryCoordinator) collectDigestBatch(sessionID string, first queuedEnvelope) []queuedEnvelope {
	if normalizePromptDeliveryMode(first.PromptMode) != store.NetworkSubscriptionModeDigest ||
		c.digestMaxEnvelopes <= 1 {
		return []queuedEnvelope{first}
	}
	if c.digestFlushInterval > 0 && c.hasDigestBatchCandidate(sessionID, first) {
		timer := time.NewTimer(c.digestFlushInterval)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-c.lifecycleCtx.Done():
			return []queuedEnvelope{first}
		}
	}

	batch := []queuedEnvelope{first}
	batch = append(batch, c.dequeueDigestBatch(sessionID, first, c.digestMaxEnvelopes-1)...)
	return batch
}

func (c *deliveryCoordinator) formatQueuedDeliveryBatch(
	sessionID string,
	batch []queuedEnvelope,
) (string, error) {
	if len(batch) == 0 {
		return "", errors.New("network: delivery batch is empty")
	}
	guidanceMode := c.guidanceModeForDelivery(sessionID, batch[0].Envelope)
	if len(batch) == 1 || normalizePromptDeliveryMode(batch[0].PromptMode) != store.NetworkSubscriptionModeDigest {
		return c.formatNetworkMessageWithDeliveryMode(batch[0].Envelope, guidanceMode, batch[0].PromptMode)
	}
	envelopes := make([]Envelope, 0, len(batch))
	for _, item := range batch {
		envelopes = append(envelopes, item.Envelope)
	}
	return formatNetworkDigestBatchMessage(envelopes, guidanceMode)
}

func deliveredPromptCostForMessage(message string) deliveredPromptCost {
	sizeBytes := int64(len(message))
	return deliveredPromptCost{
		PromptSizeBytes:       sizeBytes,
		EstimatedPromptTokens: estimatePromptTokens(sizeBytes),
	}
}

func estimatePromptTokens(sizeBytes int64) int64 {
	if sizeBytes <= 0 {
		return 0
	}
	return (sizeBytes + 3) / 4
}

func promptNetworkMeta(envelope Envelope, cost deliveredPromptCost, deliveryMode ...string) acp.PromptNetworkMeta {
	meta := acp.PromptNetworkMeta{
		MessageID:             envelope.ID,
		Kind:                  string(envelope.Kind),
		Channel:               envelope.Channel,
		From:                  envelope.From,
		Mentions:              normalizeEnvelopeMentions(envelope.Mentions),
		Trust:                 networkMessageTrustUntrusted,
		DeliveryMode:          normalizePromptDeliveryMode(firstPromptDeliveryMode(deliveryMode)),
		PromptSizeBytes:       cost.PromptSizeBytes,
		EstimatedPromptTokens: cost.EstimatedPromptTokens,
	}
	if envelope.To != nil {
		meta.To = strings.TrimSpace(*envelope.To)
	}
	if envelope.Surface != nil {
		meta.Surface = strings.TrimSpace(string(*envelope.Surface))
	}
	if envelope.ThreadID != nil {
		meta.ThreadID = strings.TrimSpace(*envelope.ThreadID)
	}
	if envelope.DirectID != nil {
		meta.DirectID = strings.TrimSpace(*envelope.DirectID)
	}
	if workID := lifecycleWorkID(envelope); workID != "" {
		meta.WorkID = workID
	}
	if envelope.ReplyTo != nil {
		meta.ReplyTo = strings.TrimSpace(*envelope.ReplyTo)
	}
	if envelope.TraceID != nil {
		meta.TraceID = strings.TrimSpace(*envelope.TraceID)
	}
	if envelope.CausationID != nil {
		meta.CausationID = strings.TrimSpace(*envelope.CausationID)
	}
	return meta.Normalize()
}

func (c *deliveryCoordinator) handleRenderFailure(
	sessionID string,
	items []queuedEnvelope,
	state *deliveryState,
	err error,
) {
	c.clearInFlight(sessionID)
	retryItems := nextRetryBatch(items)
	requeued := c.requeueFrontBatch(sessionID, retryItems)
	item := firstQueuedBatchItem(retryItems)
	c.logger.Warn(
		"network.message.render_failed",
		"session_id", sessionID,
		"envelope_id", item.Envelope.ID,
		"error", err,
		"retry_attempt", item.RetryAttempt,
	)
	if requeued && json.Valid(item.Envelope.Body) {
		c.retryAfterWorkerExit(sessionID, item, state)
	}
}

func (c *deliveryCoordinator) handleDeliveryFailure(
	sessionID string,
	items []queuedEnvelope,
	state *deliveryState,
	err error,
) {
	c.clearInFlight(sessionID)
	retryItems := nextRetryBatch(items)
	requeued := c.requeueFrontBatch(sessionID, retryItems)
	item := firstQueuedBatchItem(retryItems)
	c.logger.Warn(
		"network.message.delivery_failed",
		"session_id", sessionID,
		"envelope_id", item.Envelope.ID,
		"error", err,
		"retry_attempt", item.RetryAttempt,
	)
	if requeued {
		c.retryAfterWorkerExit(sessionID, item, state)
	}
}

func (c *deliveryCoordinator) handleInterruptedDelivery(sessionID string, items []queuedEnvelope) {
	c.clearInFlight(sessionID)
	item := firstQueuedBatchItem(items)
	c.logger.Warn(
		"network.message.delivery_interrupted",
		"session_id", sessionID,
		"message_id", item.Envelope.ID,
		"kind", string(item.Envelope.Kind),
		"channel", item.Envelope.Channel,
		"delivery_mode", item.DeliveryMode,
		"error", c.lifecycleCtx.Err(),
	)
}

func (c *deliveryCoordinator) finishDeliveredMessages(
	sessionID string,
	items []queuedEnvelope,
	cost deliveredPromptCost,
) {
	c.clearInFlight(sessionID)
	if len(items) == 0 {
		return
	}
	for _, item := range items {
		c.markGuidanceDelivered(sessionID, item)
	}

	item := items[0]
	latency := max(c.now().Sub(item.AcceptedAt), 0)

	c.logger.Info(
		"network.message.delivered",
		"session_id", sessionID,
		"message_id", item.Envelope.ID,
		"batch_size", len(items),
		"kind", string(item.Envelope.Kind),
		"channel", item.Envelope.Channel,
		"delivery_mode", item.DeliveryMode,
		"prompt_size_bytes", cost.PromptSizeBytes,
		"estimated_prompt_tokens", cost.EstimatedPromptTokens,
		"latency_ms", latency.Milliseconds(),
	)
	if c.onDelivered != nil {
		allocations := deliveredPromptCostAllocations(items, cost)
		for idx, deliveredItem := range items {
			itemLatency := max(c.now().Sub(deliveredItem.AcceptedAt), 0)
			c.onDelivered(
				sessionID,
				deliveredItem.PeerID,
				deliveredItem.Envelope,
				deliveredItem.DeliveryMode,
				itemLatency,
				allocations[idx],
			)
		}
	}
}

func (c *deliveryCoordinator) drainPromptEvents(events <-chan acp.AgentEvent) bool {
	if events == nil {
		return true
	}

	for {
		select {
		case <-c.lifecycleCtx.Done():
			return false
		case _, ok := <-events:
			if !ok {
				return true
			}
		}
	}
}

func (c *deliveryCoordinator) dequeue(sessionID string) (queuedEnvelope, bool) {
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil {
		return queuedEnvelope{}, false
	}
	return queue.dequeue()
}

func (c *deliveryCoordinator) dequeueDigestBatch(
	sessionID string,
	first queuedEnvelope,
	limit int,
) []queuedEnvelope {
	if limit <= 0 {
		return nil
	}
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil || queue.token != first.SessionToken {
		return nil
	}
	return queue.dequeueDigestBatch(first.PeerID, first.SessionToken, limit)
}

func (c *deliveryCoordinator) hasDigestBatchCandidate(sessionID string, first queuedEnvelope) bool {
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil || queue.token != first.SessionToken {
		return false
	}
	return queue.hasDigestBatchCandidate(first.PeerID, first.SessionToken)
}

func (c *deliveryCoordinator) requeueFrontBatch(sessionID string, items []queuedEnvelope) bool {
	if len(items) == 0 {
		return false
	}
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil || queue.token != items[0].SessionToken {
		return false
	}
	queue.prependBatch(items)
	return true
}

func (c *deliveryCoordinator) retryAfterWorkerExit(sessionID string, item queuedEnvelope, state *deliveryState) {
	if c == nil || state == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	delay := c.retryDelayFor(item.RetryAttempt)
	c.wg.Go(func() {
		select {
		case <-state.done:
		case <-c.lifecycleCtx.Done():
			return
		}

		c.scheduleRetry(c.lifecycleCtx, delay, func() {
			if err := c.lifecycleCtx.Err(); err != nil {
				return
			}
			if c.prompter.IsPrompting(target) {
				return
			}
			if c.queueDepth(target) == 0 {
				return
			}
			c.trigger(target)
		})
	})
}

func (c *deliveryCoordinator) retryDelayFor(attempt int) time.Duration {
	if c == nil {
		return defaultDeliveryRetryBaseDelay
	}
	delay := c.retryBaseDelay
	for i := 1; i < attempt; i++ {
		if delay >= c.retryMaxDelay/2 {
			return c.retryMaxDelay
		}
		delay *= 2
	}
	if delay > c.retryMaxDelay {
		return c.retryMaxDelay
	}
	return delay
}

func scheduleDeliveryRetry(ctx context.Context, delay time.Duration, fn func()) {
	if fn == nil {
		return
	}
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return
		default:
			fn()
			return
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		fn()
	}
}

func (c *deliveryCoordinator) stats() deliveryCoordinatorStats {
	if c == nil {
		return deliveryCoordinatorStats{}
	}

	c.mu.Lock()
	queues := make([]*inboundQueue, 0, len(c.queues))
	for _, queue := range c.queues {
		queues = append(queues, queue)
	}
	inFlightMessages := len(c.inFlight)
	c.mu.Unlock()

	stats := deliveryCoordinatorStats{
		InFlightMessages: inFlightMessages,
	}
	for _, queue := range queues {
		depth := queue.len()
		if depth <= 0 {
			continue
		}
		stats.QueuedMessages += depth
		stats.QueuedSessions++
	}
	c.deliveries.Range(func(_, _ any) bool {
		stats.DeliveryWorkers++
		return true
	})
	return stats
}

func (c *deliveryCoordinator) queueDepthMetrics() []MetricSample {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	queues := make([]*inboundQueue, 0, len(c.queues))
	for _, queue := range c.queues {
		queues = append(queues, queue)
	}
	c.mu.Unlock()

	depths := make(map[surfaceMetricKey]int64)
	for _, queue := range queues {
		for _, envelope := range queue.snapshot() {
			key := surfaceMetricKey{
				channel: strings.TrimSpace(envelope.Channel),
				surface: surfaceLabel(envelope.Surface),
			}
			depths[key]++
		}
	}
	samples := make([]MetricSample, 0, len(depths))
	for key, depth := range depths {
		samples = append(samples, surfaceMetricSample("network_delivery_queue_depth", key, depth))
	}
	sortMetricSamples(samples)
	return samples
}

func (c *deliveryCoordinator) markInFlight(sessionID string, item queuedEnvelope) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight[strings.TrimSpace(sessionID)] = item
}

func (c *deliveryCoordinator) clearInFlight(sessionID string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.inFlight, strings.TrimSpace(sessionID))
}

func newInboundQueue(maxDepth int) *inboundQueue {
	return newInboundQueueWithToken(maxDepth, 0)
}

func newInboundQueueWithToken(maxDepth int, token uint64) *inboundQueue {
	return &inboundQueue{maxDepth: maxDepth, token: token}
}

func (q *inboundQueue) enqueue(
	peerID string,
	envelope Envelope,
	promptMode string,
	acceptedAt time.Time,
	prompting bool,
) enqueueResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	var dropped *Envelope
	wasEmpty := len(q.items) == 0
	if len(q.items) >= q.maxDepth {
		evicted := cloneEnvelope(q.items[0].Envelope)
		dropped = &evicted
		copy(q.items[0:], q.items[1:])
		q.items = q.items[:len(q.items)-1]
	}
	deliveryMode := "queued"
	if !prompting && wasEmpty {
		deliveryMode = "immediate"
	}
	q.items = append(q.items, queuedEnvelope{
		Envelope:     cloneEnvelope(envelope),
		PeerID:       strings.TrimSpace(peerID),
		AcceptedAt:   acceptedAt.UTC(),
		DeliveryMode: deliveryMode,
		PromptMode:   strings.TrimSpace(promptMode),
		SessionToken: q.token,
	})

	return enqueueResult{
		Depth:        len(q.items),
		DeliveryMode: deliveryMode,
		Dropped:      dropped,
	}
}

func (q *inboundQueue) prependBatch(items []queuedEnvelope) {
	q.mu.Lock()
	defer q.mu.Unlock()

	prefix := make([]queuedEnvelope, 0, len(items))
	for _, item := range items {
		prefix = append(prefix, cloneQueuedEnvelope(item))
	}
	q.items = append(prefix, q.items...)
}

func (q *inboundQueue) dequeue() (queuedEnvelope, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return queuedEnvelope{}, false
	}

	envelope := cloneQueuedEnvelope(q.items[0])
	copy(q.items[0:], q.items[1:])
	q.items = q.items[:len(q.items)-1]
	return envelope, true
}

func (q *inboundQueue) dequeueDigestBatch(peerID string, sessionToken uint64, limit int) []queuedEnvelope {
	q.mu.Lock()
	defer q.mu.Unlock()

	if limit <= 0 || len(q.items) == 0 || q.token != sessionToken {
		return nil
	}
	targetPeerID := strings.TrimSpace(peerID)
	out := make([]queuedEnvelope, 0, min(limit, len(q.items)))
	for len(q.items) > 0 && len(out) < limit {
		item := q.items[0]
		if normalizePromptDeliveryMode(item.PromptMode) != store.NetworkSubscriptionModeDigest ||
			strings.TrimSpace(item.PeerID) != targetPeerID {
			break
		}
		out = append(out, cloneQueuedEnvelope(item))
		copy(q.items[0:], q.items[1:])
		q.items = q.items[:len(q.items)-1]
	}
	return out
}

func (q *inboundQueue) hasDigestBatchCandidate(peerID string, sessionToken uint64) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 || q.token != sessionToken {
		return false
	}
	item := q.items[0]
	return normalizePromptDeliveryMode(item.PromptMode) == store.NetworkSubscriptionModeDigest &&
		strings.TrimSpace(item.PeerID) == strings.TrimSpace(peerID)
}

func (item queuedEnvelope) withNextRetryAttempt() queuedEnvelope {
	next := cloneQueuedEnvelope(item)
	next.RetryAttempt++
	return next
}

func nextRetryBatch(items []queuedEnvelope) []queuedEnvelope {
	if len(items) == 0 {
		return nil
	}
	out := make([]queuedEnvelope, 0, len(items))
	for _, item := range items {
		out = append(out, item.withNextRetryAttempt())
	}
	return out
}

func firstQueuedBatchItem(items []queuedEnvelope) queuedEnvelope {
	if len(items) == 0 {
		return queuedEnvelope{}
	}
	return items[0]
}

func deliveredPromptCostAllocations(items []queuedEnvelope, total deliveredPromptCost) []deliveredPromptCost {
	if len(items) == 0 {
		return nil
	}
	weights := make([]int64, 0, len(items))
	var totalWeight int64
	for _, item := range items {
		weight := digestCostWeight(item.Envelope)
		if weight <= 0 {
			weight = 1
		}
		weights = append(weights, weight)
		totalWeight += weight
	}
	if totalWeight <= 0 {
		totalWeight = int64(len(items))
		for idx := range weights {
			weights[idx] = 1
		}
	}
	out := make([]deliveredPromptCost, len(items))
	var allocatedBytes int64
	var allocatedTokens int64
	for idx, weight := range weights {
		if idx == len(items)-1 {
			out[idx] = deliveredPromptCost{
				PromptSizeBytes:       total.PromptSizeBytes - allocatedBytes,
				EstimatedPromptTokens: total.EstimatedPromptTokens - allocatedTokens,
			}
			break
		}
		promptBytes := total.PromptSizeBytes * weight / totalWeight
		promptTokens := total.EstimatedPromptTokens * weight / totalWeight
		out[idx] = deliveredPromptCost{
			PromptSizeBytes:       promptBytes,
			EstimatedPromptTokens: promptTokens,
		}
		allocatedBytes += promptBytes
		allocatedTokens += promptTokens
	}
	return out
}

func digestCostWeight(envelope Envelope) int64 {
	deliveryBody, err := networkDeliveryBodyForEnvelope(envelope)
	if err != nil {
		return max(int64(len(envelope.Body)), 1)
	}
	if deliveryBody.plainTextBody != "" {
		return max(int64(len(deliveryBody.plainTextBody)), 1)
	}
	if len(deliveryBody.canonicalBody) > 0 {
		return int64(len(deliveryBody.canonicalBody))
	}
	if deliveryBody.preview != "" {
		return int64(len(deliveryBody.preview))
	}
	return 1
}

func (q *inboundQueue) snapshot() []Envelope {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	out := make([]Envelope, 0, len(q.items))
	for _, envelope := range q.items {
		out = append(out, cloneEnvelope(envelope.Envelope))
	}
	return out
}

func (q *inboundQueue) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func formatNetworkMessage(envelope Envelope) (string, error) {
	return formatNetworkMessageWithGuidance(envelope, networkMessageGuidanceVerbose)
}

func formatNetworkMessageWithGuidance(
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
) (string, error) {
	return formatNetworkMessageWithDeliveryMode(envelope, guidanceMode, store.NetworkSubscriptionModeFull)
}

type networkDeliveryBody struct {
	preview       string
	plainTextBody string
	canonicalBody []byte
}

func networkDeliveryBodyForEnvelope(envelope Envelope) (networkDeliveryBody, error) {
	body, err := envelope.DecodeBody()
	switch {
	case err == nil:
		if say, ok := body.(SayBody); ok && len(say.Artifacts) == 0 {
			return networkDeliveryBody{
				preview:       strings.TrimSpace(say.Text),
				plainTextBody: say.Text,
			}, nil
		}
		canonicalBody, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return networkDeliveryBody{}, fmt.Errorf("network: marshal canonical body for delivery: %w", marshalErr)
		}
		return networkDeliveryBody{
			preview:       previewForBody(body),
			canonicalBody: canonicalBody,
		}, nil
	case !json.Valid(envelope.Body):
		return networkDeliveryBody{}, fmt.Errorf("network: decode envelope body for delivery: %w", err)
	default:
		var compact bytes.Buffer
		if compactErr := json.Compact(&compact, envelope.Body); compactErr != nil {
			return networkDeliveryBody{}, fmt.Errorf("network: compact raw envelope body for delivery: %w", compactErr)
		}
		return networkDeliveryBody{canonicalBody: compact.Bytes()}, nil
	}
}

func formatNetworkMessageWithDeliveryMode(
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
	deliveryMode string,
) (string, error) {
	return formatNetworkMessageWithDeliveryModeAndLimit(envelope, guidanceMode, deliveryMode, 0)
}

func (c *deliveryCoordinator) formatNetworkMessageWithDeliveryMode(
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
	deliveryMode string,
) (string, error) {
	return formatNetworkMessageWithDeliveryModeAndLimit(
		envelope,
		guidanceMode,
		deliveryMode,
		defaultDeliveryStructuredBodyMaxBytes,
	)
}

func formatNetworkMessageWithDeliveryModeAndLimit(
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
	deliveryMode string,
	structuredBodyMaxBytes int,
) (string, error) {
	if normalizePromptDeliveryMode(deliveryMode) == store.NetworkSubscriptionModeDigest {
		return formatNetworkDigestMessage(envelope, guidanceMode)
	}

	deliveryBody, err := networkDeliveryBodyForEnvelope(envelope)
	if err != nil {
		return "", err
	}
	encodedBody := ""
	structuredBodyTruncated := false
	originalBodyBytes := len(deliveryBody.canonicalBody)
	deliveredBodyBytes := originalBodyBytes
	if deliveryBody.plainTextBody == "" {
		cappedBody, truncated, capErr := cappedStructuredDeliveryBody(deliveryBody, structuredBodyMaxBytes)
		if capErr != nil {
			return "", capErr
		}
		structuredBodyTruncated = truncated
		deliveredBodyBytes = len(cappedBody)
		encodedBody = base64.StdEncoding.EncodeToString(cappedBody)
	}

	var builder strings.Builder
	builder.Grow(
		base64.StdEncoding.EncodedLen(len(deliveryBody.canonicalBody)) +
			len(deliveryBody.preview)*6 +
			len(deliveryBody.plainTextBody)*2 +
			2048,
	)
	builder.WriteString("<network-message")
	writeNetworkMessageAttrs(&builder, envelope, "")
	builder.WriteString(">\n")
	if deliveryBody.preview != "" {
		builder.WriteString("  <network-preview encoding=\"xml-escaped\">")
		builder.WriteString(xmlEscape(deliveryBody.preview))
		builder.WriteString("</network-preview>\n")
	}
	if deliveryBody.plainTextBody != "" {
		builder.WriteString("  <network-body encoding=\"text\">")
		builder.WriteString(xmlEscape(deliveryBody.plainTextBody))
		builder.WriteString("</network-body>\n")
	} else {
		builder.WriteString("  <network-body encoding=\"base64-json\">")
		builder.WriteString(encodedBody)
		builder.WriteString("</network-body>\n")
		if structuredBodyTruncated {
			builder.WriteString("  <network-body-truncated")
			writeXMLAttr(&builder, "original-bytes", strconv.Itoa(originalBodyBytes))
			writeXMLAttr(&builder, "delivered-bytes", strconv.Itoa(deliveredBodyBytes))
			builder.WriteString(">true</network-body-truncated>\n")
		}
	}
	builder.WriteString("</network-message>")
	writeReplyGuidance(&builder, envelope, guidanceMode)

	return builder.String(), nil
}

func cappedStructuredDeliveryBody(
	deliveryBody networkDeliveryBody,
	maxBytes int,
) ([]byte, bool, error) {
	if maxBytes <= 0 || len(deliveryBody.canonicalBody) <= maxBytes {
		return deliveryBody.canonicalBody, false, nil
	}
	truncated := struct {
		Truncated     bool   `json:"truncated"`
		OriginalBytes int    `json:"original_bytes"`
		DeliveredMax  int    `json:"delivered_max_bytes"`
		Preview       string `json:"preview,omitempty"`
	}{
		Truncated:     true,
		OriginalBytes: len(deliveryBody.canonicalBody),
		DeliveredMax:  maxBytes,
		Preview:       strings.TrimSpace(deliveryBody.preview),
	}
	payload, err := json.Marshal(truncated)
	if err != nil {
		return nil, false, fmt.Errorf("network: marshal truncated delivery body: %w", err)
	}
	if len(payload) > maxBytes {
		truncated.Preview = ""
		payload, err = json.Marshal(truncated)
		if err != nil {
			return nil, false, fmt.Errorf("network: marshal minimal truncated delivery body: %w", err)
		}
	}
	if len(payload) > maxBytes {
		minimal := []byte(`{"truncated":true}`)
		if len(minimal) <= maxBytes {
			return minimal, true, nil
		}
		return []byte(`0`), true, nil
	}
	return payload, true, nil
}

func formatNetworkDigestMessage(
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
) (string, error) {
	return formatNetworkDigestBatchMessage([]Envelope{envelope}, guidanceMode)
}

func formatNetworkDigestBatchMessage(
	envelopes []Envelope,
	guidanceMode networkMessageGuidanceMode,
) (string, error) {
	if len(envelopes) == 0 {
		return "", errors.New("network: digest batch is empty")
	}
	if len(envelopes) == 1 {
		return formatSingleNetworkDigestMessage(envelopes[0], guidanceMode)
	}
	type digestPreview struct {
		envelope Envelope
		preview  string
	}
	previews := make([]digestPreview, 0, len(envelopes))
	var previewBytes int
	for _, envelope := range envelopes {
		preview, err := networkDigestPreview(envelope)
		if err != nil {
			return "", err
		}
		previews = append(previews, digestPreview{envelope: envelope, preview: preview})
		previewBytes += len(preview)
	}

	first := envelopes[0]
	var builder strings.Builder
	builder.Grow(previewBytes*6 + len(envelopes)*512 + 1024)
	builder.WriteString("<network-digest-batch")
	writeNetworkMessageAttrs(&builder, first, store.NetworkSubscriptionModeDigest)
	writeXMLAttr(&builder, "count", strconv.Itoa(len(envelopes)))
	builder.WriteString(">\n")
	for _, item := range previews {
		builder.WriteString("  <network-digest-message")
		writeNetworkMessageAttrs(&builder, item.envelope, "")
		builder.WriteString(" encoding=\"xml-escaped\">")
		builder.WriteString(xmlEscape(item.preview))
		builder.WriteString("</network-digest-message>\n")
	}
	builder.WriteString("</network-digest-batch>")
	writeReplyGuidance(&builder, first, guidanceMode)
	return builder.String(), nil
}

func formatSingleNetworkDigestMessage(
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
) (string, error) {
	preview, err := networkDigestPreview(envelope)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.Grow(len(preview)*6 + 1024)
	builder.WriteString("<network-message")
	writeNetworkMessageAttrs(&builder, envelope, store.NetworkSubscriptionModeDigest)
	builder.WriteString(">\n")
	builder.WriteString("  <network-digest encoding=\"xml-escaped\">")
	builder.WriteString(xmlEscape(preview))
	builder.WriteString("</network-digest>\n")
	builder.WriteString("</network-message>")
	writeReplyGuidance(&builder, envelope, guidanceMode)
	return builder.String(), nil
}

func networkDigestPreview(envelope Envelope) (string, error) {
	body, err := envelope.DecodeBody()
	preview := ""
	switch {
	case err == nil:
		preview = previewForBody(body)
	case !json.Valid(envelope.Body):
		return "", fmt.Errorf("network: decode envelope body for digest delivery: %w", err)
	default:
		var compact bytes.Buffer
		if compactErr := json.Compact(&compact, envelope.Body); compactErr != nil {
			return "", fmt.Errorf("network: compact raw envelope body for digest delivery: %w", compactErr)
		}
		preview = compact.String()
	}
	preview = strings.TrimSpace(preview)
	if preview == "" {
		preview = string(envelope.Kind)
	}
	return preview, nil
}

func writeNetworkMessageAttrs(builder *strings.Builder, envelope Envelope, deliveryMode string) {
	writeXMLAttr(builder, "id", envelope.ID)
	writeXMLAttr(builder, "from", envelope.From)
	writeXMLAttr(builder, "channel", envelope.Channel)
	writeXMLAttr(builder, "kind", string(envelope.Kind))
	if deliveryMode != "" {
		writeXMLAttr(builder, "delivery-mode", deliveryMode)
	}
	if envelope.Surface != nil {
		writeXMLAttr(builder, "surface", string(*envelope.Surface))
	}
	if envelope.ThreadID != nil {
		writeXMLAttr(builder, "thread-id", *envelope.ThreadID)
	}
	if envelope.DirectID != nil {
		writeXMLAttr(builder, "direct-id", *envelope.DirectID)
	}
	if envelope.To != nil {
		writeXMLAttr(builder, "to", *envelope.To)
	}
	if len(envelope.Mentions) > 0 {
		writeXMLAttr(builder, "mentions", strings.Join(normalizeEnvelopeMentions(envelope.Mentions), ","))
	}
	if envelope.WorkID != nil {
		if workID := lifecycleWorkID(envelope); workID != "" {
			writeXMLAttr(builder, "work-id", workID)
		}
	}
	if envelope.ReplyTo != nil {
		writeXMLAttr(builder, "reply-to", *envelope.ReplyTo)
	}
	if envelope.TraceID != nil {
		writeXMLAttr(builder, "trace-id", *envelope.TraceID)
	}
	if envelope.CausationID != nil {
		writeXMLAttr(builder, "causation-id", *envelope.CausationID)
	}
	if envelope.ExpiresAt != nil {
		writeXMLAttr(builder, "expires-at", strconv.FormatInt(*envelope.ExpiresAt, 10))
	}
	writeXMLAttr(builder, "trust", networkMessageTrustUntrusted)
}

func normalizePromptDeliveryMode(deliveryMode string) string {
	switch strings.TrimSpace(deliveryMode) {
	case store.NetworkSubscriptionModeDigest:
		return store.NetworkSubscriptionModeDigest
	case store.NetworkSubscriptionModeMute:
		return store.NetworkSubscriptionModeMute
	default:
		return store.NetworkSubscriptionModeFull
	}
}

func firstPromptDeliveryMode(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return store.NetworkSubscriptionModeFull
}

func writeReplyGuidance(
	builder *strings.Builder,
	envelope Envelope,
	guidanceMode networkMessageGuidanceMode,
) {
	if builder == nil {
		return
	}

	ctx := newReplyGuidanceContext(envelope)
	builder.WriteString("\n\n")
	writeGuidanceLine(
		builder,
		"Prefer `agh__network_send` when available; otherwise use `agh network send` to respond.",
	)
	writeGuidanceLine(builder, responseRegisterGuidanceLine(envelope))
	writeGuidanceLine(builder, ctx.replyFlagsLine())
	writeGuidanceLine(builder, ctx.causationLine())

	if traceLine := ctx.traceLine(); traceLine != "" {
		writeGuidanceLine(builder, traceLine)
	}
	if workLine := ctx.workLine(); workLine != "" {
		writeGuidanceLine(builder, workLine)
	}
	for _, line := range protocolGuidanceText {
		writeGuidanceLine(builder, line)
	}
	if guidanceMode == networkMessageGuidanceCompact {
		writeCompactReplyGuidance(builder)
		return
	}

	writeGuidanceLine(builder, "Examples:")
	writeGuidanceLine(builder, "```bash")
	ctx.writeReplyExample(builder)
	if ctx.reuseWork {
		ctx.writeProtocolReceiptExample(builder)
		ctx.writeProtocolTraceExample(builder)
		ctx.writeProtocolCapabilityExample(builder)
	}
	writeGuidanceLine(builder, "```")
	builder.WriteString("See `agh network --help` for options.")
}

func writeCompactReplyGuidance(builder *strings.Builder) {
	writeGuidanceLine(builder, compactReplyGuidanceText)
}

const (
	responseRegisterDirectGuidance = "Response register: direct replies are brief and actionable; " +
		"promote executable work to tasks."
	responseRegisterThreadGuidance = "Response register: in threads, reply briefly only when addressed, " +
		"activated, or adding value; promote executable work to tasks."
	responseRegisterChannelGuidance = "Response register: channel replies stay brief; respond only when addressed, " +
		"activated, or adding value."
)

func responseRegisterGuidanceLine(envelope Envelope) string {
	if envelope.Surface != nil {
		switch *envelope.Surface {
		case SurfaceDirect:
			return responseRegisterDirectGuidance
		case SurfaceThread:
			return responseRegisterThreadGuidance
		}
	}
	if envelope.DirectID != nil && strings.TrimSpace(*envelope.DirectID) != "" {
		return responseRegisterDirectGuidance
	}
	if envelope.ThreadID != nil && strings.TrimSpace(*envelope.ThreadID) != "" {
		return responseRegisterThreadGuidance
	}
	return responseRegisterChannelGuidance
}

type replyGuidanceContext struct {
	envelope  Envelope
	reuseWork bool
	workID    string
	traceID   string
}

func newReplyGuidanceContext(envelope Envelope) replyGuidanceContext {
	workID := lifecycleWorkID(envelope)
	ctx := replyGuidanceContext{
		envelope:  envelope,
		reuseWork: workID != "",
		workID:    workID,
	}
	if envelope.TraceID != nil {
		ctx.traceID = *envelope.TraceID
	}
	return ctx
}

func (c replyGuidanceContext) replyFlagsLine() string {
	var builder strings.Builder
	builder.WriteString("For replies to this message, keep `--session \"$AGH_SESSION_ID\"`,")
	builder.WriteString(" `--channel \"")
	builder.WriteString(c.envelope.Channel)
	builder.WriteString("\"`,")
	if c.envelope.Surface != nil {
		builder.WriteString(" `--surface \"")
		builder.WriteString(string(*c.envelope.Surface))
		builder.WriteString("\"`,")
	}
	if c.envelope.ThreadID != nil {
		builder.WriteString(" `--thread \"")
		builder.WriteString(*c.envelope.ThreadID)
		builder.WriteString("\"`,")
	}
	if c.envelope.DirectID != nil {
		builder.WriteString(" `--direct \"")
		builder.WriteString(*c.envelope.DirectID)
		builder.WriteString("\"`,")
	}
	builder.WriteString(" `--to \"")
	builder.WriteString(c.envelope.From)
	builder.WriteString("\"`")
	if c.reuseWork {
		builder.WriteString(", `--work \"")
		builder.WriteString(c.workID)
		builder.WriteString("\"`")
	}
	builder.WriteString(", and `--reply-to \"")
	builder.WriteString(c.envelope.ID)
	builder.WriteString("\"`.")
	return builder.String()
}

func (c replyGuidanceContext) causationLine() string {
	return "If this inbound message is the direct cause of your reply, set `--causation-id " +
		strconv.Quote(c.envelope.ID) + "` on the outbound message."
}

func (c replyGuidanceContext) traceLine() string {
	if c.traceID == "" {
		return ""
	}
	return "Preserve `--trace-id " + strconv.Quote(c.traceID) + "` on correlated follow-up messages."
}

func (c replyGuidanceContext) workLine() string {
	if c.workID == "" {
		return ""
	}
	return "Preserve `--work " + strconv.Quote(c.workID) + "` only while continuing the same lifecycle-bearing work."
}

func (c replyGuidanceContext) writeReplyExample(builder *strings.Builder) {
	writeGuidanceLine(builder, "# Reply")
	writeGuidanceLine(builder, "agh network send \\")
	writeGuidanceLine(builder, `  --session "$AGH_SESSION_ID" \`)
	writeQuotedFlagLine(builder, "  --channel ", c.envelope.Channel)
	c.writeSurfaceFlags(builder)
	writeGuidanceLine(builder, "  --kind say \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	if c.reuseWork {
		writeQuotedFlagLine(builder, "  --work ", c.workID)
	}

	writeQuotedFlagLine(builder, "  --reply-to ", c.envelope.ID)
	writeQuotedFlagLine(builder, "  --causation-id ", c.envelope.ID)
	c.writeTraceFlag(builder)
	writeGuidanceLine(builder, `  --body '{"text":"Reply text","intent":"reply"}' \`)
	writeGuidanceLine(builder, "  -o json")
}

func (c replyGuidanceContext) writeProtocolReceiptExample(builder *strings.Builder) {
	writeExampleSectionSeparator(builder)
	writeGuidanceLine(builder, "# Protocol receipt")
	writeGuidanceLine(builder, "agh network send \\")
	writeGuidanceLine(builder, `  --session "$AGH_SESSION_ID" \`)
	writeQuotedFlagLine(builder, "  --channel ", c.envelope.Channel)
	c.writeSurfaceFlags(builder)
	writeGuidanceLine(builder, "  --kind receipt \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	writeQuotedFlagLine(builder, "  --work ", c.workID)
	writeQuotedFlagLine(builder, "  --reply-to ", c.envelope.ID)
	writeQuotedFlagLine(builder, "  --causation-id ", c.envelope.ID)
	c.writeTraceFlag(builder)
	writeGuidanceLine(
		builder,
		`  --body '{"for_id":"`+c.envelope.ID+`","status":"accepted","detail":"Accepted for processing."}' \`,
	)
	writeGuidanceLine(builder, "  -o json")
}

func (c replyGuidanceContext) writeProtocolTraceExample(builder *strings.Builder) {
	writeExampleSectionSeparator(builder)
	writeGuidanceLine(builder, "# Protocol trace")
	writeGuidanceLine(builder, "agh network send \\")
	writeGuidanceLine(builder, `  --session "$AGH_SESSION_ID" \`)
	writeQuotedFlagLine(builder, "  --channel ", c.envelope.Channel)
	c.writeSurfaceFlags(builder)
	writeGuidanceLine(builder, "  --kind trace \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	writeQuotedFlagLine(builder, "  --work ", c.workID)
	writeQuotedFlagLine(builder, "  --reply-to ", c.envelope.ID)
	writeQuotedFlagLine(builder, "  --causation-id ", c.envelope.ID)
	c.writeTraceFlag(builder)
	writeGuidanceLine(builder, `  --body '{"state":"working","message":"Inspecting the request."}' \`)
	writeGuidanceLine(builder, "  -o json")
}

func (c replyGuidanceContext) writeProtocolCapabilityExample(builder *strings.Builder) {
	writeExampleSectionSeparator(builder)
	writeGuidanceLine(builder, "# Protocol capability")
	writeGuidanceLine(builder, "agh network send \\")
	writeGuidanceLine(builder, `  --session "$AGH_SESSION_ID" \`)
	writeQuotedFlagLine(builder, "  --channel ", c.envelope.Channel)
	c.writeSurfaceFlags(builder)
	writeGuidanceLine(builder, "  --kind capability \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	writeQuotedFlagLine(builder, "  --work ", c.workID)
	writeQuotedFlagLine(builder, "  --reply-to ", c.envelope.ID)
	writeQuotedFlagLine(builder, "  --causation-id ", c.envelope.ID)
	c.writeTraceFlag(builder)
	writeGuidanceLine(builder, capabilityBodyExample)
	writeGuidanceLine(builder, "  -o json")
}

func (c replyGuidanceContext) writeTraceFlag(builder *strings.Builder) {
	if c.traceID == "" {
		return
	}
	writeQuotedFlagLine(builder, "  --trace-id ", c.traceID)
}

func (c replyGuidanceContext) writeSurfaceFlags(builder *strings.Builder) {
	if c.envelope.Surface != nil {
		writeQuotedFlagLine(builder, "  --surface ", string(*c.envelope.Surface))
	}
	if c.envelope.ThreadID != nil {
		writeQuotedFlagLine(builder, "  --thread ", *c.envelope.ThreadID)
	}
	if c.envelope.DirectID != nil {
		writeQuotedFlagLine(builder, "  --direct ", *c.envelope.DirectID)
	}
}

func writeGuidanceLine(builder *strings.Builder, line string) {
	builder.WriteString(line)
	builder.WriteByte('\n')
}

func writeQuotedFlagLine(builder *strings.Builder, prefix string, value string) {
	builder.WriteString(prefix)
	builder.WriteString(strconv.Quote(value))
	builder.WriteString(" \\\n")
}

func writeExampleSectionSeparator(builder *strings.Builder) {
	builder.WriteByte('\n')
}

func shouldReuseInboundWork(envelope Envelope) bool {
	if envelope.WorkID == nil {
		return false
	}

	switch envelope.Kind {
	case KindSay, KindReceipt, KindTrace, KindCapability:
		return true
	default:
		return false
	}
}

func lifecycleWorkID(envelope Envelope) string {
	if !shouldReuseInboundWork(envelope) {
		return ""
	}
	return strings.TrimSpace(*envelope.WorkID)
}

func previewForBody(body Body) string {
	switch value := body.(type) {
	case GreetBody:
		return ResolveGreetSummary(value.PeerCard, value.Summary)
	case WhoisBody:
		if value.Type == WhoisTypeRequest {
			return strings.TrimSpace(value.Query)
		}
		return ""
	case SayBody:
		return strings.TrimSpace(value.Text)
	case CapabilityBody:
		if summary := strings.TrimSpace(value.Capability.Summary); summary != "" {
			return summary
		}
		if outcome := strings.TrimSpace(value.Capability.Outcome); outcome != "" {
			return outcome
		}
		return strings.TrimSpace(value.Capability.ID)
	case ReceiptBody:
		if value.Detail != nil {
			return strings.TrimSpace(*value.Detail)
		}
		return ""
	case TraceBody:
		return strings.TrimSpace(value.Message)
	default:
		return ""
	}
}

// PreviewTextForRawBody derives operator-facing preview text from one raw
// persisted message body. Invalid bodies return an empty preview.
func PreviewTextForRawBody(kind Kind, raw json.RawMessage) string {
	body, err := DecodeBody(kind, raw)
	if err != nil {
		return ""
	}
	return previewForBody(body)
}

func writeXMLAttr(builder *strings.Builder, key string, value string) {
	builder.WriteByte(' ')
	builder.WriteString(strings.TrimSpace(key))
	builder.WriteString(`="`)
	builder.WriteString(xmlEscape(value))
	builder.WriteByte('"')
}

func xmlEscape(value string) string {
	return xmlEscapeReplacer.Replace(strings.TrimSpace(value))
}

func cloneEnvelope(envelope Envelope) Envelope {
	return Envelope{
		Protocol:    envelope.Protocol,
		ID:          envelope.ID,
		WorkspaceID: envelope.WorkspaceID,
		Kind:        envelope.Kind,
		Channel:     envelope.Channel,
		Surface:     cloneSurfacePtr(envelope.Surface),
		ThreadID:    normalizeOptionalIdentifier(envelope.ThreadID),
		DirectID:    normalizeOptionalIdentifier(envelope.DirectID),
		From:        envelope.From,
		To:          normalizeOptionalIdentifier(envelope.To),
		Mentions:    normalizeEnvelopeMentions(envelope.Mentions),
		WorkID:      normalizeOptionalIdentifier(envelope.WorkID),
		ReplyTo:     normalizeOptionalIdentifier(envelope.ReplyTo),
		TraceID:     normalizeOptionalIdentifier(envelope.TraceID),
		CausationID: normalizeOptionalIdentifier(envelope.CausationID),
		TS:          envelope.TS,
		ExpiresAt:   cloneInt64Ptr(envelope.ExpiresAt),
		Body:        cloneRawMessage(envelope.Body),
		Proof:       cloneProof(envelope.Proof),
		Ext:         cloneExtensionMap(envelope.Ext),
	}
}

func cloneQueuedEnvelope(item queuedEnvelope) queuedEnvelope {
	return queuedEnvelope{
		Envelope:     cloneEnvelope(item.Envelope),
		PeerID:       item.PeerID,
		AcceptedAt:   item.AcceptedAt,
		DeliveryMode: item.DeliveryMode,
		PromptMode:   item.PromptMode,
		RetryAttempt: item.RetryAttempt,
		SessionToken: item.SessionToken,
	}
}
