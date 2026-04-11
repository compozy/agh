---
status: pending
title: Compose automation manager and wire daemon boot lifecycle
type: infra
complexity: high
dependencies:
  - task_04
  - task_05
---

# Task 06: Compose automation manager and wire daemon boot lifecycle

## Overview

Compose the store, dispatcher, scheduler, and trigger engine into the built-in automation manager described by the TechSpec. This task is where TOML synchronization, overlay resolution, lifecycle start and stop, and daemon boot integration become one coherent subsystem instead of a set of disconnected pieces.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement the automation `Manager` as the built-in daemon component that owns scheduler, trigger engine, dispatcher, store coordination, and lifecycle state.
2. MUST synchronize TOML-defined jobs and triggers into persistence on boot while preserving the hardened ownership model where only `enabled/disabled` is overlay-mutable at runtime.
3. MUST integrate automation boot into the existing daemon boot pipeline after hooks are available and before servers start, without breaking the current extension boot path.
4. MUST provide clean shutdown semantics so scheduler activity, trigger ingestion, and in-flight automation work receive context cancellation and stop deterministically.
</requirements>

## Subtasks
- [ ] 6.1 Implement the composed automation manager and its lifecycle methods
- [ ] 6.2 Add TOML sync and config-source overlay resolution for jobs and triggers
- [ ] 6.3 Wire the manager into daemon runtime dependencies and boot ordering
- [ ] 6.4 Add lifecycle status and health data needed by later transport work
- [ ] 6.5 Add integration tests for boot, sync, restart, and shutdown behavior

## Implementation Details

Use the TechSpec sections "Manager", "Daemon Boot Integration", "TOML jobs are source-of-truth", and "Testing Approach". Codebase exploration shows the current boot order is `bootHooks -> bootExtensions -> bootServers`, so automation should integrate without reintroducing dependency cycles or invalidating extension startup.

### Relevant Files
- `internal/daemon/boot.go` — Defines the current boot pipeline and cleanup ordering that automation must join
- `internal/daemon/daemon.go` — Runtime dependency publication will need the automation manager added here
- `internal/config/config.go` — The loaded config already flows through boot and will supply automation TOML definitions
- `internal/store/globaldb/global_db.go` — TOML sync and overlay resolution will persist through the global DB work from earlier tasks

### Dependent Files
- `internal/api/core/` — Transport surfaces in later tasks will depend on the manager lifecycle and query methods exposed here
- `internal/extension/host_api.go` — Extension automation methods will call into this manager once it exists
- `internal/api/httpapi/routes.go` — HTTP health and automation handlers will later consume the manager status exposed here

### Related ADRs
- [ADR-001: Built-In Daemon Component with Extension Integration Points](adrs/adr-001.md) — Governs built-in manager ownership and later extension exposure
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Requires one package and one manager composing both activation modes

## Deliverables
- Composed automation manager with lifecycle, sync, and overlay behavior
- Daemon boot integration and runtime dependency publication
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for boot, restart, sync, and shutdown behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] TOML sync creates missing config-backed jobs and triggers while leaving dynamic entries intact
  - [ ] TOML sync updates definition-owned fields from config but preserves runtime enabled overlays separately
  - [ ] Manager status reports enabled counts, trigger counts, and next-fire metadata without requiring transport-specific wrappers
  - [ ] Disabling a config-backed definition at runtime updates only the overlay state returned by the manager
- Integration tests:
  - [ ] Daemon boot initializes automation after hooks are available and before servers start accepting requests
  - [ ] Restarting the daemon preserves config-backed enabled overlays across TOML resync
  - [ ] Manager shutdown cancels active scheduler and trigger work cleanly without leaked goroutines
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Automation is booted and shut down as a first-class daemon subsystem
- TOML sync and enabled-overlay semantics are enforced by the manager rather than by transport-specific logic

