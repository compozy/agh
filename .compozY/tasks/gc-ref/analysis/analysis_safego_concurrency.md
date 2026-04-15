# GoClaw Concurrency Patterns Analysis

## Executive Summary

This analysis examines the GoClaw codebase's concurrency safety patterns, focusing on the `internal/safego/` package and broader goroutine management across the system. GoClaw uses a **multi-layered approach** to safely manage concurrent operations:

1. **Panic recovery wrapper** (`safego.Recover`)
2. **Lane-based concurrency control** (scheduler)
3. **Domain event bus** with worker pool and dedup
4. **Context-driven cancellation** for graceful shutdown
5. **WaitGroup coordination** for goroutine lifecycle
6. **Atomic counters** for lock-free metrics

These patterns are directly applicable to AGH and suitable for selective adoption without importing external dependencies.

---

## 1. The `safego` Package

### 1.1 Core Pattern: `safego.Recover()`

**Location:** `.resources/goclaw/internal/safego/recover.go`

```go
// Recover catches panics, logs an error with stack trace, and optionally
// invokes onPanic. Must be called via defer:
//
//	defer safego.Recover(nil, "job_id", id)              // log-only
//	defer safego.Recover(func(v any) { ... }, "tool", n) // log + callback
func Recover(onPanic func(v any), attrs ...any) {
	r := recover()
	if r == nil {
		return
	}
	buf := make([]byte, 8192)
	n := runtime.Stack(buf, false)
	slog.Error("goroutine panicked",
		append(attrs, "panic", fmt.Sprint(r), "stack", string(buf[:n]))...,
	)
	if onPanic != nil {
		onPanic(r)
	}
}
```

**Design Philosophy:**

- Single function, ~30 lines of code
- No external dependencies
- Variadic `attrs` for structured logging context
- Optional callback for custom panic handling
- Full stack trace captured (8KB buffer)
- Logs to `slog.Error()` (structured logging)

**Key Strengths:**

1. **Minimal surface area** — doesn't wrap goroutines, just catches panics
2. **Flexible context** — arbitrary `attrs` for identifying the goroutine
3. **Composable** — pairs with `defer` and `sync.WaitGroup.Done()`
4. **No allocation on success path** — early return if no panic

---

### 1.2 Usage Patterns Across GoClaw

**Pattern 1: Channel Event Loops**

```go
// internal/channels/slack/channel.go:166-170
go func() {
	defer c.wg.Done()
	defer safego.Recover(nil, "component", "slack_event_loop")
	c.eventLoop(smCtx)
}()
```

**Pattern 2: Concurrent Tasks with Callbacks**

```go
// internal/safego/recover_test.go:21-30
var captured string
done := make(chan struct{})
go func() {
	defer close(done)
	defer Recover(func(v any) {
		captured = v.(string)
	}, "test", "callback")
	panic("caught me")
}()
<-done
```

**Pattern 3: Background Workers**

```go
// internal/agent/loop_history_sanitize.go:231-234
go func() {
	defer sessionMu.Unlock()
	defer safego.Recover(nil, "session", sessionKey)
	// ... background work
}()
```

**Observed Usage Across Codebase:**

- 9 files use `safego.Recover` directly
- Common in: Slack, Feishu, agent loops, channel event loops
- Never nested or wrapped — always direct `defer`
- Always paired with `wg.Done()` or `close(ch)`

---

## 2. Goroutine Lifecycle Patterns

### 2.1 WaitGroup + Context Pattern

GoClaw consistently uses:

```go
type Component struct {
    wg       sync.WaitGroup
    ctx      context.Context
    cancel   context.CancelFunc
}

// Start a worker
func (c *Component) start() {
    c.wg.Add(1)
    go func() {
        defer c.wg.Done()
        defer safego.Recover(nil, "component", c.name)

        // Listen for cancellation
        select {
        case <-c.ctx.Done():
            return
        // ... other cases
        }
    }()
}

// Graceful shutdown
func (c *Component) Stop() {
    c.cancel()       // Signal cancellation
    c.wg.Wait()      // Wait for workers to finish
}
```

**Evidence:**

- Slack channel: `wg sync.WaitGroup` + `cancelFn context.CancelFunc` (line 54-55)
- Lane scheduler: `wg sync.WaitGroup`, `ctx context.Context`, `cancel context.CancelFunc` (lines 46)
- Event bus: `wg sync.WaitGroup`, `ctx context.Context`, `cancel context.CancelFunc` (lines 19-21)

