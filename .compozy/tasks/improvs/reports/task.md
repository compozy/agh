# Improvements Report — internal/task

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/task/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/task | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 46 | `TestManagerRecoverRunOnBoot` | `internal/task/manager_test.go:2035` |
| 29 | `TestManagerTaskReconciliationAcrossDependenciesAndRuns` | `internal/task/manager_test.go:1026` |
| 26 | `TestTaskManagerCancelTaskTreePersistsCancellationAudit` | `internal/task/manager_integration_test.go:351` |
| 23 | `TestManagerCancelTaskPropagatesAcrossTree` | `internal/task/manager_test.go:1165` |
| 23 | `TestManagerAttachRunSessionAndRetryLatestRunOutcome` | `internal/task/manager_test.go:1271` |
| 23 | `TestManagerAdditionalBranchCoverage` | `internal/task/manager_test.go:2421` |
| 22 | `TestManagerStartRunAndAttachErrorBranches` | `internal/task/manager_test.go:1876` |
| 21 | `TestTaskManagerRunLifecyclePersistsAndReconcilesAgainstStorage` | `internal/task/manager_integration_test.go:252` |
| 21 | `TestManagerStartRunRejectsStaleRunChannelWithoutMutation` | `internal/task/manager_test.go:1523` |
| 21 | `(*inMemoryManagerStore).ListTasks` | `internal/task/manager_test.go:138` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/task/types.go` | 397 | Domain enums, persisted records, transport DTOs, and session-bridge payloads are concentrated in one file, which is still cohesive but hard to scan. |
| `internal/task/validate.go` | 729 | Validation logic for enums, records, requests, queries, and size guards is centralized here, with visible repeated patterns across small validator blocks. |
| `internal/task/manager.go` | 2370 | The task service concentrates creation, reconciliation, run lifecycle, cancellation, boot recovery, and serialization helpers in one orchestration-heavy unit. |

### Refactoring — Duplication

`dupl -plumbing -t 20 internal/task` found the following package-local duplicates at or above the reporting threshold:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/task/validate.go:217-225` | `internal/task/validate.go:228-236` | Repeated kind+ref validation structure across `ActorIdentity.Validate` and `Ownership.Validate`. |
| `internal/task/validate.go:228-236` | `internal/task/validate.go:244-252` | Same pattern again in `Origin.Validate`. |
| `internal/task/validate.go:531-535` | `internal/task/validate.go:536-540` | Repeated non-negative limit validation across query validators. |
| `internal/task/validate.go:536-540` | `internal/task/validate.go:549-553` | Same limit guard repeated in `RunQuery.Validate` and `EventQuery.Validate`. |
| `internal/task/manager.go:1663-1670` | `internal/task/manager.go:1672-1679` | Parallel terminal-status switches for task status and run status. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `taskStatusFromSnapshot` | `internal/task/manager.go:1681` | Every manager mutation that reconciles task state calls this helper through `canonicalTaskStatus`; it currently re-scans the same run slice for active, queued/claimed, and latest terminal statuses. | `BenchmarkTaskStatusFromSnapshotLatestTerminal`, `BenchmarkTaskStatusFromSnapshotQueuedAfterTerminal` |
| `normalizeRawJSON` | `internal/task/manager.go:2249` | Every metadata/result/failure payload is normalized before validation or persistence, and the current implementation converts the payload to a trimmed string on each call. | `BenchmarkNormalizeRawJSONTrimmed256B` |
| `sameRawJSON` | `internal/task/manager.go:2269` | Mutable patch flows compare JSON payloads on the write path; the current helper normalizes and re-stringifies both inputs for equality checks. | `BenchmarkSameRawJSONTrimmed256B` |

### Optimization — Benchmark Results

Baseline command: `go test -bench=. -benchmem -count=5 ./internal/task/... | tee /tmp/task-bench-before.txt`

