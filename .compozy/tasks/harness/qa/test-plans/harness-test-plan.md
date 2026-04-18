# Harness Runtime — Test Plan

**Feature:** Harness runtime architecture
**Version:** 1.0
**Date:** 2026-04-18
**Status:** Active planning baseline
**QA Output Path:** `.compozy/tasks/harness`

---

## 1. Executive Summary

This plan defines the QA strategy for the harness runtime slice delivered by
tasks 01 through 08. The harness work changed daemon-owned policy resolution,
startup prompt assembly, prompt augmentation, synthetic prompt persistence,
transcript trust boundaries, detached task-runtime mapping, completion-driven
reentry, and lifecycle observability.

The QA goal is not generic smoke confidence. The goal is to prove, with
operator-observable evidence, that each critical seam behaves correctly through
the real daemon/runtime surfaces that `task_10` will execute.

### Objectives

1. Prove startup behavior is resolved from durable session context plus turn
   origin, with explicit section selection and deterministic network overlay
   behavior.
2. Prove ordered prompt augmentation preserves canonical stored input while the
   dispatched prompt reflects augmenter output, budget handling, and failure
   semantics.
3. Prove daemon-originated synthetic turns remain distinct from human input in
   persistence, transcript rendering, hooks, and extension-host replay.
4. Prove detached harness work uses the existing task/runtime substrate with
   durable metadata, idempotency, and recovery semantics.
5. Prove detached completion either wakes the owning session through the
   synthetic path or records an observable drop/silent outcome without hidden
   behavior.
6. Prove harness lifecycle events remain visible through `event_summaries` and
   transport parity surfaces, including restart/recovery scenarios.

---

## 2. Scope

### In Scope

- `task_01` / Workstream 1: harness context resolution and policy matrix
- `task_02` / Workstream 2: startup section selection and explicit network
  startup overlay
- `task_03` / Workstream 3: ordered prompt augmentation composite
- `task_04` / Workstream 4: synthetic prompt submission and persistence
- `task_05` / Workstream 4: transcript, hooks, and extension-host synthetic
  trust boundaries
- `task_06` / Workstream 5: detached harness work on `task` / `task_run`
  metadata
- `task_07` / Workstream 4 + 5: completion-to-reentry bridge, queueing, drop
  rules, and dedupe
- `task_08` / Workstream 6: event-summary observability and HTTP/UDS parity
- Execution evidence rules for `task_10`, including issue-file creation and
  final verification reporting

### Out of Scope

- New product behavior outside the approved harness TechSpec
- UI redesign or browser-only validation unrelated to harness runtime behavior
- Refactoring task/runtime APIs beyond what is already implemented
- Ad hoc scripts that bypass the repo-supported daemon/runtime integration
  lanes
- Production bug fixes during planning, unless the documentation task itself
  reveals a concrete discrepancy that must be recorded as a `BUG-*` artifact

---

## 3. Traceability Matrix

| Harness Area | Source of Truth | Primary Test Cases | Priority | Suite Coverage |
| --- | --- | --- | --- | --- |
| Context resolution matrix | Workstream 1, `task_01.md`, ADR-001 | `TC-INT-001` | P0 | Smoke, full |
| Startup section selection and network overlay | Workstream 2, `task_02.md`, ADR-002 | `TC-INT-001` | P0 | Smoke, targeted, full |
| Ordered augmentation + stored-input invariant | Workstream 3, `task_03.md`, ADR-002 | `TC-INT-002` | P0 | Smoke, targeted, full |
| Synthetic prompt persistence + queueing | Workstream 4, `task_04.md`, ADR-001/003 | `TC-INT-003` | P0 | Targeted, full |
| Transcript / hook / extension trust boundary | Workstream 4, `task_05.md`, ADR-002/003 | `TC-INT-004` | P0 | Targeted, full |
| Detached task/runtime mapping + idempotency | Workstream 5, `task_06.md`, ADR-003 | `TC-INT-005` | P0 | Targeted, full |
| Completion-to-reentry bridge + drop policy | Workstream 4 and 5, `task_07.md`, ADR-003 | `TC-INT-006` | P0 | Smoke, targeted, full |
| Observability + transport parity | Workstream 6, `task_08.md`, ADR-002/003/004 | `TC-INT-007` | P0 | Smoke, targeted, full |
| Restart / recovery / duplicate-protection | Workstream 5 and 6, `task_06.md` through `task_08.md` | `TC-REG-001` | P1 | Targeted, full |

