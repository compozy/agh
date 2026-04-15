# GoClaw Health Checking & Heartbeat Patterns Analysis

## Executive Summary

GoClaw implements a sophisticated multi-layer health monitoring system with:

1. **Heartbeat Ticker** - Background polling service for agent periodic check-ins
2. **MCP Health Loop** - Connection health monitoring with exponential backoff reconnection
3. **Simple HTTP Health Endpoint** - Basic readiness probe for load balancers
4. **Event-driven Architecture** - Lifecycle events emitted for external monitoring

These patterns can significantly improve AGH's daemon health monitoring, particularly around:

- Background service lifecycle management
- Graceful degradation under failures
- Event-driven health visibility
- Vendor-agnostic health checking (e.g., MCP server connectivity)

---

## 1. Health Checking Patterns Overview

### 1.1 Heartbeat Ticker Pattern (Primary)

**Location**: `internal/heartbeat/ticker.go` (523 lines)

The heartbeat ticker is a **background polling loop** that:

- Polls a database for due heartbeats every 30 seconds
- Runs eligible agents through the agent loop with custom prompts
- Tracks execution status (running, completed, error, suppressed)
- Publishes events for external monitoring
- Supports manual wake triggers for immediate execution

**Key characteristics**:

```
- Polling interval: 30 seconds
- Minimum interval between runs: 5 minutes (config)
- Maximum summary truncation: 500 chars
- Supports exponential backoff on retry (1s, 2s, 4s...)
- Wake channel capacity: 16 (non-blocking)
```

**Lifecycle**:

```go
ticker := heartbeat.NewTicker(cfg)
ticker.SetOnEvent(func(e store.HeartbeatEvent) { /* handle */ })
ticker.Start()    // goroutine running t.loop()
// ... later ...
ticker.Stop()     // close stopCh, wait for WaitGroup
```

### 1.2 MCP Health Loop Pattern

**Location**: `internal/mcp/manager_connect.go` (265-309)

The MCP health loop monitors connection state of external MCP servers:

```go
// healthLoop periodically pings the MCP server
func (m *Manager) healthLoop(ctx context.Context, ss *serverState) {
    ticker := newHealthTicker()  // configurable interval
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := ss.client.Ping(ctx); err != nil {
                if isMethodNotFound(err) {
                    // Server doesn't implement Ping, assume OK
                    ss.connected.Store(true)
                    continue
                }
                ss.healthFailures++
                if failures >= healthFailThreshold {  // typically 3
                    ss.connected.Store(false)
                    m.tryReconnect(ctx, ss)
                }
            } else {
                ss.connected.Store(true)
                ss.healthFailures = 0  // reset
            }
        }
    }
}
```

**Resilience features**:

- Ping-based health checks with method-not-found tolerance
- Consecutive failure threshold before disconnection (not single failure)
- Exponential backoff reconnection (2s, 4s, 8s...)
- Atomic state updates via `sync/atomic` Store
- Per-server `lastErr` tracking for debugging

### 1.3 HTTP Health Endpoint

**Location**: `internal/gateway/server.go:369-373`

```go
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"status":"ok","protocol":%d}`, protocol.ProtocolVersion)
}
```

**Current features**:

- Simple status response
- Includes protocol version for client compatibility checks
- No database connectivity check (could be enhanced)
- No latency SLA tracking

---

## 2. Key Code Patterns Worth Adapting

### Pattern 2.1: Polling Loop with Dual Channels (Wake + Timer)

**Source**: `heartbeat/ticker.go:108-123`

```go
func (t *Ticker) loop() {
    defer t.wg.Done()
    ticker := time.NewTicker(pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-t.stopCh:
            return
        case <-ticker.C:
            t.runDueHeartbeats()
        case agentID := <-t.wakeCh:
            go t.runOneByAgentID(agentID)
        }
    }
}
```

**Why effective**:

1. Graceful shutdown via `stopCh`
2. Periodic polling (predictable resource use)
3. Event-driven wake (responsiveness)
4. Non-blocking wake channel (skip overflow)
5. Background goroutine per wake (parallelism)

**Adapter pattern for AGH**:

```go
type HealthChecker struct {
    checkCh   chan interface{}  // wake signal
    stopCh    chan struct{}
    pollTicker *time.Ticker
}

