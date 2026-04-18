---
status: completed
title: Enriched task reads and draft publication
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 03: Enriched task reads and draft publication

## Overview

Expand the task manager’s list and detail reads so the UI can render split view, kanban, and the create flow without stitching basic summaries into richer operator cards on the client. This task also adds the explicit draft-publication command expected by the Paper create flow.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_01.md`, and `task_02.md` before reshaping task reads
- REFERENCE TECHSPEC sections "Core Interfaces", "Data Models", and "API Endpoints"
- FOCUS ON "WHAT" — expose richer manager-owned read models and publication behavior, not transport serialization
- MINIMIZE CODE — enrich existing point reads and queries instead of creating a parallel UI-only query layer in the backend
- TESTS REQUIRED — list summaries, detail joins, search behavior, and publication transitions must all be covered
- GREENFIELD: nao empurrar join de dependencias, resumo de run ativa, ou search textual para o frontend se o backend ja tem contexto suficiente
</critical>

<requirements>
- MUST add enriched task list and detail read models with child counts, dependency counts, active-run summary, latest activity, and blocker-friendly dependency references
- MUST support textual search over task title and identifier for the main tasks list
- MUST expose explicit draft publication behavior that computes the resulting runnable status instead of requiring the client to patch status manually
- MUST keep dependency and child references rich enough that the UI does not need N+1 fetches to show human-meaningful labels
- SHOULD keep the enriched read models reusable by both list and kanban surfaces
</requirements>

## Subtasks
- [x] 3.1 Extend task query and view types for enriched list and detail summaries
- [x] 3.2 Add manager/store support for search, activity ordering, and blocker-friendly dependency reads
- [x] 3.3 Implement explicit draft publication behavior in the task manager
- [x] 3.4 Add tests for enriched list cards, detail joins, search, and publication transitions

## Implementation Details

See TechSpec sections "Core Interfaces", "Data Models", and the analysis docs for split view, kanban, create modal, and empty state. This task should leave the manager with UI-ready point reads that still reflect domain meaning, not frontend presentation leakage.

### Relevant Files
- `internal/task/types.go` — query, summary, and expanded view types that need enriched read-model support
- `internal/task/manager.go` — list/detail assembly, publication logic, and query behavior
- `internal/task/manager_test.go` — search, enriched-card, and publication behavior coverage
- `internal/store/globaldb/global_db_task.go` — list query/filter support and search-friendly fetches
- `internal/store/globaldb/global_db_task_aux.go` — likely place for richer activity/run/dependency helpers
- `.compozy/tasks/tasks-ui/analysis/analysis_list-split-view.md` — identifies the current summary gaps for split view
- `.compozy/tasks/tasks-ui/analysis/analysis_create-modal.md` — identifies the draft-publication and create-flow gaps

### Dependent Files
- `internal/api/contract/tasks.go` — task_07 will map the enriched manager views into transport payloads
- `internal/api/core/tasks.go` — task_08 will expose publication and enriched point reads
- `web/src/systems/tasks/` — frontend list, kanban, and create surfaces in task_13 and task_14 depend on these richer reads
- `.compozy/tasks/tasks-ui/analysis/analysis_kanban-view.md` — the board view requirements should be satisfiable from these reads

### Related ADRs
- [ADR-002: Expand the Task Domain for Paper-Parity Semantics](adrs/adr-002.md) — List/create reads must expose the first-class semantics added to the domain

## Deliverables
- Enriched task list and detail manager reads suitable for split view and kanban
- Search support for title/identifier filtering
- Explicit draft publication operation owned by the manager **(REQUIRED)**
- Unit tests with >=80% coverage for search, enriched summaries, and publication behavior **(REQUIRED)**
- Integration tests proving the new views and publication behavior against the real store **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Enriched list items include `priority`, `max_attempts`, child/dependency counts, last activity, and active-run chips for mixed task states
  - [x] Enriched detail reads surface blocker references with identifier, title, status, and owner context so the UI can label dependencies without extra fetches
  - [x] Search by title and identifier matches case-insensitively and preserves stable activity-based ordering for equally relevant results
  - [x] Combined list filters such as `workspace`, `status`, `parent_task_id`, and text query produce the expected task set without dropping enriched fields
  - [x] Publishing a draft task transitions it into `ready` or `blocked` based on dependency state and rejects publication for already-published tasks
- Integration tests:
  - [x] Real-store list queries return enriched summaries, active-run metadata, and latest-activity values without requiring per-task follow-up reads
  - [x] `GetTask` returns children, dependency references, run summaries, and recent event context aligned with the enriched detail contract
  - [x] Store-backed search respects text query together with persisted workspace/status filters and returns the expected ordering after reload
  - [x] Publishing a persisted draft task survives reload and subsequent list/detail reads return the reconciled task state consistently
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified read-model and manager files
- Split view, kanban, and create flows have manager-owned reads and publication semantics
- The frontend no longer needs to derive core task-card meaning from multiple unrelated payloads
