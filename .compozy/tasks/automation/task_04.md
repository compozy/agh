---
status: pending
title: Build scheduler runtime for scheduled jobs
type: backend
complexity: high
dependencies:
  - task_03
---

# Task 04: Build scheduler runtime for scheduled jobs

## Overview

Build the time-based automation runtime on top of `gocron v2` so cron, interval, and one-shot jobs can register, fire, and shut down cleanly. This task should make scheduled jobs a thin activation source over the shared dispatcher rather than a second execution model.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST wrap `gocron v2` behind an automation scheduler surface that can register, update, unregister, start, and stop scheduled jobs.
2. MUST support the TechSpec schedule modes `cron`, `every`, and `at` with deterministic next-run tracking.
3. MUST use scheduler-local singleton protection only as overlap protection for one scheduled job while still delegating execution governance to the shared dispatcher.
4. MUST respect daemon shutdown via context cancellation and avoid fire-and-forget scheduling goroutines.
</requirements>

## Subtasks
- [ ] 4.1 Add the scheduler runtime wrapper around `gocron v2`
- [ ] 4.2 Map `cron`, `every`, and `at` schedule specs into scheduler registrations
- [ ] 4.3 Add singleton overlap protection and unregister behavior for disabled or deleted jobs
- [ ] 4.4 Surface next-run metadata so later API and UI tasks can display it
- [ ] 4.5 Add tests for scheduling, overlap protection, and shutdown behavior

## Implementation Details

Implement this task using the TechSpec sections "Scheduler", "Testing Approach", and "Development Sequencing". The scheduler should know how to translate scheduled jobs into dispatch requests, but it should not duplicate retry, concurrency, or run-history logic already owned by the dispatcher.

### Relevant Files
- `go.mod` — The scheduler runtime introduces `github.com/go-co-op/gocron/v2`
- `internal/automation/dispatch.go` — Scheduled fires must route through the shared dispatcher from task 03
- `internal/daemon/boot.go` — The eventual manager lifecycle must be able to start and stop the scheduler cleanly
- `internal/store/globaldb/global_db.go` — Scheduler registration and refresh will consume persisted job definitions and enabled state

### Dependent Files
- `internal/automation/manager.go` — The composed manager will own scheduler lifecycle in task 06
- `internal/api/core/` — Later transport work will expose next-run and job-state information coming from this runtime

### Related ADRs
- [ADR-003: gocron v2 as In-Process Scheduling Runtime](adrs/adr-003.md) — Governs the scheduler dependency choice and lifecycle expectations
- [ADR-004: Configurable Per-Job Retry with Fire Limits](adrs/adr-004.md) — Scheduled fires must still respect dispatcher-level retry and fire-limit rules

## Deliverables
- `gocron v2`-backed scheduler runtime under `internal/automation/`
- Schedule registration, next-run tracking, and overlap protection for jobs
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for fast schedules, overlap prevention, and shutdown **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] A `cron` job computes the expected next run from a deterministic clock input
  - [ ] An `every:30m` job registers with the expected interval semantics
  - [ ] An `at:` one-shot job unregisters after firing once
  - [ ] Singleton protection blocks an overlapping second fire while the previous run is still active
  - [ ] Disabling or unregistering a job removes future scheduler fires
- Integration tests:
  - [ ] A fast `@every 1s` schedule dispatches through the shared dispatcher and records a run
  - [ ] Scheduler shutdown cancels in-flight work via context instead of leaving goroutines behind
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Scheduled jobs can be registered and fired without bypassing dispatcher governance
- Scheduler lifecycle is cleanly startable and stoppable from the daemon runtime