// In loop, select between:
// - <-pollTicker.C: scheduled checks
// - <-checkCh: immediate checks (non-blocking send)
// - <-stopCh: graceful shutdown
```

---

### Pattern 2.2: Smart Failure Thresholding

**Source**: `mcp/manager_connect.go:285-298`

```go
if err := ss.client.Ping(ctx); err != nil {
    ss.mu.Lock()
    ss.healthFailures++
    failures := ss.healthFailures
    ss.mu.Unlock()

    slog.Warn("mcp.server.health_failed", "server", ss.name,
        "error", err, "consecutive", failures)

    // Only mark disconnected after threshold
    if failures >= healthFailThreshold {
        ss.connected.Store(false)
        m.tryReconnect(ctx, ss)
    }
} else {
    // On success, reset counter
    ss.connected.Store(true)
    ss.mu.Lock()
    ss.healthFailures = 0  // IMPORTANT: reset
    ss.mu.Unlock()
}
```

**Why effective**:

- Transient errors (network glitches, timeouts) don't trigger cascades
- Single success resets failure counter (optimistic)
- Failure threshold is configurable
- Separate tracking of reconnection attempts

**Typical threshold**: 3 consecutive failures before action

---

### Pattern 2.3: Event-Driven Lifecycle Visibility

**Source**: `heartbeat/ticker.go:90-98, 202-205, 387-393`

```go
type HeartbeatEvent struct {
    Action   string    // "running", "completed", "error", "suppressed", "skipped"
    AgentID  string
    AgentKey string
    Status   string
    Error    string
    Reason   string    // for skipped
}

func (t *Ticker) emitEvent(event store.HeartbeatEvent) {
    if t.onEvent != nil {
        t.onEvent(event)
    }
}

// Usage
ticker.SetOnEvent(func(event store.HeartbeatEvent) {
    server.BroadcastEvent(*protocol.NewEvent(protocol.EventHeartbeat, event))
})
```

**Why effective**:

- External systems can subscribe to health state changes
- Actions emit events (not just state queries)
- Callback-based (no polling by observers)
- Structured data (type-safe in Go)

---

### Pattern 2.4: Graceful Shutdown with WaitGroup

**Source**: `heartbeat/ticker.go:75-87`

```go
type Ticker struct {
    // ...
    stopCh chan struct{}
    wg     sync.WaitGroup
}

func (t *Ticker) Start() {
    t.wg.Add(1)
    go t.loop()
    slog.Info("heartbeat ticker started")
}

func (t *Ticker) Stop() {
    close(t.stopCh)      // signal all goroutines
    t.wg.Wait()          // wait for completion
    slog.Info("heartbeat ticker stopped")
}

func (t *Ticker) loop() {
    defer t.wg.Done()  // signal completion
    // ...
}
```

**Why effective**:

- No resource leaks (guarantees cleanup)
- Close is idempotent (can call Stop multiple times safely if needed)
- Structured concurrency (clear ownership)
- All goroutines tracked

---

### Pattern 2.5: State Management with Atomic + Mutex

**Source**: `mcp/manager.go:67, manager_connect.go:277-306`

```go
type serverState struct {
    connected       atomic.Bool        // fast read path
    healthFailures  int                // protected by mu
    reconnAttempts  int                // protected by mu
    lastErr         string             // protected by mu
    mu              sync.Mutex
}

// Read path (fast)
if ss.connected.Load() {
    // ...
}

// Write path (contended)
ss.mu.Lock()
ss.healthFailures++
ss.lastErr = err.Error()
ss.mu.Unlock()
```

**Why effective**:

- `atomic.Bool` is lock-free for boolean state
- Mutex protects counters that change infrequently
- Avoids lock contention on hot reads
- Mixed approach balances performance + simplicity

---

### Pattern 2.6: Request Isolation with Context

**Source**: `heartbeat/ticker.go:163-178`

```go
// Resolve agent to get tenant scope + display key
agentKey := agentIDStr
ag, agErr := t.agents.GetByIDUnscoped(context.Background(), hb.AgentID)
// ...

// Inject agent's tenant into context
if ag.TenantID != uuid.Nil {
    ctx = store.WithTenantID(ctx, ag.TenantID)
} else {
    ctx = store.WithTenantID(ctx, store.MasterTenantID)
}

