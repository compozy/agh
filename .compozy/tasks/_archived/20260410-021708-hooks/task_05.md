---
status: completed
title: Async worker pool
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 5: Async worker pool

## Overview

Implement the fixed-size goroutine worker pool for async hook execution using Go stdlib primitives (buffered channel, WaitGroup, context). This mirrors the single-worker pattern in `internal/memory/consolidation/runtime.go` but extends it to N workers with backpressure and graceful shutdown.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement a fixed-size goroutine pool with configurable worker count (default 4)
- MUST use a buffered channel as work queue with configurable capacity (default 64)
- MUST implement non-blocking send with `select`/`default` for backpressure — full buffer drops the hook
- MUST log dropped hooks as `hook.dispatch.async_dropped` with queue depth
- MUST implement graceful shutdown: close channel, drain with deadline (10s), `sync.WaitGroup.Wait()`
- MUST wrap each worker's execution in `recover()` to prevent panicking hooks from killing the pool
- MUST use `select { case task := <-ch: ... case <-ctx.Done(): return }` in each worker
- MUST track all goroutines via `sync.WaitGroup` — no fire-and-forget
</requirements>

## Subtasks
- [x] 5.1 Define async task type and pool configuration struct
- [x] 5.2 Implement worker pool with N goroutines consuming from buffered channel
- [x] 5.3 Implement non-blocking submit with drop-on-full backpressure
- [x] 5.4 Implement graceful shutdown with drain deadline
- [x] 5.5 Write unit tests including backpressure, shutdown, and panic recovery

## Implementation Details

Create new file in `internal/hooks/`:
- `pool.go` — Worker pool struct, Start, Submit, Close methods

Reference `internal/memory/consolidation/runtime.go` lines 39-51, 135-150 for the existing single-worker channel pattern. Reference TechSpec "Async Worker Pool" section.

### Relevant Files
- `internal/memory/consolidation/runtime.go:39-150` — Single-worker channel pattern to extend
- `internal/hooks/types.go` (task_01) — HookRunRecord for async execution telemetry

### Dependent Files
- `internal/hooks/` — Hooks struct (task_06) owns and operates the pool

### Related ADRs
- [ADR-008: Stdlib Worker Pool for Async Hook Execution](../adrs/adr-008.md) — Defines pool design

## Deliverables
- `internal/hooks/pool.go` with complete worker pool implementation
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Pool starts N workers — submitting N tasks runs them concurrently
  - [x] Submit to pool with available capacity succeeds
  - [x] Submit to full pool drops the task and returns false
  - [x] Dropped task is logged with queue depth
  - [x] Graceful shutdown: pending tasks in channel are drained before Close returns
  - [x] Shutdown with deadline: tasks exceeding deadline are abandoned, Close returns
  - [x] Panicking task is recovered — worker continues processing next task
  - [x] Context cancellation stops all workers
  - [x] Pool with 0 submitted tasks shuts down cleanly
  - [x] Concurrent submit from multiple goroutines is safe (no data race with -race)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `-race` flag passes with concurrent submit/shutdown
- No goroutine leaks — all workers join on Close()