Final command: `go test -bench=. -benchmem -count=5 ./internal/task/... | tee /tmp/task-bench-after.txt`

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkTaskStatusFromSnapshotLatestTerminal` | 21616 | 73728 | 4429 | 0 | fixed |
| `BenchmarkTaskStatusFromSnapshotQueuedAfterTerminal` | 4681 | 0 | 4465 | 0 | fixed |
| `BenchmarkNormalizeRawJSONTrimmed256B` | 90.32 | 544 | 3.665 | 0 | fixed |
| `BenchmarkSameRawJSONTrimmed256B` | 136.1 | 1056 | 9.372 | 0 | fixed |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/task/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/task/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/task/`.

### Concurrency — Select Audit

All selects are context-aware or input-bounded. The only production `select` is `internal/task/manager.go:2030`, where `waitAndForceStopRun` waits on either the grace-period timer or `ctx.Done()`.

### Security — Threat Model

- Trust boundaries:
  - Authenticated ingress layers in `internal/api/core`, `internal/cli`, `internal/network`, `internal/automation`, and `internal/extension` pass task-domain requests and actor metadata into `internal/task`.
  - `internal/task` persists canonical records through the injected store interfaces and delegates session lifecycle actions through the injected `SessionExecutor`.
- Attacker capabilities:
  - An authenticated caller may control task IDs, titles, descriptions, owners, network channels, JSON metadata/result/failure payloads, idempotency keys, and actor/origin refs supplied to the actor-derivation helpers.
  - Attackers do not control the injected store or session-executor implementations from within this package.
- In-scope assets:
  - Integrity of persisted task/run/dependency/event records.
  - Correct task ownership, actor attribution, and origin attribution.
  - Safe propagation of session start/attach/stop requests to the trusted session bridge.
