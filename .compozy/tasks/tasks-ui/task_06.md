---
status: pending
title: Inbox triage and approval read models
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 06: Inbox triage and approval read models

## Overview

Build the task inbox as a real backend capability instead of a client-side grouping exercise. This task adds the actor-scoped read models and mutations for inbox lanes, approval actions, read state, archive state, and dismiss/triage flows.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_02.md`, and `analysis_inbox.md` before shaping the inbox model
- REFERENCE TECHSPEC sections "Data Models", "API Endpoints", and "Known Risks"
- FOCUS ON "WHAT" — define inbox semantics, approval flows, and actor-scoped triage behavior, not transport registration
- MINIMIZE CODE — reuse durable triage state and observer/task-manager context rather than layering inbox rules in the web client
- TESTS REQUIRED — lane assignment, approval actions, read/archive/dismiss behavior, and actor scoping all need coverage
- GREENFIELD: inbox e approvals precisam ser fluxos reais de dominio/read-model; nao presets visuais em cima de `GET /api/tasks`
</critical>

<requirements>
- MUST add an inbox read model grouped by lanes such as `my_work`, `approvals`, `failed_runs`, `blocked`, and `archived`
- MUST support triage mutations for mark-read, archive, dismiss, and related actor-scoped state transitions
- MUST support task approval and rejection flows aligned with the approval semantics introduced in the task domain
- MUST keep actor scoping explicit so unread/archive state remains isolated per operator context
- MUST expose enough lane metadata and blocking/approval context that the UI can render inbox actions without ad hoc task filtering
- SHOULD keep lane assignment deterministic so dashboard/inbox state remains diagnosable
</requirements>

## Subtasks
- [ ] 6.1 Define the inbox item and lane model over durable task, run, and triage state
- [ ] 6.2 Implement actor-scoped triage mutations for read, archive, and dismiss flows
- [ ] 6.3 Implement approval and rejection command behavior for approval-backed tasks
- [ ] 6.4 Add tests for lane assignment, actor isolation, and approval/triage mutations

## Implementation Details

See TechSpec sections "Data Models", "API Endpoints", "Known Risks", and ADR-004. The inbox should be a first-class read model with explicit mutations, not a client-only filter over generic task lists.

### Relevant Files
- `internal/task/manager.go` — approval and triage mutations belong in the write-side task domain
- `internal/task/manager_test.go` — approval/reject and triage mutation coverage
- `internal/store/globaldb/global_db_task.go` — durable triage and approval-related persistence/query support
- `internal/store/globaldb/global_db_task_aux.go` — likely home for lane-oriented or actor-scoped task query helpers
- `internal/observe/tasks.go` — aggregate inbox shaping built from task and triage state
- `.compozy/tasks/tasks-ui/analysis/analysis_inbox.md` — documents the missing lane, archive, and approval semantics

### Dependent Files
- `internal/api/contract/tasks.go` — task_07 will expose inbox items and approval/triage request payloads
- `internal/api/core/tasks.go` — task_08 will expose inbox list and mutation handlers
- `internal/extension/host_api_tasks.go` — task_11 may expose inbox aggregates to extension consumers
- `web/src/systems/tasks/hooks/use-task-inbox.ts` — task_13 and task_16 will depend on this model

### Related ADRs
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Assigns inbox aggregation to dedicated task read models instead of client-side filtering

## Deliverables
- Actor-scoped inbox read model with lane grouping and action-ready metadata
- Approval, rejection, read, archive, and dismiss backend behavior
- Unit tests with >=80% coverage for lane assignment and mutation semantics **(REQUIRED)**
- Integration tests proving actor-scoped triage and approval flows **(REQUIRED)**
- Deterministic inbox behavior ready for transport and UI work

## Tests
- Unit tests:
  - [ ] Inbox items are assigned to the correct lanes for approvals, my work, blocked, failed runs, and archived states for mixed task inputs
  - [ ] Mark-read, archive, and dismiss mutations update actor-scoped triage state, unread counts, and lane membership without leaking to other actors
  - [ ] Approval and rejection flows update approval state, runnable status, and lane transitions consistently for approval-backed tasks
  - [ ] Failed-run and blocked-task lane shaping includes the expected reason metadata, latest activity, and linked run references
  - [ ] Inbox filters such as lane, unread-only, and text query return the expected item set and grouped counts
- Integration tests:
  - [ ] Persisted actor triage state survives reload and remains scoped to the originating actor context across read, archive, and dismiss mutations
  - [ ] Approval-backed tasks move between inbox lanes correctly after approve or reject actions against real stored task state
  - [ ] Archive, dismiss, and mark-read mutations update unread or archived counts on subsequent inbox queries without phantom duplicates
  - [ ] Inbox queries surface the expected grouped lanes, unread counts, and actor-scoped visibility against real stored task state
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified inbox, manager, and store files
- The inbox and approval flows are real backend capabilities rather than UI-only grouping logic
- Later API and frontend tasks can build the Paper inbox without inventing missing state