### 2.2 Semaphore-Based Lane Pattern

**Location:** `internal/scheduler/lanes.go`

The lane pattern uses **buffered channels as semaphores** for concurrency control:

```go
type Lane struct {
    name        string
    concurrency int
    sem         chan struct{} // semaphore tokens
    pending     atomic.Int64  // pending requests count
    active      atomic.Int64  // active (running) requests count
    ctx         context.Context
    cancel      context.CancelFunc
    wg          sync.WaitGroup
}

// Create lane with concurrency limit
func NewLane(name string, concurrency int) *Lane {
    ctx, cancel := context.WithCancel(context.Background())
    l := &Lane{
        name:        name,
        concurrency: concurrency,
        sem:         make(chan struct{}, concurrency),
        ctx:         ctx,
        cancel:      cancel,
    }
    // Pre-fill semaphore
    for i := 0; i < concurrency; i++ {
        l.sem <- struct{}{}
    }
    return l
}

// Submit work with bounded concurrency
func (l *Lane) Submit(ctx context.Context, fn func()) error {
    l.pending.Add(1)
    defer l.pending.Add(-1)

    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-l.ctx.Done():
        return context.Canceled
    case token, ok := <-l.sem:
        if !ok {
            return context.Canceled
        }

        l.active.Add(1)
        l.wg.Add(1)

        go func() {
            defer func() {
                l.active.Add(-1)
                l.wg.Done()
                l.sem <- token // return token
            }()
            fn()
        }()
        return nil
    }
}
```

**Strengths:**

1. **Lock-free** — uses channels, not mutexes
2. **Observability** — `pending` and `active` atomic counters
3. **Context-aware** — respects parent and lane cancellation
4. **Pre-filled tokens** — zero allocation for common case
5. **Graceful shutdown** — cancel signals goroutines to exit

---

## 3. Event Bus Pattern

### 3.1 Domain Event Bus with Worker Pool

**Location:** `internal/eventbus/bus_impl.go`

```go
type busImpl struct {
    cfg      Config
    queue    chan DomainEvent
    handlers map[EventType][]DomainEventHandler
    mu       sync.RWMutex
    dedup    *dedupSet
    wg       sync.WaitGroup
    ctx      context.Context
    cancel   context.CancelFunc
    started  atomic.Bool
    draining atomic.Bool
}

// Start creates worker pool
func (b *busImpl) Start(ctx context.Context) {
    if b.started.Swap(true) {
        return // already started
    }
    b.ctx, b.cancel = context.WithCancel(ctx)
    for range b.cfg.WorkerCount {
        b.wg.Add(1)
        go b.worker()
    }
}

// Worker loop
func (b *busImpl) worker() {
    defer b.wg.Done()
    for event := range b.queue {
        if b.ctx.Err() != nil {
            return
        }
        b.dispatch(event)
    }
}

// Graceful drain with timeout
func (b *busImpl) Drain(timeout time.Duration) error {
    b.draining.Store(true)
    close(b.queue)

    done := make(chan struct{})
    go func() {
        b.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        b.dedup.Close()
        return nil
    case <-time.After(timeout):
        b.cancel()
        b.dedup.Close()
        return fmt.Errorf("eventbus: drain timeout after %v", timeout)
    }
}

// Safe handler invocation with panic recovery
func (b *busImpl) safeCall(handler DomainEventHandler, event DomainEvent) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("eventbus: handler panic: %v", r)
            slog.Error("eventbus: handler panic", "type", event.Type, "panic", r)
        }
    }()
    return handler(b.ctx, event)
}
```

**Key Features:**

1. **Worker pool** — configurable worker count
2. **Dedup** — prevents duplicate event processing
3. **Retry with exponential backoff** — retries on handler error
4. **Built-in panic recovery** — custom `defer` + `recover()` in `safeCall()`
5. **Graceful drain** — stops accepting, waits for queue drain or timeout
6. **Atomic state** — `started` and `draining` flags prevent races

---

### 3.2 Drain Pattern with Timeout

This is a **critical pattern** for graceful shutdown:

```go
// Drain blocks until queue is empty OR timeout expires
func (b *busImpl) Drain(timeout time.Duration) error {
    b.draining.Store(true)    // Stop accepting new events
    close(b.queue)            // Signal workers to exit when queue empty

    done := make(chan struct{})
    go func() {
        b.wg.Wait()           // Wait for all workers
        close(done)
    }()

    select {
    case <-done:
        return nil            // Clean shutdown
    case <-time.After(timeout):
        b.cancel()            // Force cancel on timeout
        return fmt.Errorf("drain timeout after %v", timeout)
    }
}
```

