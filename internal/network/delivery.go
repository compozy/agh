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
	"time"

	"github.com/pedronauck/agh/internal/acp"
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
		"Do not imitate protocol `receipt` or `trace` with `--kind direct` plus " +
			"`intent:\"receipt\"` or `intent:\"trace\"`. Use the real protocol kinds `receipt` and `trace`.",
	}
)

const capabilityBodyExample = `  --body '{"capability":{"id":"reply-workflow","summary":"Compact inline checklist.","outcome":"A reusable reply workflow.","version":"1.0.0","digest":"sha256:replace-me","execution_outline":["Inspect request","Draft response"],"requirements":["workspace-write"]}}' \`

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

type deliveryCoordinator struct {
	lifecycleCtx  context.Context
	prompter      deliveryPrompter
	maxQueueDepth int
	logger        *slog.Logger
	now           func() time.Time

	mu       sync.Mutex
	queues   map[string]*inboundQueue
	inFlight map[string]queuedEnvelope

	deliveries sync.Map
	wg         sync.WaitGroup

	onDelivered func(sessionID string, envelope Envelope, mode string, latency time.Duration)
}

type deliveryState struct {
	done chan struct{}
}

type inboundQueue struct {
	mu       sync.Mutex
	maxDepth int
	items    []queuedEnvelope
}

type enqueueResult struct {
	Depth        int
	DeliveryMode string
	Dropped      *Envelope
}

type queuedEnvelope struct {
	Envelope     Envelope
	AcceptedAt   time.Time
	DeliveryMode string
}

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
	hook func(sessionID string, envelope Envelope, mode string, latency time.Duration),
) deliveryOption {
	return func(coordinator *deliveryCoordinator) {
		coordinator.onDelivered = hook
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
		queues:   make(map[string]*inboundQueue),
		inFlight: make(map[string]queuedEnvelope),
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
	result := queue.enqueue(delivery.Envelope, c.now(), c.prompter.IsPrompting(sessionID))
	if result.Dropped != nil {
		c.logger.Warn(
			"network.message.queue_overflow",
			"session_id", sessionID,
			"dropped_envelope_id", result.Dropped.ID,
			"queue_depth", result.Depth,
		)
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

func (c *deliveryCoordinator) dropSession(sessionID string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.queues, strings.TrimSpace(sessionID))
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
	envelope := item.Envelope

	message, err := formatNetworkMessage(envelope)
	if err != nil {
		c.handleRenderFailure(sessionID, item, state, err)
		return false
	}

	events, err := c.prompter.PromptNetwork(c.lifecycleCtx, sessionID, message, promptNetworkMeta(envelope))
	if err != nil {
		c.handleDeliveryFailure(sessionID, item, state, err)
		return false
	}
	if !c.drainPromptEvents(events) {
		c.handleInterruptedDelivery(sessionID, item)
		return false
	}

	c.finishDeliveredMessage(sessionID, item)
	return true
}

func promptNetworkMeta(envelope Envelope) acp.PromptNetworkMeta {
	meta := acp.PromptNetworkMeta{
		MessageID: envelope.ID,
		Kind:      string(envelope.Kind),
		Channel:   envelope.Channel,
		From:      envelope.From,
	}
	if envelope.To != nil {
		meta.To = strings.TrimSpace(*envelope.To)
	}
	if envelope.InteractionID != nil {
		meta.InteractionID = strings.TrimSpace(*envelope.InteractionID)
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
	item queuedEnvelope,
	state *deliveryState,
	err error,
) {
	c.clearInFlight(sessionID)
	c.requeueFront(sessionID, item)
	c.logger.Warn(
		"network.message.render_failed",
		"session_id", sessionID,
		"envelope_id", item.Envelope.ID,
		"error", err,
	)
	if json.Valid(item.Envelope.Body) {
		c.retryAfterWorkerExit(sessionID, state)
	}
}

func (c *deliveryCoordinator) handleDeliveryFailure(
	sessionID string,
	item queuedEnvelope,
	state *deliveryState,
	err error,
) {
	c.clearInFlight(sessionID)
	c.requeueFront(sessionID, item)
	c.logger.Warn(
		"network.message.delivery_failed",
		"session_id", sessionID,
		"envelope_id", item.Envelope.ID,
		"error", err,
	)
	c.retryAfterWorkerExit(sessionID, state)
}

func (c *deliveryCoordinator) handleInterruptedDelivery(sessionID string, item queuedEnvelope) {
	c.clearInFlight(sessionID)
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

func (c *deliveryCoordinator) finishDeliveredMessage(sessionID string, item queuedEnvelope) {
	c.clearInFlight(sessionID)

	latency := max(c.now().Sub(item.AcceptedAt), 0)

	c.logger.Info(
		"network.message.delivered",
		"session_id", sessionID,
		"message_id", item.Envelope.ID,
		"kind", string(item.Envelope.Kind),
		"channel", item.Envelope.Channel,
		"delivery_mode", item.DeliveryMode,
		"latency_ms", latency.Milliseconds(),
	)
	if c.onDelivered != nil {
		c.onDelivered(sessionID, item.Envelope, item.DeliveryMode, latency)
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

func (c *deliveryCoordinator) requeueFront(sessionID string, item queuedEnvelope) {
	c.mu.Lock()
	queue := c.queues[strings.TrimSpace(sessionID)]
	c.mu.Unlock()
	if queue == nil {
		return
	}
	queue.prepend(item)
}

func (c *deliveryCoordinator) retryAfterWorkerExit(sessionID string, state *deliveryState) {
	if c == nil || state == nil {
		return
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return
	}

	go func() {
		select {
		case <-state.done:
		case <-c.lifecycleCtx.Done():
			return
		}

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
	}()
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
	return &inboundQueue{maxDepth: maxDepth}
}

func (q *inboundQueue) enqueue(envelope Envelope, acceptedAt time.Time, prompting bool) enqueueResult {
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
		AcceptedAt:   acceptedAt.UTC(),
		DeliveryMode: deliveryMode,
	})

	return enqueueResult{
		Depth:        len(q.items),
		DeliveryMode: deliveryMode,
		Dropped:      dropped,
	}
}

func (q *inboundQueue) prepend(item queuedEnvelope) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append([]queuedEnvelope{cloneQueuedEnvelope(item)}, q.items...)
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
	body, err := envelope.DecodeBody()
	preview := ""

	var canonicalBody []byte
	switch {
	case err == nil:
		canonicalBody, err = json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("network: marshal canonical body for delivery: %w", err)
		}
		preview = previewForBody(body)
	case !json.Valid(envelope.Body):
		return "", fmt.Errorf("network: decode envelope body for delivery: %w", err)
	default:
		var compact bytes.Buffer
		if compactErr := json.Compact(&compact, envelope.Body); compactErr != nil {
			return "", fmt.Errorf("network: compact raw envelope body for delivery: %w", compactErr)
		}
		canonicalBody = compact.Bytes()
	}
	encodedBody := base64.StdEncoding.EncodeToString(canonicalBody)

	var builder strings.Builder
	builder.Grow(base64.StdEncoding.EncodedLen(len(canonicalBody)) + len(preview)*6 + 2048)
	builder.WriteString("<network-message")
	writeXMLAttr(&builder, "id", envelope.ID)
	writeXMLAttr(&builder, "from", envelope.From)
	writeXMLAttr(&builder, "channel", envelope.Channel)
	writeXMLAttr(&builder, "kind", string(envelope.Kind))
	if envelope.To != nil {
		writeXMLAttr(&builder, "to", *envelope.To)
	}
	if envelope.InteractionID != nil {
		writeXMLAttr(&builder, "interaction", *envelope.InteractionID)
	}
	if envelope.ReplyTo != nil {
		writeXMLAttr(&builder, "reply-to", *envelope.ReplyTo)
	}
	if envelope.TraceID != nil {
		writeXMLAttr(&builder, "trace-id", *envelope.TraceID)
	}
	if envelope.CausationID != nil {
		writeXMLAttr(&builder, "causation-id", *envelope.CausationID)
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
	writeReplyGuidance(&builder, envelope)

	return builder.String(), nil
}

func writeReplyGuidance(builder *strings.Builder, envelope Envelope) {
	if builder == nil {
		return
	}

	ctx := newReplyGuidanceContext(envelope)
	builder.WriteString("\n\n")
	writeGuidanceLine(builder, "Use `agh network send` to respond through the audited CLI path.")
	writeGuidanceLine(builder, ctx.replyFlagsLine())
	writeGuidanceLine(builder, ctx.causationLine())

	if traceLine := ctx.traceLine(); traceLine != "" {
		writeGuidanceLine(builder, traceLine)
	}
	if interactionLine := ctx.broadcastInteractionLine(); interactionLine != "" {
		writeGuidanceLine(builder, interactionLine)
	}
	for _, line := range protocolGuidanceText {
		writeGuidanceLine(builder, line)
	}
	if sayLine := ctx.sayLifecycleLine(); sayLine != "" {
		writeGuidanceLine(builder, sayLine)
	}

	writeGuidanceLine(builder, "Examples:")
	writeGuidanceLine(builder, "```bash")
	ctx.writeDirectReplyExample(builder)
	if ctx.reuseInteraction {
		ctx.writeProtocolReceiptExample(builder)
		ctx.writeProtocolTraceExample(builder)
		ctx.writeProtocolCapabilityExample(builder)
	}
	writeGuidanceLine(builder, "```")
	builder.WriteString("See `agh network --help` for options.")
}

