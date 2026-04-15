# GoClaw Observability/Tracing Patterns — Analysis for AGH

## Executive Summary

GoClaw implements a sophisticated observability system focused on **multi-tenant distributed tracing, event recording, and token accounting**. Key systems:

- **Collector-based span batching** with async flush cycles (5s intervals)
- **Domain event bus** with typed events, dedup, and retry
- **OpenTelemetry OTLP export** (build-tag gated)
- **Token counting** with BPE encoding and fallback heuristics
- **Cost calculation** including reasoning token splits

---

## 1. Collector-Driven Tracing (HIGH IMPACT)

**Source:** `internal/tracing/collector.go`

```go
type Collector struct {
    spanCh       chan store.SpanData         // 1000-item buffer
    spanUpdateCh chan spanUpdate             // deferred updates (two-phase)
    retryCh      chan pendingUpdate          // failed updates + retry
    dirtyTraces  map[uuid.UUID]struct{}      // traces needing aggregate update
    exporter     SpanExporter                // optional OTel export
    OnFlush      func([]uuid.UUID)           // callback for realtime events
    broadcastStatus StatusBroadcaster        // immediate status broadcast
}
```

Key patterns:

- **Non-blocking emit:** `EmitSpan()` drops if buffer full, logs warning
- **Two-phase tracing:** `EmitSpan()` (initial "running") + `EmitSpanUpdate()` (completion)
- **Detached context retry:** Uses `context.WithoutCancel()` to survive caller cancellation
- **Aggregate updates:** Dirty traces queued for batch re-aggregation on flush

### Detached Context Retry Pattern

```go
func (c *Collector) updateTraceWithRetry(ctx context.Context, traceID uuid.UUID, updates map[string]any) bool {
    detached := context.WithoutCancel(ctx)
    backoffs := []time.Duration{100*time.Millisecond, 200*time.Millisecond, 300*time.Millisecond}
    for attempt := 0; attempt <= len(backoffs); attempt++ {
        opCtx, cancel := context.WithTimeout(detached, 5*time.Second)
        err := c.store.UpdateTrace(opCtx, traceID, updates)
        cancel()
        if err == nil { return true }
        if attempt < len(backoffs) { time.Sleep(backoffs[attempt]) }
    }
    c.enqueueRetry(ctx, traceID, updates)
    return false
}
```

**AGH gap:** Observer writes synchronously per-event, no buffering or retry.

---

## 2. Domain Event Bus with Worker Pool (MEDIUM IMPACT)

**Source:** `internal/eventbus/`

### Event Type Taxonomy

```go
const (
    EventSessionCompleted EventType = "session.completed"
    EventEpisodicCreated  EventType = "episodic.created"
    EventContextPruned    EventType = "context.pruned"
    EventDelegateSent     EventType = "delegate.sent"
    EventDelegateCompleted EventType = "delegate.completed"
)
```

Each event type has a **typed payload struct** ensuring compile-time safety.

### Worker Pool with Dedup & Retry

```go
type busImpl struct {
    queue    chan DomainEvent
    handlers map[EventType][]DomainEventHandler
    dedup    *dedupSet  // SourceID-based dedup
}

func (b *busImpl) Publish(event DomainEvent) {
    select {
    case b.queue <- event:
    default:
        slog.Warn("eventbus: queue full, dropping event")
    }
}
```

### Dedup Set with TTL Cleanup

```go
type dedupSet struct {
    mu   sync.Mutex
    seen map[string]time.Time  // sourceID -> expiry
    ttl  time.Duration
    stop chan struct{}
}

func (d *dedupSet) Add(sourceID string) bool {
    if sourceID == "" { return true }
    d.mu.Lock()
    defer d.mu.Unlock()
    if _, exists := d.seen[sourceID]; exists {
        return false
    }
    d.seen[sourceID] = time.Now().Add(d.ttl)
    return true
}
```

Worker config: QueueSize=1000, WorkerCount=2, RetryAttempts=3, RetryDelay=1s (exponential), DedupTTL=5min.

---

## 3. Token Counting with BPE & Fallback (HIGH IMPACT)

**Source:** `internal/tokencount/`

### Interface

```go
type TokenCounter interface {
    Count(model string, text string) int
    CountMessages(model string, msgs []providers.Message) int
    ModelContextWindow(model string) int
}
```

### Model Registry with Longest-Prefix Matching

```go
var DefaultRegistry = map[string]ModelInfo{
    "claude-":   {TokenizerCL100K, 200_000},
    "gpt-4o":    {TokenizerO200K, 128_000},
    "gpt-5":     {TokenizerO200K, 1_000_000},
}

func resolveModelInfo(model string) ModelInfo {
    var best string
    for prefix := range DefaultRegistry {
        if len(prefix) > len(best) && strings.HasPrefix(model, prefix) {
            best = prefix
        }
    }
    if best != "" { return DefaultRegistry[best] }
    return ModelInfo{TokenizerID: TokenizerFallback, ContextWindow: 200_000}
}
```

