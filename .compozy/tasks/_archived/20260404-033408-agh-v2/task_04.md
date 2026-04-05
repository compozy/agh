---
status: completed
title: Session Package
type: ""
complexity: high
dependencies:
    - task_01
    - task_02
    - task_03
---

# Task 04: Session Package

## Overview

Implement the `internal/session` package — the core orchestration layer that manages session lifecycle, agent processes, state machine transitions, resume logic, and Notifier fan-out. This package defines the key interfaces (`AgentDriver`, `EventRecorder`, `Notifier`) and wires ACP + Store implementations through dependency injection via functional options.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST define interfaces: AgentDriver, EventRecorder, Notifier (consumed from acp/ and store/)
- MUST implement Session struct with state machine (starting → active → stopping → stopped)
- MUST implement Manager with functional options: Create, Stop, Resume, Prompt, Get, List
- MUST store ACPSessionID and ACPCaps on Session for resume support
- MUST implement resume flow: load meta.json → spawn agent → try session/load → fallback session/new
- MUST implement Prompt flow: call AgentDriver.Prompt → stream events → record to per-session DB → notify observers
- MUST write events to per-session DB directly (session/ owns per-session writes)
- MUST write meta.json atomically on state changes
- MUST fan out events via Notifier interface (observe/ and httpapi/ subscribe)
- MUST track active sessions in-memory with thread-safe access (sync.RWMutex)
- MUST handle agent process crash (transition to stopped, notify)
- MUST use functional options pattern for NewManager
</requirements>

## Subtasks
- [x] 4.1 Define interfaces: AgentDriver, EventRecorder, Notifier
- [x] 4.2 Implement Session struct with state machine and thread-safe state transitions
- [x] 4.3 Implement Manager with functional options (WithDriver, WithStore, WithNotifier, WithLogger, etc.)
- [x] 4.4 Implement Create flow: resolve agent def → open per-session DB → spawn agent → initialize → session/new
- [x] 4.5 Implement Stop flow: cancel agent → close DB → update state → notify
- [x] 4.6 Implement Resume flow: load meta.json → spawn → try session/load → fallback
- [x] 4.7 Implement Prompt flow: forward to agent → stream events → record → notify
- [x] 4.8 Implement agent crash handling (detect via AgentDriver, transition state, notify)
- [x] 4.9 Implement List and Get operations with thread-safe access

## Implementation Details

Create the following files:
- `internal/session/interfaces.go` — AgentDriver, EventRecorder, Notifier interfaces
- `internal/session/session.go` — Session struct, state machine
- `internal/session/manager.go` — Manager, functional options, Create/Stop/Resume/Prompt/List/Get

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — Core Interfaces, Session Resume Flow, Write Path Ownership

### Old Project Reference
- `.old_project/internal/kernel/session_manager.go` — Session lifecycle and state machine patterns
- `.old_project/internal/kernel/types.go` — Session and agent type definitions
- `.old_project/internal/kernel/session_config.go` — Session configuration and bootstrap
- `.old_project/internal/kernel/kernel.go` — Wiring patterns (what to do differently)
- `.old_project/internal/kernel/api_lifecycle.go` — Session state completion and lifecycle transition patterns

### Related ADRs
- [ADR-002: Pragmatic Flat Architecture](../adrs/adr-002.md) — Interfaces defined where consumed
- [ADR-007: Background Sessions With CLI Prompt](../adrs/adr-007.md) — Session lifecycle model
- [ADR-008: Direct Interfaces and Notifier Pattern](../adrs/adr-008.md) — Notifier for fan-out

## Deliverables
- `internal/session/` package with Manager, Session, interfaces
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with mock AgentDriver and real SQLite **(REQUIRED)**

## Tests
- Unit tests:
  - [x] State machine: valid transitions (starting→active, active→stopping, stopping→stopped)
  - [x] State machine: invalid transitions rejected (stopped→active)
  - [x] Create: opens per-session DB, registers in manager, state becomes active
  - [x] Stop: transitions to stopped, closes DB, calls Notifier.OnSessionStopped
  - [x] Resume: loads meta.json, spawns agent, tries session/load, updates ACPSessionID
  - [x] Resume fallback: session/load fails → falls back to session/new
  - [x] Prompt: streams events from AgentDriver to EventRecorder and Notifier
  - [x] Agent crash: detected, state transitions to stopped, Notifier called
  - [x] List: returns all sessions info
  - [x] Get: returns session by ID, returns false for unknown ID
  - [x] Thread safety: concurrent Create/Stop/Get operations
  - [x] Limits enforcement: Create returns error when max_sessions reached
  - [x] MCP servers merged correctly and passed to ACP session/new
- Integration tests:
  - [x] Full lifecycle: Create → Prompt → events recorded → Stop → Resume → Prompt again
  - [x] Manager with mock driver and real SQLite per-session DB
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Sessions persist across stop/resume cycles
- Notifier correctly fans out all events
- No data races under concurrent access
