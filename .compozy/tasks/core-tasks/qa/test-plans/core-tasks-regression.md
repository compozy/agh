# AGH Core Tasks -- Regression Test Suite

**Feature:** Core Tasks and Subtasks
**Version:** v1
**Date:** 2026-04-14
**Status:** Active

---

## Table of Contents

1. [Purpose and Scope](#1-purpose-and-scope)
2. [Smoke Suite](#2-smoke-suite)
3. [Priority Classification](#3-priority-classification)
4. [Targeted Regression by Change Area](#4-targeted-regression-by-change-area)
5. [Full Regression Suite](#5-full-regression-suite)
6. [Sanity Suite](#6-sanity-suite)
7. [Pass/Fail Criteria](#7-passfail-criteria)
8. [Execution Checklist](#8-execution-checklist)

---

## 1. Purpose and Scope

This document defines the regression testing strategy for the AGH Core Tasks feature. It covers all test case ranges:

- **TC-FUNC-001 to TC-FUNC-030:** Functional (task CRUD, dependencies, run lifecycle, cancellation, limits, cold-start)
- **TC-INT-001 to TC-INT-015:** Integration (API HTTP/UDS, CLI, session bridge, automation, extension, network)
- **TC-SEC-001 to TC-SEC-008:** Security (identity spoofing, authorization, injection, payload limits)
- **TC-PERF-001 to TC-PERF-006:** Performance (throughput, graph operations, propagation, queries)
- **SMOKE-001 to SMOKE-010:** Smoke (daemon boot, basic CRUD, run lifecycle, CLI, observe)

The regression suite ensures that changes to any package within the Core Tasks scope do not introduce regressions in task coordination, execution, security, or observability.

---

## 2. Smoke Suite

**Duration:** 15-20 minutes
**Frequency:** Daily, and before any detailed regression testing
**Gate rule:** If ANY smoke test fails, STOP immediately. Do not proceed to targeted or full regression.

### Smoke Test IDs

| ID | Area | Description |
|----|------|-------------|
| SMOKE-001 | Daemon boot | Daemon starts successfully with task schema migrated and TaskManager initialized |
| SMOKE-002 | Task create | Create a global task via HTTP API; verify 201 response and persisted record |
| SMOKE-003 | Task list/get | List tasks returns the created task; get by ID returns full detail with correct fields |
| SMOKE-004 | Task update | Patch mutable fields (title, description, owner); verify updated values |
| SMOKE-005 | Child task | Create a child task under the global task; verify parent linkage |
| SMOKE-006 | Run enqueue | Enqueue a run for the task; verify queued status |
| SMOKE-007 | Run lifecycle | Progress a run through claim, start, complete; verify terminal state |
| SMOKE-008 | CLI basic | Execute `agh task create`, `agh task list`, `agh task get` via CLI; verify output |
| SMOKE-009 | Cancellation basic | Cancel a task with a queued run; verify both task and run reach cancelled state |
| SMOKE-010 | Observe health | Query task health/metrics endpoint; verify queue depth and task count projections appear |

### Smoke Execution Notes

- Execute in the order listed (SMOKE-001 through SMOKE-010).
- Each test should be self-contained and leave the system in a clean state for the next.
- If SMOKE-001 fails, the daemon cannot start -- no further testing is meaningful.
- If SMOKE-002 through SMOKE-005 fail, task CRUD is broken -- run lifecycle tests will also fail.
- SMOKE-010 depends on observe projections; failure here indicates a wiring or projection issue.

---

## 3. Priority Classification

### P0 -- Critical Path (must all pass for release)

These tests cover the core task lifecycle, data integrity, and security boundaries. Any P0 failure is a release blocker.

| TC-ID | Area | Rationale |
|-------|------|-----------|
| TC-FUNC-001 | Task create (global scope) | Foundational CRUD; all other tests depend on task creation |
| TC-FUNC-002 | Task create (workspace scope) | Scope enforcement is a core invariant |
| TC-FUNC-003 | Task get by ID | Read path correctness; underpins all inspection flows |
| TC-FUNC-004 | Task list with filters | Query correctness across scope, status, owner, workspace |
| TC-FUNC-005 | Task update mutable fields | Mutation rules are a contract boundary |
| TC-FUNC-006 | Immutable field rejection | Identity and structural integrity protection |
| TC-FUNC-007 | Child task creation | Hierarchy is a first-class coordination mechanism |
| TC-FUNC-008 | Dependency add | Dependency graph is required for blocked/ready reconciliation |
| TC-FUNC-009 | Dependency remove | Must not leave orphan edges or corrupt graph state |
| TC-FUNC-010 | Cycle detection | Transactional cycle rejection prevents infinite loops |
| TC-FUNC-011 | Graph depth limit | Guardrail enforcement; unbounded depth corrupts reconciliation |
| TC-FUNC-012 | Dependency edge limit | Guardrail enforcement; unbounded edges degrade performance |
| TC-FUNC-013 | Direct children limit | Guardrail enforcement; unbounded children degrade queries |
| TC-FUNC-014 | Run enqueue | Queue-first execution is the only valid entry point |
| TC-FUNC-015 | Run claim | Claim transition is the only path from queued to active |
| TC-FUNC-016 | Run start (dedicated session) | Dedicated-session default is a core architectural decision |
| TC-FUNC-017 | Run complete | Terminal success path; reconciles task to completed |
| TC-FUNC-018 | Run fail | Terminal failure path; reconciles task to failed |
| TC-FUNC-019 | Run cancel | Cooperative cancellation of individual runs |
| TC-FUNC-020 | Task cancel with propagation | Tree cancellation propagates to descendants and active runs |
| TC-FUNC-021 | Forced stop escalation | Cooperative-then-forced model prevents leaked sessions |
| TC-FUNC-022 | Task status reconciliation | Manager-owned reconciliation from deps and runs |
| TC-FUNC-023 | Cold-start recovery (claimed runs) | Orphaned claimed runs re-queued on boot |
| TC-FUNC-024 | Cold-start recovery (running runs) | Orphaned running runs failed on boot |
| TC-SEC-001 | Identity spoofing rejection | Payload-supplied identity must be ignored |
| TC-SEC-002 | Unauthenticated write rejection | No anonymous task writes |
| TC-SEC-003 | Authorization enforcement (HTTP) | HTTP routes enforce principal resolution |
| TC-SEC-004 | Extension capability check | Extensions without task capability are rejected |
| TC-SEC-005 | Network peer validation | Network writes require authenticated peer context |
| TC-SEC-006 | Injection resistance | Malformed inputs do not corrupt stored data |
| TC-SEC-007 | Payload size enforcement | Oversize metadata/result/event payloads rejected before persistence |
| TC-SEC-008 | Extension origin immutability | Extension-supplied origin fields cannot override server-derived values |

### P1 -- Important (90%+ must pass)

These tests cover integration surfaces, advanced lifecycle flows, and cross-package correctness. P1 failures require documented workarounds if not fixed before release.

| TC-ID | Area | Rationale |
|-------|------|-----------|
| TC-FUNC-025 | Attach-session for resume/handoff | Explicit attach is a secondary but supported execution path |
| TC-FUNC-026 | Attach-session single-assignment | One session per live run; prevents resource contention |
| TC-FUNC-027 | Attach-session state gating | Attachment only valid in claimed/starting states |
| TC-FUNC-028 | Idempotency key deduplication | Multi-writer ingress safety for non-human callers |
| TC-FUNC-029 | Audit event persistence | Lifecycle actions produce immutable audit records |
| TC-FUNC-030 | Cold-start recovery (starting runs) | Starting runs with dead sessions failed on boot |
| TC-INT-001 | HTTP task CRUD | HTTP transport correctness for task operations |
| TC-INT-002 | HTTP run lifecycle | HTTP transport correctness for run operations |
| TC-INT-003 | HTTP response codes/envelopes | Transport consistency; stable error contract |
| TC-INT-004 | HTTP filter parity | Query filters work identically to UDS |
| TC-INT-005 | UDS task and run parity | UDS exposes the same operations as HTTP |
| TC-INT-006 | CLI task create/list/get | CLI-to-daemon round-trip for task CRUD |
| TC-INT-007 | CLI run lifecycle | CLI-to-daemon round-trip for run operations |
| TC-INT-008 | CLI flag validation | Invalid flag combinations rejected before daemon call |
| TC-INT-009 | Session bridge create | Run start creates dedicated session through injected bridge |
| TC-INT-010 | Session bridge stop | Cancellation stop request flows through bridge to session |
| TC-INT-011 | Automation direct task create | Automation creates tasks with correct origin |
| TC-INT-012 | Automation non-overlap | Task-backed automation does not duplicate execution state |
| TC-INT-013 | Extension host API task flow | Extension creates and runs tasks through capability-checked host API |
| TC-INT-014 | Network peer task create | Network peer creates channel-bound task |
| TC-INT-015 | Network channel mismatch rejection | Channel-bound task rejects mismatched ingress |

### P2 -- Supplementary (informational, not release-blocking)

These tests measure system behavior under load and at scale. Failures are tracked but do not block release.

| TC-ID | Area | Rationale |
|-------|------|-----------|
| TC-PERF-001 | Task creation throughput | Baseline throughput measurement |
| TC-PERF-002 | Graph operation performance | Cycle detection and depth checks at scale |
| TC-PERF-003 | Cancellation propagation latency | Tree cancellation timing under deep hierarchies |
| TC-PERF-004 | List query performance | Filter queries under high task count |
| TC-PERF-005 | Observe projection throughput | Metric and health projection under event load |
| TC-PERF-006 | Queue depth query performance | Queue depth calculation under high run count |

---

## 4. Targeted Regression by Change Area

When a change is scoped to a specific package, run the smoke suite first, then only the test cases mapped to the changed package(s). If a change spans multiple packages, union the test sets.

| Changed Package | Must Re-Run | Rationale |
|-----------------|-------------|-----------|
| `internal/task/` | TC-FUNC-001 to TC-FUNC-030, TC-SEC-001 to TC-SEC-008 | Core domain; all functional and security tests depend on task types, validation, lifecycle, and manager logic |
| `internal/store/globaldb/` | TC-FUNC-001 to TC-FUNC-013, TC-PERF-001 to TC-PERF-006 | Persistence layer; CRUD, graph limits, query filters, and performance all depend on store correctness |
| `internal/api/httpapi/` | TC-INT-001 to TC-INT-005, TC-SEC-003 to TC-SEC-007 | HTTP transport; route registration, response codes, payload validation, and authz enforcement |
| `internal/api/udsapi/` | TC-INT-005 | UDS transport; must maintain parity with HTTP |
| `internal/api/core/` | TC-INT-001 to TC-INT-005, TC-SEC-003, TC-SEC-006, TC-SEC-007 | Shared handlers; validation, error mapping, and payload conversion used by both transports |
| `internal/cli/` | TC-INT-006 to TC-INT-008 | CLI commands; flag parsing, UDS communication, output formatting |
| `internal/session/` | TC-INT-009 to TC-INT-010, TC-FUNC-016 to TC-FUNC-020 | Session bridge; dedicated-session creation, attach, stop, and cancellation flows |
| `internal/automation/` | TC-INT-011 to TC-INT-012 | Automation integration; direct task creation, non-overlap with automation runs |
| `internal/extension/` | TC-INT-013, TC-SEC-004, TC-SEC-008 | Extension host API; capability checks, identity derivation, origin immutability |
| `internal/network/` | TC-INT-014 to TC-INT-015, TC-SEC-005 | Network ingress; peer validation, channel binding, mismatch rejection |
| `internal/observe/` | TC-PERF-005 to TC-PERF-006, SMOKE-010 | Observe projections; health queries, metrics, queue depth calculations |
| `internal/daemon/` | SMOKE-001, TC-FUNC-023, TC-FUNC-024, TC-FUNC-030 | Composition root; boot sequence, cold-start recovery, service wiring |

### Cross-Cutting Change Rules

- **Schema migration changes** (`internal/store/globaldb/` migration files): Run the full `internal/store/globaldb/` set plus `internal/task/` set.
- **Contract/payload changes** (`internal/api/contract/`): Run TC-INT-001 to TC-INT-008, TC-SEC-003, TC-SEC-006, TC-SEC-007.
- **Domain type changes** (`internal/task/` types/enums): Run everything in `internal/task/` column plus all integration tests.
- **Boot sequence changes** (`internal/daemon/boot.go`): Run all smoke tests plus cold-start recovery tests.

---

## 5. Full Regression Suite

**Duration:** 2-3 hours
**Frequency:** Before every release, after major refactors, weekly CI gate

### Execution Order

The full regression runs in strict priority order. If a higher-priority group fails, evaluate whether to continue based on the pass/fail criteria in Section 7.

#### Phase 1: Smoke Gate (15-20 min)

Run all smoke tests first. If any fail, STOP.

| Order | Test IDs |
|-------|----------|
| 1 | SMOKE-001 |
| 2 | SMOKE-002 |
| 3 | SMOKE-003 |
| 4 | SMOKE-004 |
| 5 | SMOKE-005 |
| 6 | SMOKE-006 |
| 7 | SMOKE-007 |
| 8 | SMOKE-008 |
| 9 | SMOKE-009 |
| 10 | SMOKE-010 |

#### Phase 2: P0 Critical Path (45-60 min)

If any P0 test fails, the regression is FAILED. Log the failure, investigate root cause, and do not proceed until the failure is understood.

| Order | Test IDs | Area |
|-------|----------|------|
| 11-13 | TC-FUNC-001 to TC-FUNC-003 | Task create and read |
| 14-15 | TC-FUNC-004, TC-FUNC-005 | Task list and update |
| 16 | TC-FUNC-006 | Immutable field rejection |
| 17-19 | TC-FUNC-007 to TC-FUNC-009 | Hierarchy and dependencies |
| 20-22 | TC-FUNC-010 to TC-FUNC-012 | Graph limits and cycle detection |
| 23 | TC-FUNC-013 | Children limit |
| 24-26 | TC-FUNC-014 to TC-FUNC-016 | Run enqueue, claim, start |
| 27-29 | TC-FUNC-017 to TC-FUNC-019 | Run complete, fail, cancel |
| 30-31 | TC-FUNC-020, TC-FUNC-021 | Cancellation propagation and forced stop |
| 32 | TC-FUNC-022 | Status reconciliation |
| 33-34 | TC-FUNC-023, TC-FUNC-024 | Cold-start recovery |
| 35-42 | TC-SEC-001 to TC-SEC-008 | Full security suite |

#### Phase 3: P1 Integration and Advanced Lifecycle (45-60 min)

P1 failures are tracked. The suite continues but failures are flagged for investigation.

| Order | Test IDs | Area |
|-------|----------|------|
| 43-45 | TC-FUNC-025 to TC-FUNC-027 | Attach-session flows |
| 46-48 | TC-FUNC-028 to TC-FUNC-030 | Idempotency, audit, cold-start (starting) |
| 49-53 | TC-INT-001 to TC-INT-005 | HTTP and UDS transport |
| 54-56 | TC-INT-006 to TC-INT-008 | CLI commands |
| 57-58 | TC-INT-009 to TC-INT-010 | Session bridge |
| 59-60 | TC-INT-011 to TC-INT-012 | Automation integration |
| 61 | TC-INT-013 | Extension integration |
| 62-63 | TC-INT-014 to TC-INT-015 | Network integration |

#### Phase 4: P2 Performance (30-45 min)

P2 failures are informational. Record baseline numbers and compare against previous runs.

| Order | Test IDs | Area |
|-------|----------|------|
| 64 | TC-PERF-001 | Task creation throughput |
| 65 | TC-PERF-002 | Graph operation performance |
| 66 | TC-PERF-003 | Cancellation propagation latency |
| 67 | TC-PERF-004 | List query performance |
| 68 | TC-PERF-005 | Observe projection throughput |
| 69 | TC-PERF-006 | Queue depth query performance |

#### Phase 5: Exploratory Testing (15-30 min)

After all scripted tests pass, allocate time for exploratory testing focused on:

1. **Concurrent writer stress:** Multiple simultaneous task creates, updates, and cancellations from different writer surfaces (HTTP, CLI, automation).
2. **Deep hierarchy edge cases:** Create tasks at max depth (8), add max dependencies (32), then cancel the root -- verify propagation completes cleanly.
3. **Session bridge failure modes:** Simulate bridge timeouts and session creation failures during run start -- verify the run transitions to failed, not stuck.
4. **Cold-start under load:** Seed multiple in-flight runs, restart the daemon, verify all orphaned runs are correctly reconciled before new traffic is accepted.
5. **Network channel lifecycle:** Create a channel-bound task, invalidate the channel configuration, attempt a new run -- verify stale-channel rejection and audit trail.
6. **Extension capability revocation:** Grant task capability, create a task, revoke capability, attempt update -- verify clean rejection.

---

## 6. Sanity Suite

**Duration:** 10 minutes
**Use case:** Post-hotfix verification when the fix scope is known and narrow

### Procedure

1. Run SMOKE-001 through SMOKE-005 (daemon boot + basic CRUD). If any fail, STOP.
2. Run the specific TC-ID(s) that validate the hotfix. The fix author must identify these.
3. If the hotfix touches a security boundary, additionally run the relevant TC-SEC test(s).

### Examples

| Hotfix Area | Sanity Set |
|-------------|------------|
| Task creation validation bug | SMOKE-001 to SMOKE-005, TC-FUNC-001, TC-FUNC-002 |
| Run lifecycle state transition bug | SMOKE-001 to SMOKE-005, SMOKE-006, SMOKE-007, TC-FUNC-014 to TC-FUNC-019 |
| Cancellation propagation fix | SMOKE-001 to SMOKE-005, SMOKE-009, TC-FUNC-020, TC-FUNC-021 |
| Cold-start recovery fix | SMOKE-001 to SMOKE-005, TC-FUNC-023, TC-FUNC-024, TC-FUNC-030 |
| HTTP API response code fix | SMOKE-001 to SMOKE-005, TC-INT-001 to TC-INT-003 |
| CLI command parsing fix | SMOKE-001 to SMOKE-005, SMOKE-008, TC-INT-006 to TC-INT-008 |
| Security vulnerability fix | SMOKE-001 to SMOKE-005, all TC-SEC-001 to TC-SEC-008 |
| Extension integration fix | SMOKE-001 to SMOKE-005, TC-INT-013, TC-SEC-004, TC-SEC-008 |
| Network channel fix | SMOKE-001 to SMOKE-005, TC-INT-014, TC-INT-015, TC-SEC-005 |
| Observe/metrics fix | SMOKE-001 to SMOKE-005, SMOKE-010, TC-PERF-005, TC-PERF-006 |
| Dependency graph fix | SMOKE-001 to SMOKE-005, TC-FUNC-008 to TC-FUNC-013 |

---

## 7. Pass/Fail Criteria

### PASS

All of the following must be true:

- All SMOKE tests pass (SMOKE-001 to SMOKE-010).
- All P0 tests pass (TC-FUNC-001 to TC-FUNC-024, TC-SEC-001 to TC-SEC-008).
- 90% or more of P1 tests pass (at most 2 of 21 P1 tests may fail).
- No critical bugs discovered (data loss, unrecoverable state, security vulnerability).
- No security vulnerabilities found in any test tier.

### FAIL

Any of the following is an automatic FAIL:

- Any SMOKE test fails.
- Any P0 test fails.
- A critical bug is found: data loss, task state corruption, orphaned sessions after cancellation, identity spoofing succeeds, unauthenticated write succeeds.
- A security vulnerability is confirmed in any test tier.
- Cold-start recovery leaves orphaned runs in non-terminal state.

### CONDITIONAL PASS

The regression receives a conditional pass when:

- All SMOKE and P0 tests pass.
- Between 1 and 2 P1 tests fail, AND each failure has a documented workaround that does not impact core task lifecycle.
- P2 performance regressions are within 20% of the established baseline.
- No security or data integrity issues exist.

A conditional pass requires sign-off from the tech lead with:
- A tracking issue for each P1 failure.
- A written workaround for each P1 failure.
- Confirmation that the failures do not cascade into P0 areas.

---

## 8. Execution Checklist

### Before Regression

- [ ] Verify the daemon builds cleanly (`make build` succeeds).
- [ ] Verify `make verify` passes (fmt, lint, test, build).
- [ ] Confirm the test environment has a clean database (no leftover state from previous runs).
- [ ] Confirm the branch under test is identified and the commit SHA is recorded.
- [ ] Confirm the test runner has access to both HTTP and UDS daemon interfaces.
- [ ] Confirm the `agh` CLI binary is built from the same commit as the daemon.
- [ ] Record the baseline performance numbers from the previous regression run (for P2 comparison).
- [ ] Verify no known environment issues (disk space, port conflicts, stale UDS sockets).

### During Regression

- [ ] Execute phases in strict order: Smoke, P0, P1, P2, Exploratory.
- [ ] Stop immediately on SMOKE failure -- do not proceed to later phases.
- [ ] On P0 failure: log the failure, capture daemon logs and database state, investigate root cause before deciding whether to continue.
- [ ] On P1 failure: log the failure, continue execution, flag for post-regression investigation.
- [ ] On P2 failure: log the measured values, compare against baseline, continue execution.
- [ ] Record the actual execution time for each phase.
- [ ] Capture daemon structured logs for the entire regression duration.
- [ ] For any exploratory finding, document: steps to reproduce, observed behavior, expected behavior, severity estimate.

### After Regression

- [ ] Compile results: total pass/fail/skip counts per priority tier.
- [ ] Apply pass/fail criteria from Section 7 to determine the overall result.
- [ ] For PASS: record the result with commit SHA, test duration, and any noteworthy observations.
- [ ] For CONDITIONAL PASS: file tracking issues for each P1 failure, document workarounds, obtain tech lead sign-off.
- [ ] For FAIL: file blocking issues for each P0 or SMOKE failure, escalate to the feature owner, and schedule a re-run after fixes land.
- [ ] Archive the full test results, daemon logs, and database snapshots for the run.
- [ ] Update the baseline performance numbers if P2 results improved or regressed significantly.
- [ ] Communicate the regression outcome to the team.

---

## Appendix A: Test Case Quick Reference

| Range | Count | Category | Priority Mix |
|-------|-------|----------|-------------|
| TC-FUNC-001 to TC-FUNC-024 | 24 | Functional (critical path) | P0 |
| TC-FUNC-025 to TC-FUNC-030 | 6 | Functional (advanced) | P1 |
| TC-INT-001 to TC-INT-015 | 15 | Integration | P1 |
| TC-SEC-001 to TC-SEC-008 | 8 | Security | P0 |
| TC-PERF-001 to TC-PERF-006 | 6 | Performance | P2 |
| SMOKE-001 to SMOKE-010 | 10 | Smoke | Gate |
| **Total** | **69** | | |

## Appendix B: Package-to-Task Mapping

For traceability, this maps each changed package to the implementation tasks that defined its Core Tasks behavior:

| Package | Implementation Tasks |
|---------|---------------------|
| `internal/task/` | Task 01, 04, 05 |
| `internal/store/globaldb/` | Task 02, 03 |
| `internal/session/` | Task 06 |
| `internal/api/core/` | Task 07 |
| `internal/api/httpapi/` | Task 08 |
| `internal/api/udsapi/` | Task 08 |
| `internal/cli/` | Task 09 |
| `internal/automation/` | Task 10 |
| `internal/extension/` | Task 11 |
| `internal/network/` | Task 12 |
| `internal/observe/` | Task 13 |
| `internal/daemon/` | Task 06 (boot recovery), wiring across all tasks |
