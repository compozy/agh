# AGH Core Tasks -- Comprehensive Test Plan

**Version:** 1.0
**Date:** 2026-04-14
**Author:** QA Engineering
**Status:** Draft
**Feature Branch:** `core-tasks`

---

## 1. Executive Summary

### 1.1 Objectives

This test plan defines the verification strategy for the AGH Core Tasks feature, which introduces a complete task coordination system into the AGH Agent Operating System. The feature spans 13 implemented components across domain logic, persistence, API transport, CLI, automation, extension, network, and observability layers.

The primary objectives are:

- Validate the correctness of the 7-state task lifecycle and 7-state run lifecycle state machines, including all legal and illegal transitions.
- Verify that identity invariants (server-derived `created_by` and `origin`, 6 actor kinds, 9 origin kinds) are enforced at every ingress surface and never accepted from client payloads.
- Confirm immutability guarantees for `scope`, `workspace_id`, `parent_task_id`, `created_by`, and `origin` fields after task creation.
- Ensure bounded resource limits (metadata 16KB, payload/result 64KB, hierarchy depth 8, dependencies 32/task, children 64/parent) are enforced and produce correct errors.
- Validate cycle detection in the dependency graph under concurrent write conditions via recursive CTE within `BEGIN IMMEDIATE` transactions.
- Confirm cascading cancellation propagates correctly through task hierarchies with cooperative-then-forced session stop semantics.
- Verify cold-start boot recovery correctly reclassifies orphaned claimed/starting/running runs.
- Validate idempotency key deduplication scoped to origin for non-human callers.
- Confirm that stale network channels block new runs while preserving task readability.
- Validate API parity between HTTP and UDS transports across all 18 endpoints.
- Verify CLI command coverage for the `agh task` command group.
- Confirm observability metrics (queue depth, stuck work, forced-stop audit, recovery totals) are accurate.

### 1.2 Key Risks

| Risk | Severity |
|------|----------|
| State machine transitions allow invalid paths under concurrent writes | Critical |
| Cycle detection fails under high-contention dependency insertion | Critical |
| Cascading cancellation leaves orphaned runs or sessions | Critical |
| Immutable field bypass through malformed API payloads | High |
| Boot recovery misclassifies live sessions as orphaned | High |
| Idempotency key collision across different origins | High |
| SQLite lock contention causes timeouts under concurrent task creation | Medium |
| Observability metrics drift from actual task state | Medium |

---

## 2. Scope Definition

### 2.1 In-Scope Features

| Component | Package | Description |
|-----------|---------|-------------|
| Domain types and validation | `internal/task/` | Task, TaskRun, TaskEvent, TaskDependency, TaskRunIdempotency types; all Validate() methods; ActorContext derivation; limit constants |
| Task Manager | `internal/task/` | Manager interface implementation: CreateTask, CreateChildTask, UpdateTask, CancelTask, AddDependency, RemoveDependency, EnqueueRun, ClaimRun, StartRun, AttachRunSession, CompleteRun, FailRun, CancelRun, RecoverRunOnBoot, GetTask, ListTasks, ListTaskRuns |
| SQLite persistence | `internal/store/globaldb/` | 5 tables (tasks, task_runs, task_dependencies, task_events, task_run_idempotency); CreateDependency with cycle detection CTE; immutable field enforcement at store layer |
| HTTP API | `internal/api/httpapi/` | 18 REST endpoints under `/tasks` and `/task-runs` groups |
| UDS API | `internal/api/udsapi/` | Mirror of all 18 REST endpoints for CLI IPC |
| Contract types | `internal/api/contract/` | Shared request/response payloads for both transports |
| CLI commands | `internal/cli/` | `agh task` group: list, create, get, update, cancel, child, dependency, run |
| Automation dispatch | `internal/automation/` | Direct and agent-mediated task creation through automation triggers/schedules |
| Extension host API | `internal/extension/` | Capability-checked task writes via extension runtime |
| Network ingress | `internal/network/` | Channel-bound peer task ingress with capability checks |
| Observability | `internal/observe/` | TaskSummary, TaskMetrics, TaskHealth queries; queue depth; stuck run detection; forced-stop and recovery audit |

### 2.2 Out-of-Scope Items