// All subsequent store operations use tenant-scoped context
files, err := t.agents.GetAgentContextFiles(ctx, agentID)
```

**Why effective**:

- Tenant isolation is automatic (not manual parameter threading)
- One context.WithValue call propagates scope to all stores
- Unscoped lookup for initial resolution (to get tenant)
- Scoped operations for data access

---

### Pattern 2.7: Suppression Signal Pattern

**Source**: `heartbeat/ticker.go:449-458`

```go
// Smart suppression: If response contains "HEARTBEAT_OK", suppress delivery
func processResponse(response string, _ int) (deliver bool, cleaned string) {
    const ackToken = "HEARTBEAT_OK"
    if strings.Contains(response, ackToken) {
        return false, ""  // suppressed
    }
    return true, response  // deliver
}

// Usage
deliver, cleaned := processResponse(result.Content, hb.AckMaxChars)
if !deliver {
    t.finishRun(ctx, hb, sessionKey, agentKey, "suppressed", "", ...)
    return
}
```

**Why effective**:

- Allows agent to signal "nothing to report" without empty responses
- Reduces notification noise
- Content-based signal (no side channel needed)
- Distinguishes "error" from "no news" states

**Adapted for AGH daemons**:

- Could use "HEALTH_OK" token in output to suppress non-critical noise
- Distinguishes "healthy idle" from "healthy with alerts"

---

## 3. How These Patterns Improve AGH Daemon Health

### 3.1 Recommended Adaptations

#### A. Implement a Health Ticker System

**For**: Long-running daemons (gateway, broker, worker pool)

```go
// internal/daemon/health_ticker.go
type HealthTicker struct {
    name       string
    checks     []HealthCheck  // interface-based
    pollInterval time.Duration
    onEvent    func(HealthEvent)
    stopCh     chan struct{}
    wg         sync.WaitGroup
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) error
}

// Built-in checks:
// - DatabaseConnectivity
// - ExternalServiceReachability (NATS, Redis, etc.)
// - DiskSpace
// - MemoryUsage
// - MessageQueueDepth
```

**Polling loop** (adapted from heartbeat):

```go
func (ht *HealthTicker) loop() {
    defer ht.wg.Done()
    ticker := time.NewTicker(ht.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ht.stopCh:
            return
        case <-ticker.C:
            ht.runChecks()
        }
    }
}
```

---

#### B. Multi-Layer Health State

Adapt the MCP failure-thresholding pattern:

```go
type ComponentHealth struct {
    Name              string
    Status            string      // "healthy", "degraded", "unhealthy"
    LastCheckTime     time.Time
    ConsecutiveFails  int
    FailThreshold     int         // e.g., 3
    LastError         string

    mu sync.Mutex
}

func (ch *ComponentHealth) RecordFailure(err error) {
    ch.mu.Lock()
    defer ch.mu.Unlock()

    ch.ConsecutiveFails++
    ch.LastError = err.Error()
    ch.LastCheckTime = time.Now()

    if ch.ConsecutiveFails >= ch.FailThreshold {
        ch.Status = "unhealthy"
    } else if ch.ConsecutiveFails > 0 {
        ch.Status = "degraded"
    }
}

func (ch *ComponentHealth) RecordSuccess() {
    ch.mu.Lock()
    defer ch.mu.Unlock()

    ch.Status = "healthy"
    ch.ConsecutiveFails = 0
    ch.LastError = ""
    ch.LastCheckTime = time.Now()
}
```

---

#### C. Event-Driven Health Visibility

```go
type HealthEvent struct {
    Timestamp    time.Time
    Component    string              // "gateway", "broker", "worker-pool"
    Event        string              // "check_started", "check_passed", "check_failed", "status_changed"
    PreviousStatus string            // for transitions
    CurrentStatus  string
    Details        map[string]interface{}
}

// Subscribers can:
// 1. Push to observability system (Prometheus, DataDog)
// 2. Broadcast to WebSocket clients
// 3. Log structured metrics
// 4. Trigger alerts on thresholds
```

---

#### D. HTTP Health Endpoints with Granularity

Enhance the basic `/health` endpoint:

```go
// GET /health - simple liveness probe (current)
// Response: {"status":"ok","protocol":1}

// GET /health/ready - readiness probe
// Response: {
//   "ready": true,
//   "components": {
//     "database": "healthy",
//     "broker": "healthy",
//     "mcp_servers": "degraded"
//   }
// }