type replyGuidanceContext struct {
	envelope         Envelope
	reuseInteraction bool
	interactionID    string
	traceID          string
}

func newReplyGuidanceContext(envelope Envelope) replyGuidanceContext {
	ctx := replyGuidanceContext{
		envelope:         envelope,
		reuseInteraction: shouldReuseInboundInteraction(envelope) && envelope.InteractionID != nil,
	}
	if envelope.InteractionID != nil {
		ctx.interactionID = *envelope.InteractionID
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
	builder.WriteString(" `--to \"")
	builder.WriteString(c.envelope.From)
	builder.WriteString("\"`")
	if c.reuseInteraction {
		builder.WriteString(", `--interaction-id \"")
		builder.WriteString(c.interactionID)
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

func (c replyGuidanceContext) broadcastInteractionLine() string {
	if c.envelope.Kind != KindSay || c.interactionID == "" {
		return ""
	}
	return "If you reply to this broadcast `say` with `--kind direct`, choose a NEW `--interaction-id` unique to your targeted conversation instead of reusing `" +
		c.interactionID + "`."
}

func (c replyGuidanceContext) sayLifecycleLine() string {
	if c.envelope.Kind != KindSay {
		return ""
	}
	return "Do not send `receipt` or `trace` directly against this broadcast `say`. Open a new targeted `direct` interaction first if you need lifecycle messages."
}

func (c replyGuidanceContext) writeDirectReplyExample(builder *strings.Builder) {
	writeGuidanceLine(builder, "# Direct reply")
	writeGuidanceLine(builder, "agh network send \\")
	writeGuidanceLine(builder, `  --session "$AGH_SESSION_ID" \`)
	writeQuotedFlagLine(builder, "  --channel ", c.envelope.Channel)
	writeGuidanceLine(builder, "  --kind direct \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)

	switch {
	case c.reuseInteraction:
		writeQuotedFlagLine(builder, "  --interaction-id ", c.interactionID)
	case c.envelope.Kind == KindSay:
		writeQuotedFlagLine(builder, "  --interaction-id ", "int-my-peer-reply-"+c.envelope.ID)
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
	writeGuidanceLine(builder, "  --kind receipt \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	writeQuotedFlagLine(builder, "  --interaction-id ", c.interactionID)
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
	writeGuidanceLine(builder, "  --kind trace \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	writeQuotedFlagLine(builder, "  --interaction-id ", c.interactionID)
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
	writeGuidanceLine(builder, "  --kind capability \\")
	writeQuotedFlagLine(builder, "  --to ", c.envelope.From)
	writeQuotedFlagLine(builder, "  --interaction-id ", c.interactionID)
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

func shouldReuseInboundInteraction(envelope Envelope) bool {
	if envelope.InteractionID == nil {
		return false
	}

	switch envelope.Kind {
	case KindDirect, KindReceipt, KindTrace, KindCapability:
		return true
	default:
		return false
	}
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
		Protocol:      envelope.Protocol,
		ID:            envelope.ID,
		Kind:          envelope.Kind,
		Channel:       envelope.Channel,
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

func cloneQueuedEnvelope(item queuedEnvelope) queuedEnvelope {
	return queuedEnvelope{
		Envelope:     cloneEnvelope(item.Envelope),
		AcceptedAt:   item.AcceptedAt,
		DeliveryMode: item.DeliveryMode,
	}
}