- Web UI (React SPA) rendering of task views -- covered by separate frontend test plan
- ACP subprocess protocol internals -- tested independently in `internal/acp/`
- Session lifecycle management beyond the `SessionExecutor` interface contract
- Memory/Skills/State layers (Phase 2 features)
- Agent network protocol (Phase 3 features)
- Database migration tooling (greenfield alpha -- no migrations exist)
- Load testing beyond the performance benchmarks defined in section 3.6
- Cross-platform binary builds (darwin/linux/windows)

---

## 3. Test Strategy

### 3.1 Unit Tests

**Scope:** Pure domain logic within `internal/task/`, validation functions, state machine transitions, normalization, helper functions.

**Approach:**
- Table-driven tests with `t.Run` subtests for all validation paths
- `t.Parallel()` for independent subtests
- Mock `Store` and `SessionExecutor` interfaces via test doubles
- Deterministic clocks via `WithManagerNow` and deterministic IDs via `WithIDGenerator`
- Cover every valid and invalid status transition in `allowsRunTransition`
- Cover every actor/origin pair validation in `validateActorOriginPair`
- Cover all 5 immutable field checks in `ValidateImmutableTaskFields`
- Cover size limit enforcement for metadata (16KB), payload (64KB), and result (64KB)
- Cover bounded count checks: hierarchy depth (8), dependencies (32), children (64)

**Coverage target:** 90%+ for `internal/task/` package.

### 3.2 Integration Tests

**Scope:** Manager operations against real SQLite persistence; full store round-trips; cycle detection under transaction isolation.

**Approach:**
- Build tag: `//go:build integration`
- Co-located with packages under test
- Real SQLite databases via `t.TempDir()`
- Full `GlobalDB` initialization with schema creation
- Test dependency cycle detection with the recursive CTE under `BEGIN IMMEDIATE`
- Test cascading cancellation through 3-level task hierarchies
- Test boot recovery with pre-seeded orphaned runs
- Test idempotency key deduplication across origins
- Test concurrent task/run creation under SQLite write contention
- `TestMain` for expensive one-time setup where needed
- Target execution time: <30s per package

**Coverage target:** 85%+ for `internal/store/globaldb/` task operations.

### 3.3 API Tests

**Scope:** HTTP and UDS endpoint behavior, request/response contract validation, error mapping.

**Approach:**
- Test both transports (HTTP via `httptest.Server`, UDS via real socket in `t.TempDir()`)
- Verify response status codes for success and all error categories (400, 404, 409, 413, 422)
- Verify JSON response shapes match `contract.*Payload` types
- Verify that `created_by` and `origin` fields are never accepted from request bodies
- Verify query parameter parsing and filter behavior for list endpoints
- Verify PATCH semantics: partial updates, no-op detection, immutable field rejection
- Confirm API parity: every endpoint that exists on HTTP also exists on UDS with identical behavior

**Coverage target:** 80%+ for `internal/api/httpapi/` and `internal/api/udsapi/` handler code.

### 3.4 CLI Tests

**Scope:** `agh task` command group flag parsing, output formatting, error handling.

**Approach:**
- Test Cobra command registration and flag binding
- Test output formatting for JSON and table modes
- Test error messages for invalid flag combinations
- Test workspace resolution for `--workspace` flag
- Mock UDS client for deterministic responses

**Coverage target:** 80%+ for CLI task command code.

### 3.5 Security Tests

**Scope:** Identity enforcement, permission checks, payload injection, field tampering.

**Approach:**
- Verify that `ActorContext` is always server-derived, never from client payload
- Verify that `Authority` checks gate every Manager method (read, write, create_global, create_workspace)
- Verify that `requireLifecycleIdempotency` enforces idempotency keys for non-human actors
- Verify that actors cannot escape their allowed origin kinds (e.g., human actor with automation origin)
- Verify that extensions without `task.write` capability are rejected
- Verify that network peers without channel-bound `task.write` capability are rejected
- Verify that payload size limits prevent resource exhaustion (metadata, result, payload)
- Verify SQL injection resistance in query filter parameters

### 3.6 Performance Tests

**Scope:** Throughput and latency under representative concurrent load.

**Approach:**
- Benchmark task creation throughput with 100 concurrent goroutines
- Benchmark dependency cycle detection with deep dependency chains (30 edges)
- Benchmark cascading cancellation with wide hierarchies (64 children)
- Measure SQLite write lock contention under concurrent EnqueueRun calls
- Measure queue depth query latency with 10,000 tasks
- Measure observability summary computation latency with 10,000 tasks and 50,000 runs