// GET /health/detailed - debug endpoint (auth-gated)
// Response: {
//   "uptime_seconds": 3600,
//   "components": [
//     {
//       "name": "database",
//       "status": "healthy",
//       "last_check": "2025-04-15T10:30:00Z",
//       "latency_ms": 5
//     }
//   ]
// }
```

---

### 3.2 Specific Small Pieces to Adapt

#### Piece 1: Wake Channel Pattern

```go
// From heartbeat/ticker.go:70
wakeCh chan uuid.UUID, 16  // non-blocking, cap=16

// Sending (non-blocking)
select {
case t.wakeCh <- agentID:
default: // skip if full
}

// Benefit: Immediate health check trigger without blocking
```

**For AGH**: Use this in gateway to trigger daemon health checks immediately:

```go
healthTicker.Wake()  // immediate check instead of waiting for next poll
```

---

#### Piece 2: Skip Reason Tracking

```go
// From heartbeat/ticker.go:396-415
type HeartbeatRunLog struct {
    Status     string
    SkipReason *string  // e.g., "active_hours", "queue_busy", "empty_checklist"
    // ...
}

// Benefits:
// - Understand why checks were skipped
// - Distinguish transient (queue_busy) from permanent (empty_checklist) skips
// - Tune intervals based on skip patterns
```

**For AGH**: Track why health checks were skipped:

```go
type HealthCheckLog struct {
    ComponentName string
    Timestamp     time.Time
    Status        string     // "passed", "failed", "skipped"
    SkipReason    *string    // "degraded_mode", "rate_limited", etc.
    DurationMS    int
}
```

---

#### Piece 3: Exponential Backoff on Retry

```go
// From heartbeat/ticker.go:294-296
for attempt := range maxAttempts {
    // ... run ...
    if attempt < maxAttempts-1 {
        time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
        // 1s, 2s, 4s...
    }
}
```

**For AGH**: Use when retrying failed health checks:

```go
func (h *HealthCheck) RetryWithBackoff(maxAttempts int) error {
    for attempt := range maxAttempts {
        if err := h.Check(ctx); err == nil {
            return nil
        }
        if attempt < maxAttempts-1 {
            backoff := time.Duration(1<<uint(attempt)) * time.Second
            time.Sleep(backoff)
        }
    }
    return fmt.Errorf("check failed after %d attempts", maxAttempts)
}
```

---

#### Piece 4: Token-Based Signal Pattern

```go
// From heartbeat/ticker.go:449-458
// Response contains "HEARTBEAT_OK" → suppress delivery
const ackToken = "HEARTBEAT_OK"
if strings.Contains(response, ackToken) {
    return false, ""
}
```

**For AGH**: Use for daemon health reporting:

- Daemon outputs "HEALTH_OK" when no actionable issues
- Monitor system only notifies on non-OK responses
- Reduces alert fatigue

---

#### Piece 5: Structured Logging with Counters

```go
// From heartbeat/ticker.go:334-394
type HeartbeatEvent struct {
    Action       string
    AgentID      string
    Status       string
    Error        string
    RunCount     int
    SuppressCount int
}

// Logged every run with context
slog.Warn("heartbeat.insert_log_failed",
    "agent_id", agentIDStr,
    "run_count", hb.RunCount,
    "error", err)
```

**For AGH**: Track health check invocation counts:

```go
type HealthMetrics struct {
    ComponentName      string
    TotalChecks        int
    SuccessfulChecks   int
    FailedChecks       int
    SkippedChecks      int
    AvgDurationMS      float64
    LastFailureTime    *time.Time
}
```

---

## 4. Implementation Roadmap for AGH

### Phase 1: Health Ticker Infrastructure (Week 1)

- [ ] Copy ticker loop pattern from goclaw
- [ ] Define HealthCheck interface (pluggable checks)
- [ ] Implement graceful shutdown with WaitGroup
- [ ] Add event callback system

### Phase 2: Built-in Checks (Week 2)

- [ ] Database connectivity check
- [ ] NATS/message broker check
- [ ] Worker pool queue depth check
- [ ] Disk space monitoring

### Phase 3: Event Broadcasting (Week 2-3)

- [ ] Event structure (status, component, timestamp)
- [ ] HTTP event streaming for WebSocket clients
- [ ] Structured logging integration

### Phase 4: HTTP Endpoints (Week 3)

- [ ] `/health` - simple liveness (existing)
- [ ] `/health/ready` - component-level readiness
- [ ] `/health/detailed` - debug endpoint (auth-gated)

### Phase 5: Observability (Week 4)

- [ ] Prometheus metrics export
- [ ] Alert rules on health transitions
- [ ] Dashboard integration

---

## 5. Code Snippets to Copy

### 5.1 Graceful Shutdown Pattern

**Source**: `heartbeat/ticker.go:60-87`

```go
type Service struct {
    stopCh chan struct{}
    wg     sync.WaitGroup
}

