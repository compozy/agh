---
status: pending
title: ClaimNextRun And Lease Fencing Service
type: backend
complexity: critical
dependencies:
  - task_07
---

# Task 08: ClaimNextRun And Lease Fencing Service

## Overview
Implement the authoritative task-run claim primitive and lease fencing rules. `ClaimNextRun(criteria)` becomes the only way autonomous sessions take new work, while heartbeat, completion, failure, and release operations must prove ownership through the current claim token.

<critical>
- ALWAYS READ `_techspec.md`, ADR-003, ADR-004, ADR-009, and ADR-010 before changing claim behavior
- `ClaimNextRun(criteria)` IS THE ONLY AUTHORITATIVE NEXT-WORK PRIMITIVE - scheduler and coordinator code must not bypass it
- EVERY MUTATION AFTER CLAIM MUST BE FENCED BY THE CURRENT CLAIM TOKEN
- ONE ACTIVE TASK-RUN LEASE PER SESSION IN MVP unless the TechSpec is explicitly changed
- TESTS REQUIRED - race, stale token, expired lease, heartbeat, completion, and boot recovery paths are mandatory
- NO WORKAROUNDS - do not use `time.Sleep` to make lease tests pass; use deterministic clocks and synchronization
</critical>

<requirements>
- MUST add transactional `ClaimNextRun(criteria)` to the task service/globaldb layer using SQLite transaction semantics.
- MUST filter eligible work by queued status, workspace/scope, capabilities, lease expiration, authority, and one-active-lease-per-session rules.
- MUST generate raw claim tokens only for successful synchronous claim responses and persist only the hash.
- MUST fence heartbeat, release, complete, and fail operations with the current raw claim token.
- MUST recover expired or abandoned leases without creating duplicate active ownership.
- MUST co-emit typed task-run hooks through the task_03 bridge for claim, heartbeat, release, completion, failure, and lease expiration when externally meaningful.
</requirements>

## Subtasks
- [ ] 8.1 Add `ClaimCriteria`, `ClaimResult`, and lease mutation interfaces to `internal/task`.
- [ ] 8.2 Implement transactional pending-run selection and claim update in `internal/store/globaldb`.
- [ ] 8.3 Add token generation, hashing, and constant-time token verification helpers.
- [ ] 8.4 Fence heartbeat/release/complete/fail operations by token hash and lease state.
- [ ] 8.5 Add deterministic expired-lease recovery and one-active-lease-per-session enforcement.
- [ ] 8.6 Add concurrency, recovery, hook, and regression tests covering all lease state transitions.

## Implementation Details
Keep the transaction small: select the best eligible run, update its claim metadata, and return the claimed run with the raw token to the caller. Avoid a scheduler-owned queue and avoid generic lock managers.

Use injected clocks or store-level time providers for tests. Race tests should coordinate goroutines with channels/barriers and prove that exactly one claimant wins.

### Relevant Files
- `internal/task/interfaces.go` - service/store contracts for claim-next and lease mutations.
- `internal/task/manager.go` - domain-level claim, heartbeat, release, complete, and fail semantics.
- `internal/task/events.go` - task-domain events for lease lifecycle.
- `internal/store/globaldb/global_db_task*.go` - transactional claim and lease persistence.
- `internal/store/globaldb/*_test.go` - SQLite claim and concurrency tests.
- `internal/hooks/*` - typed hook dispatch payloads from task_03.
- `.resources/paperclip/cli/src/commands/heartbeat-run.ts` - reference for heartbeat command semantics and failure modes.
- `.resources/hermes/cron/scheduler.py` - reference for lease recovery and retry orchestration boundaries.
- `.resources/hermes/agent/retry_utils.py` - reference for retry classification without swallowing root causes.

### Dependent Files
- `internal/api/udsapi/*` - task_09 exposes these operations to agents.
- `internal/daemon/task_runtime.go` - task_10/task_11 use enqueue and recovery behavior.
- `internal/scheduler/*` - task_11 consumes expired-lease recovery without claiming work.

### Related ADRs
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - claim and fencing invariants.
- [ADR-004: Coordinator-Agent Plus Mechanical Scheduler](adrs/adr-004.md) - scheduler must not become the claimant.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - hook emission boundaries.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - operator and agent paths share the same claim model.

## Deliverables
- Transactional `ClaimNextRun(criteria)` implementation.
- Fenced heartbeat, release, complete, and fail operations.
- Expired lease recovery without duplicate ownership.
- Unit tests with 80%+ coverage for lease state machine helpers **(REQUIRED)**.
- SQLite integration and race tests proving exactly-once active claims **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] Claim criteria validation rejects missing claimer session, invalid workspace scope, and unsupported capability names.
  - [ ] Token hashing and verification never compare raw token strings directly.
  - [ ] Heartbeat extends only the current lease and rejects stale, missing, expired, or mismatched tokens.
  - [ ] Complete/fail/release require the current token and clear or preserve fields according to the TechSpec.
  - [ ] One-active-lease-per-session rejects a second claim until the current run is completed, failed, released, or expired.
- Integration tests:
  - [ ] Concurrent claim attempts against one queued run result in exactly one successful claimant.
  - [ ] Capability matching returns only runs whose required capabilities are all satisfied.
  - [ ] Expired leases become claimable again, while unexpired leases are skipped.
  - [ ] Boot recovery identifies stale leases and emits the configured task-run events/hooks once.
  - [ ] Manual operator-enqueued runs and agent-created runs are both claimable through the same primitive.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- No code path claims or mutates active task-run ownership without lease fencing.
- Scheduler/coordinator tasks can build on a deterministic, transactionally safe claim service.