### 3.7 Regression Approach

- All test cases are automated and run in CI via `make verify`
- Integration tests run separately via `make test-integration`
- Smoke tests (SMOKE-001 through SMOKE-010) form the minimum regression gate for every PR
- Any test failure blocks merge -- zero tolerance per CLAUDE.md

---

## 4. Environment Requirements

| Requirement | Specification |
|-------------|---------------|
| Go version | 1.25.0 (per `go.mod`) |
| SQLite | Embedded via `modernc.org/sqlite` (CGo-free) |
| OS | darwin (development), linux (CI) |
| Test runner | `go test -race` via `make test` |
| Lint | `golangci-lint` via `make lint` (zero issues) |
| Integration tests | `go test -race -tags integration` via `make test-integration` |
| Temp storage | `t.TempDir()` for all file/database isolation |
| Mocking | Interface-based test doubles (no reflection mocking frameworks) |
| Build gate | `make verify` (fmt, lint, test, build) must pass |

---

## 5. Entry Criteria

All of the following must be true before test execution begins:

1. All 13 component packages compile without errors (`make build` passes)
2. `make fmt` produces no changes (code is properly formatted)
3. `make lint` reports zero issues
4. SQLite schema creation succeeds in `t.TempDir()` without migration errors
5. All task domain types, interfaces, and error sentinels are defined and exported
6. The `task.Manager` interface implementation (`TaskManager`) compiles and satisfies `var _ Manager = (*TaskManager)(nil)`
7. All 18 API endpoints are registered on both HTTP and UDS routers
8. All 8 CLI subcommands are registered under `agh task`
9. Feature branch is rebased on current `main` with no merge conflicts
10. PRD and technical design documents are reviewed and approved

---

## 6. Exit Criteria

| Criterion | Threshold |
|-----------|-----------|
| Unit test pass rate | 100% (zero failures) |
| Integration test pass rate | 100% (zero failures) |
| Unit test coverage for `internal/task/` | >= 90% |
| Unit test coverage for `internal/store/globaldb/` task ops | >= 85% |
| Unit test coverage for API handlers | >= 80% |
| Unit test coverage for CLI commands | >= 80% |
| Overall package coverage | >= 80% (per CLAUDE.md requirement) |
| `make verify` | Passes with zero warnings, zero errors |
| Race detector | Zero races detected under `-race` |
| All P0 (blocker) test cases | Pass |
| All P1 (critical) test cases | Pass |
| P2 (major) test cases | >= 95% pass rate |
| P3 (minor) test cases | >= 90% pass rate |
| Smoke tests (SMOKE-001 to SMOKE-010) | 100% pass |
| Performance benchmarks | No regression > 20% from baseline |
| Security test cases | 100% pass |

---

## 7. Risk Assessment Table

| # | Risk | Probability | Impact | Mitigation |
|---|------|-------------|--------|------------|
| R1 | Task state machine allows invalid transitions under concurrent run mutations | Medium | Critical | Integration tests with concurrent goroutines racing ClaimRun/CompleteRun; `-race` flag enforcement |
| R2 | Dependency cycle detection CTE fails or deadlocks under write contention | Low | Critical | Integration test with 10+ concurrent AddDependency calls forming near-cycle topologies; `BEGIN IMMEDIATE` transaction isolation |
| R3 | Cascading cancellation leaves orphaned sessions without force-stop | Medium | Critical | Integration test with 3-level hierarchy, running sessions, and verified ForceTaskStop calls; mock SessionExecutor tracking |
| R4 | Immutable field bypass through API PATCH with created_by/origin in body | Low | High | Security test: send PATCH with immutable fields in JSON body; verify 422/ignored at handler and store layers |
| R5 | Boot recovery marks live sessions as orphaned, killing active work | Medium | High | Integration test: pre-seed running run with mock live session; verify RecoverRunOnBoot chooses `mark_running` not `fail` |
| R6 | Idempotency key collision across different origin kinds returns wrong run | Low | High | Unit test: same key, two different origins; verify independent deduplication scopes |
| R7 | SQLite write lock timeout under burst task creation (100+ concurrent) | Medium | Medium | Performance benchmark with 100 concurrent CreateTask calls; measure p99 latency and failure rate |
| R8 | Observability metrics drift from actual persisted task/run state | Medium | Medium | Integration test: create known task/run distribution; query TaskSummary/TaskMetrics; assert exact counts match |
| R9 | Network channel validation accepts stale channels after peer disconnect | Medium | Medium | Unit test: configure channel validator that returns error; verify EnqueueRun and StartRun reject with `ErrStaleNetworkChannel` |
| R10 | Task hierarchy depth check off-by-one allows depth 9 | Low | Medium | Unit test: create chain of 8 parents; verify 9th creation fails with `ErrGraphLimitExceeded` |
| R11 | Workspace-scoped child task created under global parent violates scope invariant | Low | Medium | Unit test: global parent, workspace child -- verify success; workspace parent, global child -- verify rejection |
| R12 | Extension host API bypasses capability check for task writes | Low | High | Integration test: register extension without `task.write` capability; attempt task creation; verify rejection |
| R13 | CancelTask on already-terminal task returns misleading success | Medium | Low | Unit test: cancel completed task; verify `ErrInvalidStatusTransition` |
| R14 | ListTasks query filters silently ignore invalid enum values | Low | Low | Unit test: pass invalid scope/status/owner_kind values; verify validation error before query execution |
| R15 | Metadata/payload size validation uses `len()` on non-trimmed JSON | Low | Medium | Unit test: 16KB metadata with leading/trailing whitespace; verify trimmed size is checked |