**Application to AGH:**

- Ensures graceful shutdown with upper time bound
- Prevents hanging on slow consumers
- Force-cancels if timeout expires

---

## 4. Graceful Shutdown Architecture

### 4.1 Signal Handling + Context Cancellation

**Location:** `cmd/gateway.go:456-462` + `cmd/gateway_lifecycle.go:142-150`

```go
// Setup graceful shutdown
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

// Signal handler goroutine
go func() {
    sig := <-sigCh
    slog.Info("graceful shutdown initiated", "signal", sig)

    // Broadcast shutdown event
    d.server.BroadcastEvent(*protocol.NewEvent(protocol.EventShutdown, nil))

    // Stop channels, cron, heartbeat
    d.channelMgr.StopAll(context.Background())
    d.pgStores.Cron.Stop()
    deps.heartbeatTicker.Stop()

    // Drain scheduler
    deps.sched.Stop()

    // Drain domain event bus with 10s timeout
    if err := domainBus.Drain(10 * time.Second); err != nil {
        slog.Warn("domain event bus drain timeout", "error", err)
    }

    // Finally cancel all child contexts
    cancel()
}()
```

**Shutdown Sequence:**

1. Receive `SIGINT`/`SIGTERM`
2. Broadcast shutdown event (notify clients)
3. Stop accepting new work (channels, cron, heartbeat)
4. Drain queues with timeouts
5. Cancel root context (forces goroutines to exit)
6. Wait for all goroutines via `wg.Wait()` in component `.Stop()` methods

---

## 5. Key Patterns Summary

| Pattern                | Purpose                  | GoClaw Location                 | Recommendation for AGH            |
| ---------------------- | ------------------------ | ------------------------------- | --------------------------------- |
| `safego.Recover()`     | Panic logging + callback | `internal/safego/recover.go`    | **ADOPT** — Direct copy, no deps  |
| Lane semaphore         | Bounded concurrency      | `internal/scheduler/lanes.go`   | **ADOPT** — Useful for work pools |
| Event bus drain        | Graceful queue shutdown  | `internal/eventbus/bus_impl.go` | **ADAPT** — Modify for AGH events |
| Context + Cancel       | Cancellation signaling   | Throughout codebase             | **USE** — Already in stdlib       |
| WaitGroup + defer Done | Goroutine lifecycle      | Standard Go pattern             | **USE** — Already in stdlib       |

---

## 6. Practical Recommendations for AGH

### 6.1 Immediate Adoption: `safego.Recover`

**Cost:** ~30 lines, zero external dependencies  
**Benefit:** Crash-safe goroutines across entire codebase  
**Implementation:**

```go
// agh/internal/safego/recover.go
package safego

import (
	"fmt"
	"log/slog"
	"runtime"
)

// Recover catches panics and logs with stack trace.
// Call via defer at goroutine entry.
func Recover(onPanic func(v any), attrs ...any) {
	r := recover()
	if r == nil {
		return
	}
	buf := make([]byte, 8192)
	n := runtime.Stack(buf, false)
	slog.Error("goroutine panicked",
		append(attrs, "panic", fmt.Sprint(r), "stack", string(buf[:n]))...,
	)
	if onPanic != nil {
		onPanic(r)
	}
}
```

**Usage in AGH:**

```go
// Any goroutine spawned during agent execution
go func() {
    defer wg.Done()
    defer safego.Recover(nil, "agent", agentID, "stage", "memory_write")
    // ... work
}()
```

---

### 6.2 Adapt: Lane-Based Scheduler

**Cost:** ~240 lines (copy `internal/scheduler/lanes.go`)  
**Benefit:** Bounded concurrency, work distribution, observability  
**When to use:** If AGH needs to limit concurrent agent runs or tool invocations

**Minimal adaptation:**

- Keep semaphore token pattern (no mutexes)
- Add `pending` + `active` atomic counters for metrics
- Keep context + cancel for shutdown

---

### 6.3 Adapt: Domain Event Bus

**Cost:** ~150 lines (copy `internal/eventbus/bus_impl.go`)  
**Benefit:** Decoupled event handling, worker pool, dedup, retry  
**When to use:** If AGH has multiple event sources (agent transitions, tool completion, memory updates)

**Required changes:**

- Change `DomainEvent` struct to match AGH event types
- Adjust handler signature for AGH's event interface
- Modify `safeCall()` if AGH handlers return different types

