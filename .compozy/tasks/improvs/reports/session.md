# Improvements Report — internal/session

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/session/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 $(rg --files internal/session --glob '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 19 | `(*Manager).ListAll` | `internal/session/query.go:16` |
| 18 | `(*Manager).ExecEnvironment` | `internal/session/environment_exec.go:30` |
| 14 | `NewManager` | `internal/session/manager.go:222` |
| 14 | `(*Manager).runContextCompaction` | `internal/session/manager_hooks.go:465` |
| 13 | `classifyStopReason` | `internal/session/stop_reason.go:14` |
| 13 | `(*Manager).pumpPrompt` | `internal/session/manager_prompt.go:200` |
| 13 | `(*Manager).finalizeEnvironment` | `internal/session/environment.go:564` |
| 13 | `(*Manager).StopWithCause` | `internal/session/stop_reason.go:89` |
| 12 | `normalizePreparedEnvironmentState` | `internal/session/environment.go:786` |
| 12 | `(*Manager).startSession` | `internal/session/manager_start.go:122` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/session/environment.go` | 1131 | Environment preparation, runtime sync, stop teardown, metadata fallback, and workspace file-count helpers all live in one large unit. |
| `internal/session/manager_hooks.go` | 863 | Session, agent, event, message, and compaction hook payload assembly/dispatch are co-located in one dense orchestration file. |
| `internal/session/session.go` | 611 | Session state transitions, metadata snapshots, prompt coordination, and process handle helpers are concentrated in one file. |
| `internal/session/manager.go` | 544 | Manager construction, dependency wiring, session registries, and finalization bookkeeping share one large implementation unit. |
| `internal/session/manager_start.go` | 393 | Create/resume preparation, runtime start, prompt assembly, and metadata bootstrapping are bundled together. |
| `internal/session/hooks.go` | 371 | Hook interfaces, hook-set accessors, and all no-op implementations are defined together rather than split by hook domain. |
| `internal/session/manager_lifecycle.go` | 365 | Create/resume entry points, process watching, stop finalization, and workspace/MCP preparation live together. |
| `internal/session/manager_prompt.go` | 325 | Prompt API entry points, input validation, prompt setup serialization, and stream pumping are all in one file. |
| `internal/session/interfaces.go` | 316 | Agent-process data model, constructor defaults, driver interfaces, and related abstractions share one large surface file. |
| `internal/session/query.go` | 313 | Session listing/status/history/event queries, disk-open logic, and stored-session ID validation are all concentrated together. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 30 $(rg --files internal/session --glob '!**/*_test.go' | sort)` found the following production duplicates at or above the reporting threshold:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/session/query.go:108-128` | `internal/session/query.go:131-151` | `Events` and `History` duplicate the same open/defer/query wrapper shape with only the recorder method changed. |
| `internal/session/manager_hooks.go:444-463` | `internal/session/manager_hooks.go:540-559` | Repeated post-patch payload/denial handling appears in multiple hook dispatch branches. |
| `internal/session/environment.go:400-408` | `internal/session/environment.go:460-468` | Environment state/update bookkeeping repeats across prepare/sync/stop transitions. |
| `internal/session/environment.go:460-468` | `internal/session/environment.go:499-507` | Same metadata persistence/update shape repeats in adjacent environment state transitions. |
| `internal/session/hooks.go:32-59` | `internal/session/hooks.go:76-94` | Hook interface method shapes repeat across prompt/event/agent/conversation hook domains. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*Manager).dispatchEnvironmentSyncBefore` | `internal/session/environment.go:287` | This runs on every environment sync path before runtime transfer and was building hook payload metadata even when no environment hooks were configured. | `BenchmarkDispatchEnvironmentSyncBeforeNoHooks` |
| `(*Manager).ListAll` | `internal/session/query.go:16` | This is the shared disk-backed session listing path for API/CLI callers and merges in-memory state with on-disk metadata on every list call. | `BenchmarkManagerListAllLarge` |
| `(*Session).Info` | `internal/session/session.go:88` | This allocation-heavy read model is called throughout notifier, lifecycle, and query code whenever session state is observed. | `BenchmarkSessionInfo` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -run '^$' -bench=. -benchmem -count=5 ./internal/session/...`

Final `after` command: `go test -run '^$' -bench=. -benchmem -count=5 ./internal/session/...`

Values below use the mean of 5 runs captured in `/tmp/session-bench-before.txt` and `/tmp/session-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkDispatchEnvironmentSyncBeforeNoHooks` | 269506.0 | 92776.8 | 162.3 | 288.0 | fixed-with-benchmark |
| `BenchmarkManagerListAllLarge` | 3994116.8 | 540985.2 | 4023230.8 | 540987.0 | not-hot-confirmed-by-benchmark |
| `BenchmarkSessionInfo` | 173.0 | 544.0 | 173.1 | 544.0 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/session/manager_prompt.go:88` | `Manager.PromptWithOpts` | Stops when the driver closes `source` or the request `ctx` is canceled; always closes `out` on exit. | Prompt stream pump that mirrors agent events into storage and notifier hooks. |
| `internal/session/manager_lifecycle.go:108` | `Manager.watchProcess` | Stops when `m.lifecycleCtx` is canceled or `proc.Done()` closes. | Process watcher that routes runtime exit into `handleProcessExit` / `finalizeStopped`. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/session/interfaces.go:64` | external | Agent runtime / driver implementation | Underlying driver/native process | `AgentProcess.Done`, `Wait`, `isProcessDone`, process watcher | Read-only runtime exit signal carried by `AgentProcess`. |
| `internal/session/interfaces.go:90` | 0 | `NewAgentProcess` fallback path | `NewAgentProcess` closes the synthesized channel immediately when no runtime `Done` channel is provided. | `AgentProcess.Done` / `Wait` callers | Closed fallback channel prevents nil-channel hangs in custom driver implementations. |
| `internal/session/manager.go:62` | 0 per session | `Manager.finalizing` map | `finishFinalization` and `remove` | `claimOrWaitFinalization`, `WaitForFinalizations` | Per-session finalization latch prevents duplicate stop finalization. |
| `internal/session/manager.go:495` | 0 | `claimFinalization` | `finishFinalization` / `remove` | `claimOrWaitFinalization`, `WaitForFinalizations` | Backing channel for one active finalizer per session. |
| `internal/session/manager_prompt.go:85` | `m.promptBufSize` | `PromptWithOpts` / `pumpPrompt` | `pumpPrompt` | Prompt callers collecting streamed agent events | Buffered agent-event fan-out channel per prompt turn. |
| `internal/session/session.go:92` | 0 | `Session` prompt setup coordination | `finishPromptSetup` or `closedSignalChan` | `waitForPromptSetup`, `prepareStop` | Gates stop/finalize while prompt setup is still in flight. |
| `internal/session/session.go:608` | 0 | `closedSignalChan` helper | `closedSignalChan` closes it immediately before returning | `beginPromptSetup`, `prepareStop` | Convenience closed channel used to represent “no prompt setup in flight”. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/session/manager.go:59` | read-heavy (`sync.RWMutex`) | `sessions`, `pending`, `finalizing`, `networkPeers`, `turnEndNotifier` | Central manager registry/late-bound dependency lock. |
| `internal/session/session.go:65` | read-heavy (`sync.RWMutex`) | Session lifecycle state, metadata snapshots, process handle, prompt setup coordination, and stop classification fields | Most read paths are `Info`/handle access; writes happen on lifecycle transitions. |

### Concurrency — Select Audit

| File:Line | Notes |
| --- | --- |
| `internal/session/manager_prompt.go:225` | `pumpPrompt` waits on `ctx.Done()` or the agent event source channel. |
| `internal/session/manager_prompt.go:242` | `pumpPrompt` forwards one event to `out` or exits on `ctx.Done()`. |
| `internal/session/manager_helpers.go:202` | `isProcessDone` uses a non-blocking probe with `default`; intentionally input-bounded and not a long-lived wait. |
| `internal/session/manager_helpers.go:214` | `waitForPromptSetup` waits on `promptSetupDone` with a `ctx.Done()` escape hatch. |
| `internal/session/manager.go:524` | `WaitForFinalizations` waits on each finalization latch or the caller context. |
| `internal/session/manager_lifecycle.go:109` | `watchProcess` exits on manager lifecycle cancellation or process completion. |
| `internal/session/manager_lifecycle.go:211` | `claimOrWaitFinalization` waits for an in-flight finalizer or the caller context. |
| `internal/session/stop_reason.go:114` | `StopWithCause` waits for `proc.Done()` only on the successful stop path, and still observes `ctx.Done()`. |

### Security — Threat Model

- Trust boundaries:
  - `internal/session` sits behind the daemon’s HTTP/UDS/CLI/control-plane packages, which pass session IDs, prompt messages, stop requests, permission approvals, and environment-exec requests into this package.
  - The package also reads and writes per-session metadata and event databases under the AGH home directory, so filesystem boundaries around `home/sessions/<id>` are security-relevant.
  - Runtime subprocesses and tool hosts are downstream dependencies of this package; `internal/session` decides when they receive prompts, stop signals, and environment exec requests.
- Attacker capabilities:
  - A caller with daemon/API access can control session IDs, prompt messages, turn sources, approval requests, and environment exec commands that reach this package.
  - A same-user local attacker could tamper with on-disk session metadata/event files if they already control the daemon home directory.
  - Attackers do not control generated session IDs or direct SQL text in this package.
- In-scope assets:
  - Session metadata and event-store path boundaries under `home/sessions`.
  - Correct lifecycle state transitions for stop/resume/query flows.
  - Prompt/event integrity and authorization boundaries around permission approvals and environment exec delegation.
- Out-of-scope:
  - Transport/authz policy before requests enter `internal/session`.
  - A fully compromised local user who already controls the daemon home directory or host filesystem.
  - Correctness/security of the trusted runtime subprocess or tool host after this package intentionally delegates a valid operation to it.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/session/manager_lifecycle.go:19` | `CreateOpts` fields from CLI/API callers (`AgentName`, `Name`, `Workspace`, `WorkspacePath`, `Channel`, `Type`). | `dispatchSessionPreCreate`, `resolveCreateWorkspace`, `aghconfig.ResolveAgentName`, `strings.TrimSpace`, and `normalizeSessionType` in `prepareCreateStart`. | Session metadata/session-dir setup and runtime launch through `startSession`. | LOW — rejected; values are normalized/validated before they reach filesystem paths or driver startup. |
| `internal/session/manager_prompt.go:22` and `internal/session/manager_prompt.go:92` | Prompt session IDs, message text, and turn source from callers. | `parsePromptRequest` requires non-nil context, non-empty session ID/message, and allowlisted turn sources; stored-session fallback now flows through `readMeta` validation. | `recordPromptInputEvent`, `lookupPromptSession`, and `m.driver.Prompt`. | LOW — rejected; prompt text is persisted and forwarded as data only, while session ID lookups use validated active/stored session resolution. |
| `internal/session/query.go:86`, `internal/session/query.go:108`, `internal/session/query.go:131`, `internal/session/transcript.go:14`, `internal/session/manager_lifecycle.go:38`, `internal/session/stop_reason.go:89`, `internal/session/manager_prompt.go:163` | Session IDs from control-plane callers across status/history/event/transcript/resume/stop/approval flows. | `normalizeStoredSessionID` in `readMeta` rejects blank, absolute, dot-segment, slash, and backslash-containing IDs before any on-disk lookup; in-memory lookups still use trimmed IDs only. | `store.SessionMetaFile(filepath.Join(...))` and `store.SessionDBFile(filepath.Join(...))` in `query.go`, plus lifecycle decisions keyed by session ID. | LOW — fixed; traversal-style IDs no longer reach session metadata or event-store path resolution. |
| `internal/session/environment_exec.go:30` | `EnvironmentExecRequest{SessionID, Command, Timeout}` from callers. | Trims and requires non-empty session ID and command, verifies active session/environment state, and requires a tool host before delegation. | `toolHost.CreateTerminal`, `WaitForTerminalExit`, and `TerminalOutput`. | LOW — rejected; this is an intentional privileged capability behind a trusted caller boundary, not an injection sink inside `internal/session`. |
| `internal/session/manager_prompt.go:163` | `acp.ApproveRequest` fields from a caller resolving a pending permission request. | `req.Validate()`, trimmed session ID, active-session lookup, and typed ACP error mapping. | `session.ApprovePermission` / driver-facing permission resolution. | LOW — rejected; request structure is validated and only forwarded to an already-owned active session. |
| `internal/session/query.go:196` and `internal/session/manager_lifecycle.go:49` | Persisted session metadata loaded back from disk on query/resume paths. | `store.ReadSessionMeta`, `repairInactiveMeta`, and `validateInfrastructure` bound the resumed/read state before reuse. | `sessionInfoFromMeta`, `prepareResumeStart`, and on-disk event-store reopen paths. | LOW — rejected; filesystem tampering requires same-user home-dir control, which is outside the declared threat model. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `SES-SEC-001` | security-review | high | `internal/session/query.go:196` | Stored-session queries accepted traversal-style session IDs and fed them into `SessionMetaFile` / `SessionDBFile` path resolution for metadata and event history access. | fixed |
| `SES-PERF-001` | extreme-software-optimization | medium | `internal/session/environment.go:287` | `dispatchEnvironmentSyncBefore` always walked the workspace to count files even when no environment hooks were configured to consume the payload. | fixed |
| `SES-CON-001` | deadlock-finder-and-fixer | high | `internal/session/stop_reason.go:111` | A failing `driver.Stop` could still fall into stop finalization if the process exited concurrently during the failed stop call, making `StopWithCause` block behind unrelated finalization work. | fixed |
| `SES-REF-001` | refactoring-analysis | medium | `internal/session/environment.go:1` | `environment.go` remains a 1100+ LOC unit that mixes environment lifecycle orchestration, metadata repair, sync bookkeeping, and helper utilities. | deferred |
| `SES-REF-002` | refactoring-analysis | medium | `internal/session/manager_hooks.go:1` | `manager_hooks.go` remains an 800+ LOC unit that concentrates multiple hook domains and repeated dispatch patterns. | deferred |
| `SES-REF-003` | refactoring-analysis | low | `internal/session/query.go:108` | `Events` and `History` still duplicate the same recorder open/defer/query wrapper shape. | wontfix |

## Per-Skill Notes

### refactoring-analysis

- `environment.go` and `manager_hooks.go` remain the two largest production units in the package and are still the best candidates for a focused follow-up split.
- I left the `Events`/`History` wrapper duplication in place because extracting another helper for those two short shells would add abstraction without reducing any real maintenance cost in this pass.
- The other duplication sites are mostly same-file/state-transition boilerplate; none justified a new abstraction under the task’s “no speculative abstractions” rule.

### extreme-software-optimization

- Added `internal/session/perf_bench_test.go` before changing production code so the optimization pass had measured baselines for each selected candidate.
- Fixed `SES-PERF-001` by skipping `environmentSyncFileCount` when the environment hook set is absent or explicitly no-op. `BenchmarkDispatchEnvironmentSyncBeforeNoHooks` improved from `269506.0 ns/op, 92776.8 B/op, 787 allocs/op` to `162.3 ns/op, 288 B/op, 1 alloc/op`.
- `BenchmarkManagerListAllLarge` and `BenchmarkSessionInfo` did not justify package-local optimization work. Their before/after results stayed within noise, so they are recorded as `not-hot-confirmed-by-benchmark`.

### ubs

- `not-run` due missing skill-runner support in this session; no CLI/manual substitute was used.

### deadlock-finder-and-fixer

- Fixed `SES-CON-001` by remembering whether the process was already exited before `driver.Stop` began and returning the stop error immediately when the failure happened while the process was still live. That keeps `StopWithCause` from blocking behind concurrent watcher finalization on the failure path.
- Regression coverage for the stop race comes from `TestStopWithCauseLifecycle`, which now passes repeatedly with `go test ./internal/session -run '^TestStopWithCauseLifecycle$' -count=20`.
- No additional goroutine leaks, channel ownership bugs, or missing cancellation branches were confirmed in the production inventory.

### security-review

- Threat model and attacker-input inventory were completed before the security verdict.
- Fixed `SES-SEC-001` by introducing `normalizeStoredSessionID`, which rejects absolute paths, dot segments, and path separators before stored-session lookups touch the filesystem.
- No remaining HIGH-confidence or MEDIUM-confidence exploit path survived the package-local review. The remaining surfaces are validated control-plane inputs, intended privileged capabilities, or same-user filesystem tampering outside the declared threat model.
- Package coverage is now `81.0%` via `go test -cover ./internal/session/...`, which clears the package target.

## Deferred Items (carry forward)

- `SES-REF-001` — Split `environment.go` only as a focused follow-up around environment sync/metadata helper boundaries; doing it opportunistically here would create large churn across a very stateful file.
- `SES-REF-002` — Split `manager_hooks.go` by hook domain only when there is room for a dedicated hook-orchestration cleanup pass.
- `SES-REF-003` — The `Events`/`History` duplication is low-value wrapper code and not worth another abstraction in this task.

## `make verify`

Command: `make verify`

Exit code: `2`

Package-local validation completed before the repo-wide gate:

- `go test ./internal/session/...` → pass
- `go test -cover ./internal/session/...` → pass (`81.0%`)
- `go test ./internal/session -run '^TestStopWithCauseLifecycle$' -count=20` → pass

Fresh repo-wide output excerpt from `/tmp/session-make-verify.txt`:

```text
Test Files  82 passed (82)
Tests  677 passed (677)
✓ built in 931ms
extensions/bridges/github/provider.go:1336:7: string `review` has 4 occurrences, make it a constant (goconst)
extensions/bridges/github/provider.go:1481:16: string `issue` has 5 occurrences, make it a constant (goconst)
2 issues:
* goconst: 2
Error: running "go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run --fix --allow-parallel-runners ./..." failed with exit code 1
make: *** [verify] Error 1
```

Blocked on unrelated repo-wide lint findings outside the allowed edit scope for this task:

- `extensions/bridges/github/provider.go:1336`
- `extensions/bridges/github/provider.go:1481`