---

## 8. Test Case Summary Matrix

### 8.1 By Component Area and Test Type

| Area | Unit | Integration | API | CLI | Security | Performance | Total |
|------|------|-------------|-----|-----|----------|-------------|-------|
| Task lifecycle (status transitions) | 12 | 3 | 2 | -- | -- | -- | 17 |
| Run lifecycle (status transitions) | 10 | 3 | 2 | -- | -- | -- | 15 |
| Identity & actor context | 6 | -- | 2 | -- | 4 | -- | 12 |
| Immutability enforcement | 5 | 2 | 1 | -- | 2 | -- | 10 |
| Size limits & validation | 8 | 1 | 1 | -- | 1 | 1 | 12 |
| Dependency graph & cycles | 4 | 4 | 1 | 1 | -- | 1 | 11 |
| Cascading cancellation | 3 | 3 | 1 | 1 | -- | 1 | 9 |
| Boot recovery | 3 | 3 | -- | -- | -- | -- | 6 |
| Session bridge | 4 | 2 | 1 | -- | -- | -- | 7 |
| Idempotency | 3 | 2 | 1 | -- | 1 | -- | 7 |
| Network channel | 3 | 2 | -- | -- | -- | -- | 5 |
| API endpoints (HTTP) | -- | 3 | 10 | -- | -- | -- | 13 |
| API endpoints (UDS) | -- | 3 | 6 | -- | -- | -- | 9 |
| CLI commands | -- | -- | -- | 8 | -- | -- | 8 |
| Automation dispatch | 3 | 2 | -- | -- | -- | -- | 5 |
| Extension host API | 2 | 2 | -- | -- | 2 | -- | 6 |
| Network ingress | 3 | 2 | -- | -- | 2 | -- | 7 |
| Observability | 4 | 3 | 2 | -- | -- | 2 | 11 |
| Smoke tests | -- | -- | -- | -- | -- | -- | 10 |

### 8.2 By Priority

| Priority | Count | Description |
|----------|-------|-------------|
| P0 (Blocker) | 22 | State machine correctness, cycle detection, identity enforcement, immutability, boot recovery |
| P1 (Critical) | 25 | Cascading cancellation, session bridge, idempotency, API contract parity, size limits |
| P2 (Major) | 18 | Query filters, CLI output, observability metrics, network channel validation |
| P3 (Minor) | 4 | Edge cases in normalization, optional field handling, metadata whitespace |
| Smoke | 10 | End-to-end happy paths covering critical user journeys |

---

## 9. Timeline and Deliverables

| Phase | Duration | Deliverables |
|-------|----------|------------|
| Phase 1: Unit tests for `internal/task/` | 2 days | TC-FUNC-001 to TC-FUNC-020 implemented; 90%+ coverage for task package |
| Phase 2: Store integration tests | 2 days | TC-INT-001 to TC-INT-008 implemented; cycle detection, immutability, concurrency verified |
| Phase 3: API and CLI tests | 2 days | TC-FUNC-021 to TC-FUNC-030, TC-INT-009 to TC-INT-015 implemented; API parity verified |
| Phase 4: Security and performance tests | 1 day | TC-SEC-001 to TC-SEC-008, TC-PERF-001 to TC-PERF-006 implemented |
| Phase 5: Smoke tests and regression | 1 day | SMOKE-001 to SMOKE-010 implemented; full regression pass; `make verify` green |
| Phase 6: Test report and sign-off | 0.5 day | Test execution report, coverage report, defect summary |