- Out-of-scope:
  - Malicious or compromised store/session-executor implementations injected by the composition root.
  - Authorization decisions made before this package (the task package trusts the supplied authority envelope after validation).
  - SQL, filesystem, or network behavior implemented in downstream packages rather than `internal/task` itself.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/task/actors.go:19` | Human actor/origin refs from authenticated CLI, web, UDS, or HTTP ingress. | `DeriveHumanActorContext` restricts `origin.kind`; `deriveActorContext` runs `ActorContext.Validate`. | Immutable actor/origin fields persisted on tasks and events. | LOW — rejected; strict non-empty ref and actor-origin pairing validation, with no execution sink in this package. |
| `internal/task/actors.go:34` | Agent-session refs from session-owned task writes. | `deriveActorContext` validates non-empty refs and fixed `agent_session` origin pairing. | Immutable actor/origin fields on tasks, runs, and events. | LOW — rejected; trusted internal identity binding only. |
| `internal/task/actors.go:42` | Automation-linked session refs and automation origin refs. | Empty origin refs collapse to the session ref before `ActorContext.Validate`. | Immutable actor/origin fields on persisted tasks and run events. | LOW — rejected; no privilege escalation inside this package after validation. |
| `internal/task/actors.go:51` | Automation actor refs. | Empty origin refs collapse to actor refs before `ActorContext.Validate`. | Immutable actor/origin fields on persisted tasks and run events. | LOW — rejected; trusted internal identity binding only. |
| `internal/task/actors.go:60` | Extension actor refs. | Empty origin refs collapse to actor refs before `ActorContext.Validate`. | Immutable actor/origin fields on persisted tasks and run events. | LOW — rejected; no command execution or deserialization sink here. |
| `internal/task/actors.go:70` | Network-peer actor and origin refs. | Empty origin refs collapse to actor refs before `ActorContext.Validate`. | Immutable actor/origin fields and idempotency scoping. | LOW — rejected; origin pairing is constrained and the package only persists the data. |
| `internal/task/actors.go:79` | Daemon actor/origin refs. | Empty origin refs collapse to actor refs before `ActorContext.Validate`. | Boot-recovery and daemon-owned task/run events. | LOW — rejected; internal-only surface. |
| `internal/task/manager.go:143` | `CreateTask` request fields: IDs, scope/workspace, title/description, owner, metadata, and network channel. | `normalizeCreateTaskSpec`, `CreateTask.Validate`, `validateParentConstraints`, `validateNetworkChannel`, and `Task.Validate`. | `store.CreateTask` + `recordTaskEvent`. | LOW — rejected; validated persistence only, no execution sink. |
| `internal/task/manager.go:233` | `UpdateTask` patch fields: title/description, metadata, owner, and network channel. | `normalizeTaskPatch`, `Patch.Validate`, `validateNetworkChannel`, and immutable-field preservation via `applyTaskPatch`. | `store.UpdateTask` + `recordTaskEvent`. | LOW — rejected; bounded metadata and validated patch semantics only. |
| `internal/task/manager.go:292` | `CancelTask` reason/metadata and task ID. | `normalizeCancelTask`, `CancelTask.Validate`, task-tree loading, and transition guards. | `store.UpdateTask`, `store.UpdateTaskRun`, session stop requests, and cancellation events. | LOW — rejected; the trusted session bridge receives validated task-owned session IDs only. |
| `internal/task/manager.go:829` | Dependency task IDs and dependency kind. | `normalizeAddDependencySpec`, `AddDependency.Validate`, dependency count/cycle checks, and reconciliation. | `store.CreateDependency` / `store.DeleteDependency` + dependency events. | LOW — rejected; no external execution or traversal sink. |
| `internal/task/manager.go:894` | `EnqueueRun` task ID, optional idempotency key, and optional network channel. | `normalizeEnqueueRunSpec`, `requireLifecycleIdempotency`, `validateNetworkChannel`, `lookupIdempotentRun`, and task-status guards. | `store.CreateTaskRun`, idempotency store, and run-enqueued event. | LOW — rejected; persisted dedupe key only, with no command execution. |
| `internal/task/manager.go:968` | `ClaimRun` run ID and optional idempotency key. | `normalizeClaimRun`, `requireLifecycleIdempotency`, `loadRunWithTask`, and transition validation. | `store.UpdateTaskRun` + run-claimed event. | LOW — rejected; validated state transition only. |
| `internal/task/manager.go:1020` | `StartRun` run ID and optional idempotency key. | `normalizeStartRun`, `requireLifecycleIdempotency`, `loadRunWithTask`, `ensureTaskExecutable`, `validateRunChannelUsable`, and session-ref validation. | `SessionExecutor.StartTaskSession`, `store.UpdateTaskRun`, and run-started event. | LOW — rejected; the session executor is trusted and receives validated task/run context, not attacker-controlled shell fragments. |
| `internal/task/manager.go:1084` | `AttachRunSession` run ID and session ID. | `strings.TrimSpace`, `requireSessionExecutor`, `validateAttachedSessionBinding`, and `CountActiveSessionBindings`. | `SessionExecutor.AttachTaskSession`, `store.UpdateTaskRun`, and session-bound event. | LOW — rejected; only validated existing session IDs reach the trusted bridge. |
| `internal/task/manager.go:1165` | `CompleteRun` run ID and JSON result payload. | `normalizeRunResult` + `RunResult.Validate`. | `store.UpdateTaskRun` + completed-run event payload. | LOW — rejected; the package stores the JSON verbatim without interpreting it. |
| `internal/task/manager.go:1212` | `FailRun` run ID, error text, and JSON metadata. | `normalizeRunFailure` + `RunFailure.Validate`. | `store.UpdateTaskRun` + failed-run event payload. | LOW — rejected; bounded metadata only, no execution or rendering sink. |
| `internal/task/manager.go:1235` | `CancelRun` run ID, reason, and metadata. | `normalizeCancelRun`, `CancelRun.Validate`, and transition guards. | `store.UpdateTaskRun`, session stop requests, and run-canceled / force-stopped events. | LOW — rejected; validated persistence plus trusted session control only. |
| `internal/task/manager.go:746` | Task ID lookups for detail reads. | `strings.TrimSpace`, read-authority checks, and store query validation. | `store.GetTask`, `ListDependencies`, `ListTasks`, `ListTaskRuns`, `ListTaskEvents`. | LOW — rejected; read-only query surface. |
| `internal/task/manager.go:794` | Task ID plus run-list query filters. | `strings.TrimSpace` on task ID; downstream `RunQuery.Validate` in the store. | `store.ListTaskRuns`. | LOW — rejected; read-only query surface. |
| `internal/task/manager.go:820` | Task list query filters (`scope`, `workspace_id`, `status`, `owner`, `network_channel`, `limit`). | Read-authority check plus downstream `Query.Validate` in the store. | `store.ListTasks`. | LOW — rejected; read-only query surface. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `TASK-OPT-001` | extreme-software-optimization | low | `internal/task/manager.go:1681` | `taskStatusFromSnapshot` rescanned the run slice and allocated via `latestTerminalRun`, inflating reconciliation cost on terminal-run paths. | fixed |
| `TASK-OPT-002` | extreme-software-optimization | low | `internal/task/manager.go:2249` | Raw JSON normalization/comparison converted payloads to trimmed strings on each call, adding avoidable allocations on mutation paths. | fixed |
| `TASK-REF-001` | refactoring-analysis | medium | `internal/task/manager.go:1` | `manager.go` remains a very large orchestration unit that mixes lifecycle, reconciliation, boot recovery, and serialization helpers. | deferred |
| `TASK-REF-002` | refactoring-analysis | medium | `internal/task/validate.go:217` | Validator boilerplate is repeated across identity/origin validators and query limit guards. | deferred |

## Per-Skill Notes

### refactoring-analysis

- `internal/task` is structurally dominated by `manager.go` and `validate.go`; both exceed repository norms for non-test file size.
- The duplication scan found real production duplication in validator blocks and terminal-status helpers, but most extraction options would introduce shared helpers without reducing operational risk in this pass.
- The package still benefits from direct helper tests: coverage increased slightly to `80.7%`, and the targeted additions now lock the optimized helper semantics in place.

### extreme-software-optimization

- Baseline benchmarks were added in `internal/task/perf_bench_test.go` and captured before any production code change.
- Targeted CPU profiles (`go test ./internal/task -run '^$' -bench BenchmarkTaskStatusFromSnapshotLatestTerminal -cpuprofile /tmp/task-status-cpu.out` and `... -bench BenchmarkSameRawJSONTrimmed256B -cpuprofile /tmp/task-json-cpu.out`) pointed at the expected roots: `latestTerminalRun`/`hasQueuedOrClaimedRun` on the status path and `stringtoslicebyte` / `slicebytetostring` around `normalizeRawJSON`.
- `taskStatusFromSnapshot` now resolves status in a single pass over the run slice and stores the latest terminal run by value, eliminating the previous 256 allocs/op on the terminal-run benchmark.
- `normalizeRawJSON` now trims with `bytes.TrimSpace`, and `sameRawJSON` now compares normalized byte slices directly, dropping both helper benchmarks to zero allocations.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the package contains no goroutines, channels, or mutexes.
- The only production `select` is `internal/task/manager.go:2030`, and it already watches `ctx.Done()` while waiting for the forced-stop grace period.

### security-review

- The package is a validated domain boundary that persists canonical task/run data and delegates session control to trusted injected collaborators.
- No HIGH-confidence or MEDIUM-confidence vulnerability survived the source-to-sink review under the declared threat model.

## Deferred Items (carry forward)

- **`TASK-REF-001`** — Splitting `manager.go` by lifecycle/reconciliation/recovery concern would need a deliberate package-internal reorganization and broader benchmark/test review; keep it as a dedicated follow-up instead of mixing it into this pass.
- **`TASK-REF-002`** — The repeated validator boilerplate is real, but extracting helper layers just for cosmetic deduplication would add new abstractions without reducing immediate behavioral risk.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/task (1.115s)
DONE 4510 tests in 11.398s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0` on the final tree after the `internal/task` changes and report refresh.
