---
status: completed
title: Task live timelines, streams, and run detail views
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 04: Task live timelines, streams, and run detail views

## Overview

Add the task-native live surfaces required for the detail, run-detail, and multi-agent-live screens. This task keeps event ordering, live joins, and descendant execution state in the backend so the web app is not forced to stitch session SSE into a task UI architecture.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the live-screen analysis docs before designing task-native live reads
- REFERENCE TECHSPEC sections "Core Interfaces", "API Endpoints", and "Known Risks"
- FOCUS ON "WHAT" — define task-native timeline, stream, tree, and run-detail behavior, not transport registration
- MINIMIZE CODE — reuse task events, existing run/session links, and the current notification pattern instead of creating a second timeline store
- TESTS REQUIRED — event ordering, stream behavior, reconnect semantics, and run-detail aggregation all need coverage
- GREENFIELD: a timeline da task deve ser dona do join principal; nao empurrar a colagem task + session SSE para o cliente como arquitetura oficial
</critical>

<requirements>
- MUST add a task timeline read that normalizes task events with linked run/session context where available
- MUST add a task-scoped stream surface with stable event sequencing suitable for SSE transport
- MUST add a task-tree live view that returns parent plus descendants with active-run and latest-activity state
- MUST add a task-run detail read with task reference, linked session, timing, and operational summary data
- MUST reuse existing task event storage instead of creating a parallel live-event database
- SHOULD keep reconnect and cursor semantics explicit so transport tasks can document them cleanly
</requirements>

## Subtasks
- [x] 4.1 Add task-native live service interfaces and supporting read-model types
- [x] 4.2 Implement timeline and run-detail queries over existing task/run/session state
- [x] 4.3 Implement task-tree live aggregation for parent and descendant execution state
- [x] 4.4 Wire task-scoped stream emission and event sequencing through the existing notifier pattern
- [x] 4.5 Add focused tests for ordering, reconnect behavior, and live aggregation

## Implementation Details

See TechSpec sections "Core Interfaces", "API Endpoints", and ADR-003. The backend should own the joins between task history, active run state, and linked session context so the frontend consumes one live domain model instead of multiple loosely related streams.

### Relevant Files
- `internal/task/interfaces.go` — natural place to define the live/read service surface consumed by handlers
- `internal/task/manager.go` — source of task/run/domain state and likely coordinator for live reads
- `internal/store/globaldb/global_db_task_aux.go` — existing task event and auxiliary read helpers that can back timeline/run-detail queries
- `internal/task/manager_test.go` — unit coverage for ordering, tree aggregation, and run-detail assembly
- `internal/task/manager_integration_test.go` — durable integration coverage for live reads over real task events and runs
- `.compozy/tasks/tasks-ui/analysis/analysis_detail-events-sse.md` — current timeline/SSE gaps
- `.compozy/tasks/tasks-ui/analysis/analysis_run-detail.md` — current run-detail aggregation gaps
- `.compozy/tasks/tasks-ui/analysis/analysis_multi-agent-live.md` — current task-tree aggregation gaps

### Dependent Files
- `internal/api/core/interfaces.go` — task_08 will depend on a stable live/read service contract
- `internal/api/core/sse.go` — task_08 and task_09 will use the live stream with SSE helpers
- `internal/api/contract/tasks.go` — task_07 will serialize these live payloads into public contracts
- `web/src/systems/tasks/hooks/use-task-live.ts` — task_13 and task_17 will consume these task-native live surfaces

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Requires task-native timeline, stream, tree, and run-detail APIs

## Deliverables
- Task-native timeline, stream, tree, and run-detail backend reads
- Explicit event ordering and reconnect-ready stream semantics
- Unit tests with >=80% coverage for live aggregation and sequencing **(REQUIRED)**
- Integration tests over real task/run/event data **(REQUIRED)**
- No requirement for the frontend to stitch session SSE as its primary task-live architecture

## Tests
- Unit tests:
  - [x] Timeline rows are ordered deterministically across task events and run-linked events using stable tie-breakers when timestamps collide
  - [x] Timeline cursor or limit behavior returns stable windows so reconnect or pagination logic does not duplicate or skip rows
  - [x] Run-detail reads include linked task, linked session, timing, tool-call, and token-usage summary fields when available, while omitting optional fields safely
  - [x] Task-tree views include descendants with parent linkage, active-run chips, and latest-activity state across multi-level hierarchies
  - [x] Stream events emit stable sequence metadata and event typing across reconnect-friendly emission boundaries
- Integration tests:
  - [x] Persisted task events and runs produce the expected timeline payload, including run-linked rows and ordering across mixed event types
  - [x] Persisted runs produce run-detail payloads with linked task/session context and operational summaries aligned with real stored data
  - [x] Task-tree reads work across parent/child task hierarchies without N+1 client fetch requirements or missing descendant activity
  - [x] Stream consumers observe task updates for active execution, descendant state changes, and reconnect-style resubscription without sequence drift
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified live-read files
- Task detail, run detail, and multi-agent live surfaces have backend-owned live models
- The API layer can expose task-native live contracts without re-deriving semantics from generic session streams
