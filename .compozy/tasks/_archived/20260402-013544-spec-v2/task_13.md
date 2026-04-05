---
status: completed
title: Error Handling & Resilience
type: ""
complexity: medium
dependencies:
    - task_05
    - task_06
---

# Task 13: Error Handling & Resilience

## Overview
Implement the fault tolerance layer including suture supervisor trees for agent lifecycle management with automatic restart and exponential backoff, sony/gobreaker circuit breakers per agent for message delivery protection, PID-based liveness health checks, and failure notification to workgroup masters.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST wrap each agent as a suture.Service with Serve(ctx) per docs/spec-v2/09-resilience.md
- MUST configure suture with restart_max_attempts and restart_backoff_base from config
- MUST implement per-agent circuit breakers using sony/gobreaker per docs/spec-v2/09-resilience.md
- MUST configure breaker: 3 consecutive failures to trip, 30s timeout, 1 probe in half-open
- MUST implement PID liveness checks via syscall.Kill(pid, 0) at health_check_interval
- MUST implement PTY EOF detection for near-instantaneous crash detection
- MUST notify workgroup master on agent failure (worker/reviewer/researcher death)
- MUST notify parent master and freeze workgroup on master death
- MUST log all failure events to the events table
</requirements>

## Subtasks
- [x] 13.1 Implement AgentService wrapping AgentDriver as suture.Service
- [x] 13.2 Configure suture supervisor with restart policy from config
- [x] 13.3 Implement per-agent circuit breaker wrapping message delivery
- [x] 13.4 Implement PID liveness health check goroutine at configured interval
- [x] 13.5 Implement failure notification chain (agent → master → parent)
- [x] 13.6 Implement workgroup freeze on master death

## Implementation Details
Refer to docs/spec-v2/09-resilience.md for suture spec, circuit breaker config, health check implementation, and failure scenarios table.

### Relevant Files
- `docs/spec-v2/09-resilience.md` — complete resilience spec
- `docs/spec-v2/02-kernel.md` — graceful shutdown sequence

### Dependent Files
- `internal/pty/` — PTY manager for process handle and PID
- `internal/registry/` — agent and workgroup state updates
- `internal/state/` — event logging for failure events
- `internal/transport/` — NATS messaging for failure notifications

## Deliverables
- AgentService (suture.Service wrapper)
- Circuit breaker integration for message delivery
- Health check goroutine
- Failure notification and workgroup freeze logic
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] AgentService.Serve calls driver.Start and blocks until context cancelled
  - [x] Suture restarts agent after crash (up to max_attempts)
  - [x] Circuit breaker opens after 3 consecutive SendMessage failures
  - [x] Circuit breaker half-open allows 1 probe after 30s timeout
  - [x] Circuit breaker closes on successful probe
  - [x] Health check detects dead PID (Alive=false)
  - [x] Health check detects alive PID (Alive=true)
  - [x] Worker death notifies workgroup master
  - [x] Master death freezes workgroup (state → closing)
  - [x] Master death notifies parent workgroup master
  - [x] Failure events logged to events table
- Test coverage target: >=80%
- All tests must pass with -race flag

## Validation Evidence
- `go test -race ./internal/kernel ./internal/pty ./internal/drivers/claude`
- `go test -cover ./internal/kernel` -> `80.3%`
- `make verify`

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Failure scenarios match docs/spec-v2/09-resilience.md table
