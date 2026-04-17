---
status: pending
title: Daemon relaunch helper and restart operation store
type: backend
complexity: high
dependencies: []
---

# Task 03: Daemon relaunch helper and restart operation store

## Overview

Implement the production-grade restart path chosen in the TechSpec by extracting detached launch logic into a reusable relaunch helper flow. This task is responsible for persisted restart state, the `agh daemon relaunch` internal command path, and durable status transitions that survive daemon shutdown and reconnect.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "API Endpoints", "Technical Dependencies", and "Known Risks"
- FOCUS ON "WHAT" — build a reliable asynchronous restart flow, not an in-process exec shortcut
- MINIMIZE CODE — factor existing detached start logic instead of duplicating it in daemon and CLI
- TESTS REQUIRED — helper spawn, state durability, release waiting, and failure reporting all need coverage
- GREENFIELD: Não usar `syscall.Exec`, não subir o replacement daemon antes de liberar lock/socket/daemon.json
</critical>

<requirements>
- MUST implement the internal helper subcommand `agh daemon relaunch`
- MUST persist restart-operation state under `HomePaths.HomeDir/restarts/<operation_id>.json`
- MUST update restart status through durable states such as `stopping`, `waiting_release`, `starting`, `ready`, and terminal failure
- MUST reuse or extract detached-process launch behavior from the existing CLI start path instead of duplicating spawn logic
- MUST wait for current-daemon singleton resource release before launching the replacement process
- MUST preserve enough restart context for the UI to reconcile success or failure after reconnect
</requirements>

## Design References

The restart action surface lives on the `General` settings page; every restart-required mutation across the other settings screens ultimately triggers this flow. See `_techspec.md` → *Design References* for the full 10-artboard table.

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| General (restart action) | `docs/design/paper/settings/AGH Settings — General@2x.png` | `AGH Settings — General` (`VP8-0`) |

## Subtasks

- [ ] 3.1 Extract reusable detached launch behavior from the CLI start path
- [ ] 3.2 Add restart-operation state persistence and lifecycle transitions under `~/.agh/restarts/`
- [ ] 3.3 Implement the `agh daemon relaunch` helper flow and operation context handoff
- [ ] 3.4 Wait for lock and `daemon.json` release before replacement-daemon launch
- [ ] 3.5 Mark the replacement daemon `ready` only after successful boot and fresh discovery state
- [ ] 3.6 Cover success, helper failure, and replacement-boot failure paths with tests

## Implementation Details

See TechSpec sections "API Endpoints", "Integration Points", "Development Sequencing", and ADR-003. This task should stay focused on runtime orchestration plus persistent status storage; API exposure of the operation comes later through contract/core/transports.

### Relevant Files

- `internal/cli/daemon.go` — current detached daemon start logic to factor into shared relaunch behavior
- `internal/daemon/daemon.go` — daemon composition root that will own restart services and state handling
- `internal/daemon/boot.go` — boot sequencing and readiness hooks the replacement daemon must satisfy
- `internal/daemon/info.go` — daemon discovery state that helps define restart readiness
- `internal/daemon/lock.go` — singleton release condition the helper must wait for

### Dependent Files

- `internal/api/contract/contract.go` — will expose restart action payloads in task_04
- `internal/api/core/handlers.go` — will call restart orchestration in task_05
- `internal/daemon/*_integration_test.go` — should verify persisted restart state and helper transitions
- `internal/cli/*_test.go` — should protect extracted detached launch behavior

### Related ADRs

- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines the helper-based relaunch flow and durable restart status

## Deliverables

- Shared detached-launch helper or equivalent reusable runtime for restart orchestration
- Restart-operation persistence under `~/.agh/restarts/` with durable status transitions
- Internal `agh daemon relaunch` helper path that coordinates shutdown and replacement start **(REQUIRED)**
- Unit tests for state transitions and helper behavior **(REQUIRED)**
- Integration tests for release waiting, replacement boot, and persisted status reconciliation **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] Restart operation records serialize and reload with the expected state transitions
  - [ ] Restart operation state transitions reject illegal regressions or double-terminal updates
  - [ ] Helper startup failure is persisted as a terminal failed operation
  - [ ] Release waiting logic does not mark replacement start until lock and daemon info are released
  - [ ] Replacement daemon readiness requires fresh discovery state before marking `ready`
  - [ ] Restart status reads preserve `old_pid`, `active_session_count`, timestamps, and only set `new_pid` after replacement boot succeeds
- Integration tests:
  - [ ] Restart operation file is written before the current daemon begins shutdown
  - [ ] Persisted restart operation captures the pre-restart PID and active session count used by the helper flow
  - [ ] Successful relaunch produces a new PID, releases the old daemon lock, and records a `ready` persisted operation state
  - [ ] Helper startup failure remains queryable through restart-status polling after the original daemon exits
  - [ ] Replacement boot failure remains visible through the persisted restart record after reconnect
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for new restart orchestration code
- Restart status survives daemon shutdown and replacement-daemon failure
- The daemon restart path no longer depends on ad hoc CLI-only spawn code