func (s *Service) Start() {
    s.wg.Add(1)
    go s.loop()
}

func (s *Service) Stop() {
    close(s.stopCh)
    s.wg.Wait()
}

func (s *Service) loop() {
    defer s.wg.Done()
    // ...
    select {
    case <-s.stopCh:
        return
    // ...
    }
}
```

---

### 5.2 Health State with Atomic + Mutex

**Source**: `mcp/manager_connect.go:276-306`

```go
type ComponentState struct {
    connected      atomic.Bool
    healthFailures int          // protected by mu
    lastErr        string       // protected by mu
    mu             sync.Mutex
}

func (c *ComponentState) RecordFailure(err error) {
    c.mu.Lock()
    c.healthFailures++
    c.lastErr = err.Error()
    failures := c.healthFailures
    c.mu.Unlock()

    if failures >= 3 {
        c.connected.Store(false)
        // trigger reconnect
    }
}

func (c *ComponentState) RecordSuccess() {
    c.connected.Store(true)
    c.mu.Lock()
    c.healthFailures = 0
    c.lastErr = ""
    c.mu.Unlock()
}
```

---

### 5.3 Event Emission Pattern

**Source**: `heartbeat/ticker.go:90-98`

```go
type Service struct {
    onEvent func(Event)
}

func (s *Service) SetOnEvent(fn func(Event)) {
    s.onEvent = fn
}

func (s *Service) emitEvent(e Event) {
    if s.onEvent != nil {
        s.onEvent(e)
    }
}

// Caller
service.SetOnEvent(func(e Event) {
    // handle event: log, broadcast, metrics, etc.
})
```

---

## 6. Potential Pitfalls & Solutions

| Pitfall                        | GoClaw Solution                        | AGH Adaptation                                 |
| ------------------------------ | -------------------------------------- | ---------------------------------------------- |
| **Goroutine leak on shutdown** | WaitGroup + stopCh pattern             | Ensure all background tasks use same pattern   |
| **Transient network errors**   | Failure threshold (3 consecutive)      | Don't cascade on single ping failure           |
| **High-frequency polling**     | Configurable interval (5-30s)          | Set reasonable defaults per check type         |
| **Channel overflow**           | Non-blocking send with default         | Queue-aware wake signal capacity=16            |
| **Contention on state**        | Atomic bool + mutex mix                | Use atomic for booleans, mutex for counters    |
| **Alert fatigue**              | Token-based suppression (HEARTBEAT_OK) | Allow daemons to signal "no actionable issues" |
| **Lost context**               | Context injection with WithTenantID    | Propagate correlation IDs in health events     |

---

## 7. Measurable Improvements for AGH

### Current State (Hypothetical)

- No structured health monitoring
- Cascading failures on single service outage
- No visibility into daemon state transitions
- Binary alive/dead perception

### After Goclaw Patterns Adoption

- **Resilience**: 3-strike failure threshold reduces false positives
- **Visibility**: Event stream provides real-time state changes
- **Debuggability**: Skip reasons + structured logs aid troubleshooting
- **Graceful Degradation**: Component-level health allows partial operation
- **Operator Confidence**: Clear readiness signals for deployment automation

---

## 8. References

- **Heartbeat Ticker**: `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/internal/heartbeat/ticker.go` (523 lines)
- **MCP Health Loop**: `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/internal/mcp/manager_connect.go` (265-309)
- **Gateway Setup**: `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/gateway_heartbeat.go` (95 lines)
- **Doctor Diagnostics**: `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/doctor.go` (234 lines)
- **HTTP Health Endpoint**: `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/internal/gateway/server.go:369-373`

---

## Conclusion

GoClaw's health checking patterns are production-grade with proven resilience:

1. **Polling loop** with optional wake signals balances responsiveness + predictability
2. **Failure thresholding** prevents transient errors from cascading
3. **Event-driven visibility** enables comprehensive monitoring
4. **Graceful shutdown** with WaitGroup guarantees clean resource cleanup

These patterns directly address AGH daemon health monitoring gaps and can be incrementally adopted starting with the ticker infrastructure.