**Total estimated duration:** 8.5 working days

---

## 10. Traceability Matrix

### 10.1 Functional Tests (TC-FUNC-001 to TC-FUNC-030)

| TC-ID | Feature | Description | Priority | Type |
|-------|---------|-------------|----------|------|
| TC-FUNC-001 | Task lifecycle | CreateTask produces `ready` status with server-derived created_by/origin | P0 | Unit |
| TC-FUNC-002 | Task lifecycle | CreateTask with workspace scope requires non-empty workspace_id | P0 | Unit |
| TC-FUNC-003 | Task lifecycle | CreateTask with global scope rejects non-empty workspace_id | P0 | Unit |
| TC-FUNC-004 | Task lifecycle | CreateChildTask increments parent child count and emits child_created event | P1 | Unit |
| TC-FUNC-005 | Task lifecycle | CreateChildTask enforces MaxDirectChildren (64) limit | P1 | Unit |
| TC-FUNC-006 | Task lifecycle | CreateChildTask enforces MaxHierarchyDepth (8) limit | P1 | Unit |
| TC-FUNC-007 | Task lifecycle | UpdateTask applies partial patch and preserves immutable fields | P0 | Unit |
| TC-FUNC-008 | Task lifecycle | UpdateTask with no changed fields returns current task without write | P2 | Unit |
| TC-FUNC-009 | Task lifecycle | CancelTask on ready task transitions to cancelled with ClosedAt set | P0 | Unit |
| TC-FUNC-010 | Task lifecycle | CancelTask on terminal (completed/failed) task returns ErrInvalidStatusTransition | P1 | Unit |
| TC-FUNC-011 | Run lifecycle | EnqueueRun on ready task creates queued run with correct attempt number | P0 | Unit |
| TC-FUNC-012 | Run lifecycle | EnqueueRun on cancelled task returns ErrInvalidStatusTransition | P0 | Unit |
| TC-FUNC-013 | Run lifecycle | ClaimRun transitions queued run to claimed with actor identity | P0 | Unit |
| TC-FUNC-014 | Run lifecycle | StartRun from claimed state: starting -> session bind -> running | P0 | Unit |
| TC-FUNC-015 | Run lifecycle | StartRun from starting state with session binding transitions to running | P0 | Unit |
| TC-FUNC-016 | Run lifecycle | CompleteRun transitions running run to completed with result payload | P1 | Unit |
| TC-FUNC-017 | Run lifecycle | FailRun transitions running/starting run to failed with error message | P1 | Unit |
| TC-FUNC-018 | Run lifecycle | CancelRun on queued/claimed run cancels immediately without session stop | P1 | Unit |
| TC-FUNC-019 | Run lifecycle | CancelRun on running run triggers cooperative then forced session stop | P0 | Unit |
| TC-FUNC-020 | Run lifecycle | All invalid run transitions return ErrInvalidStatusTransition | P0 | Unit |
| TC-FUNC-021 | Dependency graph | AddDependency creates edge and reconciles task to blocked if unresolved | P0 | Unit |
| TC-FUNC-022 | Dependency graph | RemoveDependency deletes edge and reconciles task to ready if all resolved | P1 | Unit |
| TC-FUNC-023 | Dependency graph | AddDependency self-referential (A depends on A) rejected by validation | P1 | Unit |
| TC-FUNC-024 | Cascading cancel | CancelTask propagates to all non-terminal descendants | P0 | Unit |
| TC-FUNC-025 | Cascading cancel | CancelTask skips already-terminal descendants | P1 | Unit |
| TC-FUNC-026 | Boot recovery | RecoverRunOnBoot requeue resets claimed run to queued | P0 | Unit |
| TC-FUNC-027 | Boot recovery | RecoverRunOnBoot mark_running promotes starting run with session to running | P0 | Unit |
| TC-FUNC-028 | Boot recovery | RecoverRunOnBoot fail marks orphaned run as failed with recovery metadata | P0 | Unit |
| TC-FUNC-029 | Session bridge | AttachRunSession on claimed run transitions to starting with session_id | P1 | Unit |
| TC-FUNC-030 | Session bridge | AttachRunSession rejects if session already bound to active run | P1 | Unit |