Every P0/P1 case above names the exact workstream and task file it proves.

---

## 4. Test Strategy

### 4.1 Core Execution Rule

`task_10` must execute harness QA through the repository’s normal proof paths:

- repo verification gates
- real daemon/runtime integration tests
- transcript and observe endpoints
- HTTP/UDS parity surfaces
- task-runtime boot/recovery flows

Do not replace these with one-off shell scripts as final proof.

### 4.2 Execution Lanes

| Lane | Surface | Purpose | Required Evidence |
| --- | --- | --- | --- |
| Baseline gate | repo root | Establish that the branch is in a valid starting state before runtime-specific QA | command, exit code, key output lines |
| Daemon/runtime lane | `internal/daemon`, `internal/session` | Validate startup selection, augmentation, detached work, and completion-to-reentry semantics | session ids, task ids, task-run ids, persisted event types, ordered summaries |
| Transcript/hook/extension lane | `internal/transcript`, `internal/session`, `internal/extension` | Validate synthetic trust boundaries end to end | transcript excerpts, hook class, extension turn-id evidence |
| Observability/parity lane | `internal/observe`, `internal/api/httpapi`, `internal/api/udsapi` | Validate `event_summaries` ordering and transport parity | HTTP/UDS responses showing identical harness event sequence |
| Recovery lane | daemon boot/recovery + task runtime | Validate restart, dedupe, and orphan-recovery semantics | pre/post restart state, duplicate-protection evidence |

### 4.3 Suggested Repo-Supported Anchors

The following existing tests are the strongest execution anchors for `task_10`
and should be treated as the preferred runtime lanes when they match the case:

- `internal/daemon/harness_context_integration_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/daemon/task_runtime_test.go`
- `internal/session/manager_integration_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`
- `internal/observe/observer_test.go`
- `internal/extension/host_api_test.go`

The external references used to widen scenario breadth are:

- OpenClaw QA lanes: artifact discipline and suite structure
- Hermes checkpoint-resumption: restart/recovery integrity
- OpenFang API reference: externally inspectable async/eventful runtime behavior
- Claude Code task framework / main-session task: background completion and
  foreground reentry semantics

---

## 5. Environment Matrix

| Environment | Surface | Minimum Requirement | Notes |
| --- | --- | --- | --- |
| Local dev / CI | repo root | `make verify` must pass | blocking completion gate |
| Integration lane | Go runtime with integration tags | `make test-integration` or equivalent targeted integration commands | required by Workstream 6 verification expectations |
| HTTP transport | daemon HTTP API | harness lifecycle events and transcript endpoints readable | used for parity evidence |
| UDS transport | daemon UDS API | same event and transcript visibility as HTTP | parity must match HTTP |
| SQLite-backed persistence | temp AGH home / temp workspaces | durable `task`, `task_run`, session events, and `event_summaries` available | required for detached/recovery evidence |
| Optional browser evidence | only if task_10 touches a user-visible surface directly impacted by harness | screenshots in `qa/screenshots/` | not required for daemon/runtime proof |

---

## 6. Entry Criteria

All of the following must be true before `task_10` starts live execution:

1. `task_01` through `task_08` are implemented on the current branch.
2. The QA artifact root remains `.compozy/tasks/harness/qa/`.
3. The test plan, regression suite, and all P0/P1 case files from this task are
   present and readable.
4. The execution branch can run repo-defined verification commands from the
   workspace root.
5. The execution lane can create isolated AGH homes/workspaces for runtime and
   recovery scenarios.
6. Observe/query surfaces are available so harness lifecycle evidence can be
   collected instead of inferred from code alone.
7. The branch owner agrees that any QA-discovered bug must be fixed at the
   production root cause, not hidden by weaker tests or manual exceptions.

---

## 7. Exit Criteria

| Criterion | Threshold |
| --- | --- |
| P0 runtime cases | 100% pass |
| P1 runtime cases | 90%+ pass, with no unresolved High/Critical bug |
| Baseline repo gate | fresh `make verify` pass |
| Integration proof | fresh harness-relevant integration lane evidence recorded |
| Observability evidence | required harness event-summary sequence captured for executed cases |
| Transport parity | HTTP and UDS evidence agree for harness-visible flows |
| Bug handling | every discovered issue has either a root-cause fix or a `BUG-*` artifact tied to the originating test case |
| Final report | `.compozy/tasks/harness/qa/verification-report.md` written with a clear PASS / FAIL / CONDITIONAL verdict |