---

### 6.4 Adopt: Graceful Shutdown Pattern

GoClaw's shutdown sequence is battle-tested. **Adapt directly:**

1. Create root `context.WithCancel()` at startup
2. Collect `sync.WaitGroup` pointers in components
3. On signal:
   - Broadcast shutdown notification
   - Stop accepting work (set flag or close intake channel)
   - Drain queues with timeout
   - Cancel root context
   - Wait for all `wg` in components

---

## 7. Comparison to AGH's Current Approach

**Assumption:** AGH is building an Agent OS with concurrent agent runs, pipelines, and tools.

| Aspect                 | GoClaw Pattern                                      | AGH Best Practice                                            |
| ---------------------- | --------------------------------------------------- | ------------------------------------------------------------ |
| **Panic recovery**     | `safego.Recover()`                                  | Use directly (no changes)                                    |
| **Goroutine spawning** | `wg.Add(1)` then `go func()` with `defer wg.Done()` | Standard Go pattern                                          |
| **Work queuing**       | Lane semaphore (unbuffered, tokens represent slots) | Buffered channel if queue needed; semaphore if bounded slots |
| **Cancellation**       | `context.WithCancel()` + `<-ctx.Done()`             | Standard Go pattern                                          |
| **Shutdown**           | Drain → Cancel → Wait                               | Reuse GoClaw's sequence                                      |

---

## 8. Code Snippets Worth Copying

### 8.1 Minimal Panic Recovery (10 lines)

```go
defer func() {
    if r := recover(); r != nil {
        slog.Error("goroutine panicked", "panic", fmt.Sprint(r))
    }
}()
```

### 8.2 Bounded Concurrency with Semaphore (15 lines)

```go
sem := make(chan struct{}, maxConcurrent)
for i := 0; i < maxConcurrent; i++ {
    sem <- struct{}{}
}

// Acquire token
token := <-sem
defer func() { sem <- token }()

// Do work
go func() {
    defer wg.Done()
    // ...
}()
```

### 8.3 Graceful Drain with Timeout (12 lines)

```go
close(queue)  // Signal completion
done := make(chan struct{})
go func() {
    wg.Wait()
    close(done)
}()

select {
case <-done:
    return nil
case <-time.After(timeout):
    cancel()  // Force shutdown
    return fmt.Errorf("drain timeout")
}
```

### 8.4 Component Lifecycle (20 lines)

```go
type Component struct {
    wg     sync.WaitGroup
    ctx    context.Context
    cancel context.CancelFunc
}

func (c *Component) Start(parentCtx context.Context) {
    c.ctx, c.cancel = context.WithCancel(parentCtx)
    c.wg.Add(1)
    go func() {
        defer c.wg.Done()
        defer safego.Recover(nil, "component", "name")
        // Loop until ctx.Done()
    }()
}

func (c *Component) Stop() {
    c.cancel()
    c.wg.Wait()
}
```

---

## 9. Do's and Don'ts for AGH

### Do:

- Always use `defer wg.Done()` in every spawned goroutine
- Always add `defer safego.Recover()` to catch panics
- Use `select { case <-ctx.Done(): return }` in loops
- Pre-allocate semaphore tokens (don't send inside loop)
- Use `atomic.Int64` for counters accessed from multiple goroutines

### Don't:

- Spawn goroutines without `wg.Add()` + `defer wg.Done()`
- Ignore `ctx.Err()` in long-running loops
- Use `time.Sleep()` instead of `<-time.After()` inside select
- Share maps across goroutines without `sync.RWMutex` or `sync.Map`
- Close channels from consumer side (let producer close)

---

## 10. Validation Checklist

Before adopting any pattern in AGH:

- [ ] Copy exact code (don't rewrite)
- [ ] Verify `context` passed through all goroutines
- [ ] Confirm `wg.Wait()` is called in shutdown
- [ ] Test graceful shutdown with concurrent work in progress
- [ ] Add metrics/observability (counters, gauges) early
- [ ] Document which goroutines are "owned" by which components

---

## Conclusion

GoClaw's concurrency patterns are **production-proven** and **minimal** — most don't require external dependencies. The `safego.Recover()` function alone can be adopted immediately with ~30 lines of code. The lane scheduler and event bus are more complex but worth understanding for bounded concurrency and event decoupling.

For AGH, start with `safego.Recover()`, then adopt the graceful shutdown pattern. Adapt the lane scheduler only if you need bounded concurrency beyond simple goroutines + channels.