### 10.2 Integration Tests (TC-INT-001 to TC-INT-015)

| TC-ID | Feature | Description | Priority | Type |
|-------|---------|-------------|----------|------|
| TC-INT-001 | Store: task CRUD | Create, read, update, list tasks round-trip through real SQLite | P0 | Integration |
| TC-INT-002 | Store: run CRUD | Create, read, update, list runs round-trip through real SQLite | P0 | Integration |
| TC-INT-003 | Store: dependency cycle | AddDependency detects cycle via recursive CTE under BEGIN IMMEDIATE | P0 | Integration |
| TC-INT-004 | Store: dependency limit | AddDependency enforces 32-edge limit within transaction | P1 | Integration |
| TC-INT-005 | Store: immutability | UpdateTask rejects changes to scope, workspace_id, parent_task_id, created_by, origin | P0 | Integration |
| TC-INT-006 | Store: idempotency | GetTaskRunByIdempotencyKey returns correct run scoped to origin | P1 | Integration |
| TC-INT-007 | Manager: full lifecycle | Create task -> enqueue -> claim -> start -> complete; verify all events recorded | P0 | Integration |
| TC-INT-008 | Manager: cascading cancel | 3-level hierarchy with running sessions; cancel root; verify all descendants cancelled and sessions stopped | P0 | Integration |
| TC-INT-009 | API: HTTP task CRUD | POST/GET/PATCH/LIST tasks via HTTP transport with real manager and store | P1 | Integration |
| TC-INT-010 | API: UDS task CRUD | POST/GET/PATCH/LIST tasks via UDS transport with real manager and store | P1 | Integration |
| TC-INT-011 | API: HTTP run lifecycle | Enqueue/claim/start/complete run via HTTP with correct status transitions | P1 | Integration |
| TC-INT-012 | API: UDS run lifecycle | Enqueue/claim/start/complete run via UDS with correct status transitions | P1 | Integration |
| TC-INT-013 | Observe: task summary | Create mixed task/run state; verify TaskSummary buckets match expected counts | P2 | Integration |
| TC-INT-014 | Observe: task metrics | Full lifecycle with cancellation; verify forced-stop and recovery counters | P2 | Integration |
| TC-INT-015 | Observe: stuck runs | Create claimed run; advance clock past threshold; verify stuck run detection | P2 | Integration |

### 10.3 Security Tests (TC-SEC-001 to TC-SEC-008)

| TC-ID | Feature | Description | Priority | Type |
|-------|---------|-------------|----------|------|
| TC-SEC-001 | Identity enforcement | CreateTask ignores created_by/origin from request body; uses server-derived values | P0 | Security |
| TC-SEC-002 | Identity enforcement | All 6 actor kinds validated against allowed origin kinds | P0 | Security |
| TC-SEC-003 | Permission checks | Read-only authority cannot call CreateTask/UpdateTask/CancelTask | P0 | Security |
| TC-SEC-004 | Permission checks | Write authority without create_global cannot create global-scope tasks | P1 | Security |
| TC-SEC-005 | Idempotency enforcement | Non-human actors without idempotency_key rejected for EnqueueRun/ClaimRun/StartRun | P1 | Security |
| TC-SEC-006 | Extension capability | Extension without task.write capability rejected at host API | P1 | Security |
| TC-SEC-007 | Network capability | Network peer without task.write capability rejected at ingress | P1 | Security |
| TC-SEC-008 | Payload injection | Oversized metadata (>16KB), result (>64KB), and event payload (>64KB) rejected with ErrPayloadTooLarge | P1 | Security |

### 10.4 Performance Tests (TC-PERF-001 to TC-PERF-006)

| TC-ID | Feature | Description | Priority | Type |
|-------|---------|-------------|----------|------|
| TC-PERF-001 | Task creation throughput | 100 concurrent CreateTask calls complete within 5s with zero failures | P2 | Performance |
| TC-PERF-002 | Dependency cycle detection | Cycle check on 30-edge dependency chain completes within 100ms | P2 | Performance |
| TC-PERF-003 | Cascading cancellation | Cancel root of 64-child hierarchy completes within 2s | P2 | Performance |
| TC-PERF-004 | SQLite write contention | 100 concurrent EnqueueRun calls; measure p99 latency and failure rate | P2 | Performance |
| TC-PERF-005 | Queue depth query | TaskSummary query with 10,000 tasks completes within 500ms | P3 | Performance |
| TC-PERF-006 | Observability computation | TaskMetrics with 10,000 tasks and 50,000 runs completes within 1s | P3 | Performance |

