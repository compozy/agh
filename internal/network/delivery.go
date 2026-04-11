package network

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
)

type deliveryPrompter interface {
	PromptNetwork(ctx context.Context, sessionID string, message string) (<-chan acp.AgentEvent, error)
	IsPrompting(sessionID string) bool
}

type deliveryOption func(*deliveryCoordinator)

type deliveryCoordinator struct {
	lifecycleCtx  context.Context
	prompter      deliveryPrompter
	maxQueueDepth int
	logger        *slog.Logger
	now           func() time.Time

	mu     sync.Mutex
	queues map[string]*inboundQueue

	deliveries sync.Map
	wg         sync.WaitGroup
}

type deliveryState struct {
	done chan struct{}
}

type inboundQueue struct {
	mu       sync.Mutex
	maxDepth int
	items    []Envelope
}

type enqueueResult struct {
	Depth   int
	Dropped *Envelope
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
		lifecycleCtx:  ctx,
		prompter:      prompter,
		maxQueueDepth: maxQueueDepth,
		logger:        slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
		queues: make(map[string]*inboundQueue),
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
	result := queue.enqueue(delivery.Envelope)
	if result.Dropped != nil {
		c.logger.Warn(
			"network.message.queue_overflow",
			"session_id", sessionID,
			"dropped_envelope_id", result.Dropped.ID,
			"queue_depth", result.Depth,
		)
	}

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

	queue := newInboundQueue(c.maxQueueDepth)
	c.queues[target] = queue
	return queue
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

		envelope, ok := c.dequeue(target)
		if !ok {
			return
		}

		message, err := formatNetworkMessage(envelope)
		if err != nil {
			c.requeueFront(target, envelope)
			c.logger.Warn(
				"network.message.render_failed",
				"session_id", target,
				"envelope_id", envelope.ID,
				"error", err,
			)
			return
		}

		events, err := c.prompter.PromptNetwork(c.lifecycleCtx, target, message)
		if err != nil {
			c.requeueFront(target, envelope)
			c.logger.Warn(
				"network.message.delivery_failed",
				"session_id", target,
				"envelope_id", envelope.ID,
				"error", err,
			)
			return
		}

		c.drainPromptEvents(events)
	}
}

func (c *deliveryCoordinator) drainPromptEvents(events <-chan acp.AgentEvent) {
	if events == nil {
		return
	}

	for {
		select {
		case <-c.lifecycleCtx.Done():
			return
		case _, ok := <-events:
			if !ok {
				return
			}
		}
	}
}

func (c *deliveryCoordinator) dequeue(sessionID string) (Envelope, bool) {
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil {
		return Envelope{}, false
	}
	return queue.dequeue()
}

func (c *deliveryCoordinator) requeueFront(sessionID string, envelope Envelope) {
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil {
		return
	}
	queue.prepend(envelope)
}

func newInboundQueue(maxDepth int) *inboundQueue {
	return &inboundQueue{maxDepth: maxDepth}
}

func (q *inboundQueue) enqueue(envelope Envelope) enqueueResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	var dropped *Envelope
	if len(q.items) >= q.maxDepth {
		evicted := cloneEnvelope(q.items[0])
		dropped = &evicted
		copy(q.items[0:], q.items[1:])
		q.items = q.items[:len(q.items)-1]
	}
	q.items = append(q.items, cloneEnvelope(envelope))

	return enqueueResult{
		Depth:   len(q.items),
		Dropped: dropped,
	}
}

func (q *inboundQueue) prepend(envelope Envelope) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append([]Envelope{cloneEnvelope(envelope)}, q.items...)
}

func (q *inboundQueue) dequeue() (Envelope, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return Envelope{}, false
	}

	envelope := cloneEnvelope(q.items[0])
	copy(q.items[0:], q.items[1:])
	q.items = q.items[:len(q.items)-1]
	return envelope, true
}

func (q *inboundQueue) snapshot() []Envelope {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	out := make([]Envelope, 0, len(q.items))
	for _, envelope := range q.items {
		out = append(out, cloneEnvelope(envelope))
	}
	return out
}

func (q *inboundQueue) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func formatNetworkMessage(envelope Envelope) (string, error) {
	body, err := envelope.DecodeBody()
	if err != nil {
		return "", fmt.Errorf("network: decode envelope body for delivery: %w", err)
	}

	canonicalBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("network: marshal canonical body for delivery: %w", err)
	}
	encodedBody := base64.StdEncoding.EncodeToString(canonicalBody)

	preview := previewForBody(body)

	var builder strings.Builder
	builder.WriteString("<network-message")
	writeXMLAttr(&builder, "id", envelope.ID)
	writeXMLAttr(&builder, "from", envelope.From)
	writeXMLAttr(&builder, "space", envelope.Space)
	writeXMLAttr(&builder, "kind", string(envelope.Kind))
	if envelope.InteractionID != nil {
		writeXMLAttr(&builder, "interaction", *envelope.InteractionID)
	}
	if envelope.ExpiresAt != nil {
		writeXMLAttr(&builder, "expires-at", strconv.FormatInt(*envelope.ExpiresAt, 10))
	}
	writeXMLAttr(&builder, "trust", "untrusted")
	builder.WriteString(">\n")
	if preview != "" {
		builder.WriteString("  <network-preview encoding=\"xml-escaped\">")
		builder.WriteString(xmlEscape(preview))
		builder.WriteString("</network-preview>\n")
	}
	builder.WriteString("  <network-body encoding=\"base64-json\">")
	builder.WriteString(encodedBody)
	builder.WriteString("</network-body>\n")
	builder.WriteString("</network-message>")

	return builder.String(), nil
}

func previewForBody(body Body) string {
	switch value := body.(type) {
	case GreetBody:
		return strings.TrimSpace(value.Summary)
	case WhoisBody:
		if value.Type == WhoisTypeRequest {
			return strings.TrimSpace(value.Query)
		}
		return ""
	case SayBody:
		return strings.TrimSpace(value.Text)
	case DirectBody:
		return strings.TrimSpace(value.Text)
	case RecipeBody:
		if summary := strings.TrimSpace(value.Recipe.Summary); summary != "" {
			return summary
		}
		return strings.TrimSpace(value.Recipe.Title)
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

func writeXMLAttr(builder *strings.Builder, key string, value string) {
	builder.WriteByte(' ')
	builder.WriteString(strings.TrimSpace(key))
	builder.WriteString(`="`)
	builder.WriteString(xmlEscape(value))
	builder.WriteByte('"')
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

func cloneEnvelope(envelope Envelope) Envelope {
	return Envelope{
		Protocol:      envelope.Protocol,
		ID:            envelope.ID,
		Kind:          envelope.Kind,
		Space:         envelope.Space,
		From:          envelope.From,
		To:            normalizeOptionalIdentifier(envelope.To),
		InteractionID: normalizeOptionalIdentifier(envelope.InteractionID),
		ReplyTo:       normalizeOptionalIdentifier(envelope.ReplyTo),
		TraceID:       normalizeOptionalIdentifier(envelope.TraceID),
		CausationID:   normalizeOptionalIdentifier(envelope.CausationID),
		TS:            envelope.TS,
		ExpiresAt:     cloneInt64Ptr(envelope.ExpiresAt),
		Body:          cloneRawMessage(envelope.Body),
		Proof:         cloneProof(envelope.Proof),
		Ext:           cloneExtensionMap(envelope.Ext),
	}
}