Any failure of the following is release-blocking for the harness slice:

- wrong startup section selection for session/channel context
- stored input overwritten by augmentation
- synthetic turns persisted or replayed as user input
- detached completion emits duplicate synthetic wake for the same `task_run`
- missing or divergent harness lifecycle evidence across HTTP/UDS surfaces

---

## 8. Risk Assessment

| Risk | Probability | Impact | Mitigation | Primary Cases |
| --- | --- | --- | --- | --- |
| Policy matrix drift causes wrong startup behavior | Medium | Critical | Validate channel vs non-channel resolution and resume stability with observable summaries | `TC-INT-001`, `TC-INT-007` |
| Augmentation silently mutates canonical stored input | Medium | Critical | Compare persisted input to dispatched prompt and assert explicit failure semantics | `TC-INT-002` |
| Synthetic events lose trust boundary and look human-originated | Medium | Critical | Inspect persistence, transcript, hook input class, and extension replay boundaries | `TC-INT-003`, `TC-INT-004` |
| Detached harness work drifts away from `task` / `task_run` substrate | Low | High | Validate metadata persistence, idempotency, scope mapping, and normal query surfaces | `TC-INT-005` |
| Completion-to-reentry duplicates or drops wakeups invisibly | Medium | Critical | Validate wake, silent drop, FIFO, and post-restart dedupe with event evidence | `TC-INT-006`, `TC-REG-001` |
| Event summaries exist in one transport but not the other | Medium | High | Compare HTTP and UDS ordered harness event sequences on the same scenario | `TC-INT-007` |
| Recovery after restart replays stale completion incorrectly | Medium | High | Execute checkpoint/resume style recovery scenario and compare pre/post restart state | `TC-REG-001` |

---

## 9. Evidence Contract

All planning and execution artifacts must remain under:

```text
.compozy/tasks/harness/qa/
├── test-plans/
├── test-cases/
├── issues/
├── screenshots/
└── verification-report.md
```

### `verification-report.md` Required Sections

1. Baseline repository gate
2. Smoke suite results
3. P0 execution matrix
4. P1 execution matrix
5. Event-summary and transport-parity evidence
6. Recovery / restart evidence
7. Bugs discovered and fixes applied
8. Final verdict

### Required Evidence Per Executed Test Case

- exact command or runtime lane used
- execution timestamp
- pass/fail result
- session id / task id / task-run id when relevant
- the operator-observable artifact that proves the claim:
  - transcript excerpt
  - stored event payload excerpt
  - `event_summaries` sequence
  - HTTP/UDS parity response excerpt
  - restart/recovery state comparison
- link to `BUG-*` artifact if the case fails

### Artifact Rules

- Use `qa/issues/BUG-*.md` only when execution exposes a concrete discrepancy.
- Use `qa/screenshots/` only when browser or other visual evidence is actually
  relevant to the changed branch.
- Do not change the QA output root in `task_10`.

---

## 10. Timeline and Deliverables

| Phase | Deliverable |
| --- | --- |
| Planning (`task_09`) | this test plan, runtime test cases, regression suite |
| Execution (`task_10`) | `verification-report.md`, optional `BUG-*` files, optional screenshots |
| Completion gate | fresh verification evidence plus updated task tracking |

---

## 11. Handoff to Task 10

`task_10` should consume these artifacts unchanged:

- `qa/test-plans/harness-test-plan.md`
- `qa/test-plans/harness-regression.md`
- `qa/test-cases/TC-INT-001.md`
- `qa/test-cases/TC-INT-002.md`
- `qa/test-cases/TC-INT-003.md`
- `qa/test-cases/TC-INT-004.md`
- `qa/test-cases/TC-INT-005.md`
- `qa/test-cases/TC-INT-006.md`
- `qa/test-cases/TC-INT-007.md`
- `qa/test-cases/TC-REG-001.md`

The first execution priority is:

1. Baseline repo gate
2. `TC-INT-001`
3. `TC-INT-002`
4. `TC-INT-006`
5. `TC-INT-007`

If any of those fail, stop and fix before moving into the deeper targeted and
recovery lanes.
