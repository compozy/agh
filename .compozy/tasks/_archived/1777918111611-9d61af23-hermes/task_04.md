---
status: completed
title: Durable Automation Scheduler
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 04: Durable Automation Scheduler

## Overview

Replace volatile automation scheduling assumptions with a durable scheduler state model. This task persists automation cursors, advances state before dispatch, reconciles missed work on boot, separates delivery errors from schedule state, and proves at-most-once dispatch behavior under crash and restart scenarios.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-002, and task_01 outputs before changing automation state
- ADVANCE durable cursor/state before dispatch to prevent duplicate delivery after restart
- DO NOT use `time.Sleep()` in orchestration tests; use controlled clocks and synchronization
- DO NOT mix delivery failure details into scheduler cursor correctness
- EVERY scheduler loop must stop on context cancellation and be owned by daemon lifecycle
</critical>

<requirements>
- MUST add persistent scheduler state schema and store APIs
- MUST advance next-run cursor durably before dispatching automation work
- MUST implement boot reconciliation for missed runs with an explicit catch-up or misfire policy
- MUST track delivery errors separately from scheduler state
- MUST add tests for restart, crash window, missed runs, and at-most-once dispatch
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 4.1 Add scheduler state migration and global store APIs for automation cursor metadata
- [x] 4.2 Refactor scheduler execution to persist next-run state before dispatch
- [x] 4.3 Implement boot reconciliation with documented catch-up and misfire behavior
- [x] 4.4 Separate delivery error recording from cursor advancement and scheduling health
- [x] 4.5 Add deterministic tests for missed runs, restart windows, duplicate prevention, and cancellation
- [x] 4.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

The scheduler must be able to resume from durable state without guessing from in-memory timers. Store APIs should make cursor updates transactional enough to prove the dispatch window invariant. CLI/API changes should expose enough scheduler state for users to debug paused, failed, or misfired automations.

### Relevant Files
- `internal/automation/schedule.go` - schedule calculation and next-run behavior
- `internal/automation/manager.go` - scheduler lifecycle ownership
- `internal/automation/dispatch.go` - dispatch and delivery-error boundaries
- `internal/automation/model/` - automation state models
- `internal/store/globaldb/global_db_automation.go` - durable automation state persistence
- `internal/api/contract/automation.go` - API-visible automation state
- `internal/api/core/automation.go` - automation handlers and conversions
- `internal/cli/automation.go` - CLI automation status and diagnostics

### Dependent Files
- `internal/automation/*_test.go` - scheduler and dispatch invariants
- `internal/store/globaldb/*automation*_test.go` - durable cursor storage tests
- `internal/api/core/*automation*_test.go` - API contract coverage
- `web/src/` - automation UI or typed client changes for scheduler state
- `packages/site/` - automation scheduling docs and examples
- `.compozy/tasks/hermes/task_10.md` - QA plan must include restart and duplicate-prevention scenarios

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - includes automation hardening as a selected track
- [ADR-002: Durable Automation Scheduler State](adrs/adr-002-durable-automation-scheduler-state.md) - defines cursor, catch-up, and delivery-error decisions

## Deliverables
- Durable automation scheduler state schema and store APIs
- Scheduler implementation that persists cursor state before dispatch
- Boot reconciliation for missed automation runs
- Separate delivery-error reporting and health state
- Restart and at-most-once tests for scheduler behavior
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] Schedule state advances before dispatch
  - [x] Missed-run policy is deterministic and documented by tests
  - [x] Delivery failure does not roll back or corrupt scheduler cursor state
  - [x] Scheduler loop exits cleanly on context cancellation
- Integration tests:
  - [x] Restart after cursor advancement does not duplicate an already claimed run
  - [x] Restart with missed runs reconciles according to policy
  - [x] API/CLI automation status exposes durable scheduler state and delivery errors consistently
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Automation scheduling survives daemon restart without duplicate dispatch
- Operators can distinguish schedule health from delivery failures
- Catch-up and misfire semantics are explicit, tested, and documented
- Affected backend, CLI, and web/docs tests pass