### 10.5 Smoke Tests (SMOKE-001 to SMOKE-010)

| TC-ID | Feature | Description | Priority |
|-------|---------|-------------|----------|
| SMOKE-001 | Task creation | Create global task via CLI; verify `agh task get` returns it | P0 |
| SMOKE-002 | Workspace task | Create workspace-scoped task via HTTP; verify scope and workspace_id in response | P0 |
| SMOKE-003 | Task list | Create 3 tasks; list with status filter; verify filtered results | P0 |
| SMOKE-004 | Task update | Create task; PATCH title; verify updated title in GET response | P0 |
| SMOKE-005 | Task cancel | Create task; cancel; verify status is cancelled with closed_at set | P0 |
| SMOKE-006 | Child task | Create parent; create child via POST /:id/children; verify parent_task_id | P0 |
| SMOKE-007 | Dependency | Create two tasks; add dependency A->B; verify A is blocked | P0 |
| SMOKE-008 | Run lifecycle | Create task; enqueue run; claim; start; complete; verify task status is completed | P0 |
| SMOKE-009 | Run cancel | Create task; enqueue run; cancel run; verify run status is cancelled | P0 |
| SMOKE-010 | Observability | Create tasks and runs; query task summary via API; verify non-zero totals | P1 |

### 10.6 Full Traceability: Feature to Test Cases

| Feature | TC IDs |
|---------|--------|
| Task create (global/workspace) | TC-FUNC-001, TC-FUNC-002, TC-FUNC-003, TC-INT-001, TC-SEC-001, TC-PERF-001, SMOKE-001, SMOKE-002 |
| Task update (partial patch) | TC-FUNC-007, TC-FUNC-008, TC-INT-005, SMOKE-004 |
| Task cancel (single + cascade) | TC-FUNC-009, TC-FUNC-010, TC-FUNC-024, TC-FUNC-025, TC-INT-008, TC-PERF-003, SMOKE-005 |
| Child task create | TC-FUNC-004, TC-FUNC-005, TC-FUNC-006, SMOKE-006 |
| Dependency graph | TC-FUNC-021, TC-FUNC-022, TC-FUNC-023, TC-INT-003, TC-INT-004, TC-PERF-002, SMOKE-007 |
| Run enqueue | TC-FUNC-011, TC-FUNC-012, TC-INT-002, TC-SEC-005, TC-PERF-004, SMOKE-008 |
| Run claim | TC-FUNC-013, TC-FUNC-020 |
| Run start | TC-FUNC-014, TC-FUNC-015, TC-FUNC-020 |
| Run complete | TC-FUNC-016, TC-FUNC-020, SMOKE-008 |
| Run fail | TC-FUNC-017, TC-FUNC-020 |
| Run cancel | TC-FUNC-018, TC-FUNC-019, TC-FUNC-020, SMOKE-009 |
| Boot recovery | TC-FUNC-026, TC-FUNC-027, TC-FUNC-028 |
| Session bridge | TC-FUNC-029, TC-FUNC-030 |
| Idempotency | TC-INT-006, TC-SEC-005 |
| Identity enforcement | TC-SEC-001, TC-SEC-002, TC-SEC-003, TC-SEC-004 |
| Extension host API | TC-SEC-006 |
| Network ingress | TC-SEC-007 |
| Payload limits | TC-SEC-008 |
| HTTP API | TC-INT-009, TC-INT-011, SMOKE-001 through SMOKE-010 |
| UDS API | TC-INT-010, TC-INT-012 |
| CLI commands | SMOKE-001, SMOKE-003, SMOKE-004, SMOKE-005 |
| Observability | TC-INT-013, TC-INT-014, TC-INT-015, TC-PERF-005, TC-PERF-006, SMOKE-010 |
| Task list with filters | TC-INT-001, SMOKE-003 |
| Immutability | TC-FUNC-007, TC-INT-005, TC-SEC-001, TC-SEC-004 |
| Network channel validation | TC-FUNC-011, TC-FUNC-014 |
| Size limits | TC-SEC-008, TC-PERF-001 |