### Per-Message Cache with FNV Hash

```go
func messageHash(m providers.Message) uint64 {
    h := fnv.New64a()
    h.Write([]byte(m.Role))
    h.Write([]byte{0})
    h.Write([]byte(m.Content))
    for _, tc := range m.ToolCalls {
        h.Write([]byte{0})
        h.Write([]byte(tc.ID + tc.Name))
    }
    return h.Sum64()
}
```

### Cost Calculation with Reasoning Token Split

```go
func CalculateCost(pricing *config.ModelPricing, usage *providers.Usage) float64 {
    cost := float64(usage.PromptTokens) * pricing.InputPerMillion / 1_000_000
    if pricing.ReasoningPerMillion > 0 && usage.ThinkingTokens > 0 {
        visible := max(usage.CompletionTokens-usage.ThinkingTokens, 0)
        cost += float64(visible) * pricing.OutputPerMillion / 1_000_000
        cost += float64(usage.ThinkingTokens) * pricing.ReasoningPerMillion / 1_000_000
    } else {
        cost += float64(usage.CompletionTokens) * pricing.OutputPerMillion / 1_000_000
    }
    // Cache read/create costs...
    return cost
}
```

**Key:** ThinkingTokens are a SUB-COUNT of CompletionTokens — only split when `ReasoningPerMillion > 0`.

---

## 4. OpenTelemetry Integration (Build-Tag Gated)

**Source:** `cmd/gateway_otel.go`, `internal/tracing/otelexport/`

```go
//go:build otel

func initOTelExporter(ctx context.Context, cfg *config.Config, collector *tracing.Collector) {
    if !cfg.Telemetry.Enabled || cfg.Telemetry.Endpoint == "" { return }
    otelExp, _ := otelexport.New(ctx, otelexport.Config{
        Endpoint:    cfg.Telemetry.Endpoint,
        Protocol:    cfg.Telemetry.Protocol,  // "grpc" or "http"
        ServiceName: cfg.Telemetry.ServiceName,
    })
    collector.SetExporter(otelExp)
}
```

Uses `gen_ai.*` attributes following OpenTelemetry GenAI semantic conventions.

---

## 5. Utility Patterns

### String Truncation with Mid-Removal

```go
func TruncateMid(s string, maxLen int) string {
    s = strings.ToValidUTF8(s, "")
    if len(s) <= maxLen { return s }
    marker := fmt.Sprintf(truncateMarker, len(s)-maxLen)
    usable := maxLen - len(marker)
    head := usable * 2 / 3  // 2/3 head, 1/3 tail
    tail := usable - head
    // Align to rune boundaries...
    return s[:head] + marker + s[tailStart:]
}
```

### JSON Array Truncation (Binary Search)

Keeps first + last elements of message arrays, shows placeholder for omitted middle.

---

## GoClaw vs AGH Comparison

| Aspect         | GoClaw                              | AGH                   | Gap                        |
| -------------- | ----------------------------------- | --------------------- | -------------------------- |
| Buffering      | Batch spans, flush 5s               | Synchronous per-event | No async path              |
| Context        | Trace + Span hierarchy              | Session-scoped events | Flat, no hierarchy         |
| Export         | OTel OTLP (optional)                | Registry interface    | No external export         |
| Retry          | Exponential backoff (10 attempts)   | None                  | No resilience              |
| Dedup          | SourceID-based                      | None                  | Potential duplicates       |
| Token counting | BPE + cache + fallback              | None                  | No context budget tracking |
| Cost calc      | Per-model pricing + reasoning split | None                  | No cost visibility         |

---

## Recommended Adaptations for AGH

### Phase 1: Token Counting (QUICK WIN)

- Add `internal/tokencount` with BPE support
- Integrate `ModelContextWindow()` into context pruning
- Add cache invalidation on message compaction

### Phase 2: Event Dedup & Retry

- Add dedup set for session events (SourceID-based, 5min TTL)
- Add exponential backoff retry for critical updates (session completion)
- Use `context.WithoutCancel()` for must-complete operations

### Phase 3: Async Event Processing

- Extract expensive operations from synchronous path
- Implement event worker pool with bounded queue (256-1000)
- Add metrics for queue depth

### Phase 4: OTel Integration (Optional)

- Create `internal/tracing/otelexport` with build-tag gating
- Map session events to OTel spans using `gen_ai.*` semantic conventions
- Support optional OTLP endpoint via TOML config
