---
status: completed
title: Detached harness work on task runtime metadata and submission paths
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 06: Detached harness work on task runtime metadata and submission paths

## Overview

Map detached harness execution onto the existing task runtime and `task_runs` substrate. This task makes async harness work durable and inspectable without introducing a parallel `BackgroundRun` entity, while preserving enough metadata for later wake-up targeting and policy-based reentry.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_01.md`, and the current task runtime before starting
- REFERENCE TECHSPEC sections "Workstream 5: Detached Async Runtime on Task Infrastructure" and "Why This Decision Is Final"
- FOCUS ON "WHAT" - represent harness-owned detached work on `task` and `task_run`; do not yet emit synthetic wake-ups on completion here
- MINIMIZE CODE - reuse task runtime concepts and persistence instead of cloning a second async state machine
- TESTS REQUIRED - metadata shape, idempotency, session linkage, and boot-recovery compatibility need coverage
- GREENFIELD: nao recriar um `BackgroundRun`; o substrato duravel aqui e `task`/`task_runs`
</critical>

<requirements>
- MUST reuse `task` and `task_run` as the durable substrate for detached harness work
- MUST add explicit harness-oriented origin and metadata so detached harness runs are distinguishable from ordinary task traffic
- MUST preserve task-runtime idempotency, recovery, and session-bridge semantics
- MUST carry owner-session and wake-up targeting data needed by later synthetic reentry
- SHOULD keep the harness-to-task mapping daemon-owned rather than leaking task-domain details into higher-level prompt code
</requirements>

## Subtasks
- [x] 6.1 Define the harness metadata and origin contract carried on detached task records and runs
- [x] 6.2 Add daemon-owned submission paths that enqueue detached harness work on the task runtime
- [x] 6.3 Thread owner-session, workspace, and wake-up targeting metadata through persisted task/run records
- [x] 6.4 Reuse task-runtime idempotency and recovery semantics for detached harness work
- [x] 6.5 Add unit and integration coverage for submission, persistence, and recovery behavior

## Implementation Details

See TechSpec "Workstream 5: Detached Async Runtime on Task Infrastructure" and ADR-003. The key design constraint is to make harness async work durable and observable while still feeling like harness-owned runtime behavior, not a second task subsystem with duplicated lifecycle rules.

### Relevant Files
- `internal/daemon/task_runtime.go` - daemon-owned bridge onto the task runtime and the natural place for harness-owned detached submission
- `internal/task/types.go` - task, run, origin, and metadata types that detached harness work must reuse
- `internal/task/manager.go` - manager APIs for enqueueing and transitioning durable runs
- `internal/store/globaldb/global_db_task.go` - canonical persistence for task and task-run records
- `internal/store/globaldb/global_db_task_aux.go` - idempotency and task-event helpers that detached harness work should reuse
- `internal/daemon/harness_detached_work.go` - new daemon-owned bridge module introduced by this task

### Dependent Files
- `internal/daemon/task_runtime_test.go` - task-runtime integration tests should cover harness-owned detached submissions
- `internal/task/manager_test.go` - manager semantics around harness metadata and idempotency need assertions
- `internal/store/globaldb/global_db_task_test.go` - persistence tests should cover harness-owned metadata and run targeting
- `internal/daemon/daemon_automation_task_integration_test.go` - existing task-runtime integration lane is a likely place for detached harness parity coverage

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - Detached work still derives behavior from the same daemon-owned policy vocabulary
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - This task is the concrete implementation of the substrate decision

### External References
- `.resources/claude-code/utils/swarm/spawnUtils.ts` - good reference for carrying runtime inheritance and execution context into detached work
- `.resources/claude-code/utils/swarm/backends/teammateModeSnapshot.ts` - useful precedent for capturing mode/runtime state at detached-spawn time
- `.resources/openclaw/src/agents/tools/sessions-spawn-tool.ts` - strong reference for runtime selection and detached/subagent run registration
- `.resources/openclaw/src/agents/subagent-spawn.ts` - useful model for passing focused detached-run context without reusing the full parent state blindly
- `.resources/hermes/tools/process_registry.py` - concrete reference for durable detached sessions, completion queues, and restart recovery
- `.resources/openfang/crates/openfang-kernel/src/background.rs` - useful background-runtime reference once detached work has a durable substrate
- `.resources/openfang/crates/openfang-types/src/scheduler.rs` - helpful schema reference for wake-up targets and scheduled run metadata

## Deliverables
- Daemon-owned detached harness submission path implemented on top of the task runtime
- Harness-specific detached-work metadata and origin conventions **(REQUIRED)**
- Owner-session and wake-up targeting persisted with detached runs **(REQUIRED)**
- Reuse of task-runtime idempotency and recovery semantics, not a parallel implementation **(REQUIRED)**
- Regression coverage for scope validation, metadata persistence, and boot-recovery compatibility **(REQUIRED)**
- Unit and integration tests with >=80% coverage for the new detached-work bridge paths **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Detached harness submission creates `task` and `task_run` records with the expected daemon or harness origin and metadata payload
  - [x] Idempotent detached submissions reuse task-runtime deduplication semantics and do not create duplicate runs
  - [x] Owner-session, workspace binding, and wake-up targeting metadata are persisted and retrievable through the store
  - [x] Unsupported detached-work scopes, missing target session ids, or invalid metadata fail validation cleanly
  - [x] Boot-recovery metadata survives store round-trips without losing harness-specific fields
- Integration tests:
  - [x] Detached harness work survives task-runtime boot recovery using the existing reconciliation rules instead of a harness-only path
  - [x] Detached harness runs can be listed and inspected through the normal task-runtime persistence and query paths
  - [x] Detached harness submission can create workspace-scoped and global-scoped runs using the same daemon bridge without semantic drift
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Detached harness work uses the existing task runtime as its durable substrate
- Later reentry work can target real task-run completions without inventing a second async runtime model
