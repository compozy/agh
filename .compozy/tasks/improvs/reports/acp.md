# Improvements Report — internal/acp

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 4 benchmarks in `internal/acp/acp_bench_test.go`, before/after numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surfaces below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo internal/acp | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 37 | `(*helperACPAgent).Prompt` | `internal/acp/client_test.go:1189` |
| 24 | `TestPermissionHelperBranches` | `internal/acp/handlers_test.go:1114` |
| 22 | `TestHelperUtilities` | `internal/acp/handlers_test.go:940` |
| 19 | `TestHandleInboundPermissionRequest` | `internal/acp/handlers_test.go:335` |
| 18 | `TestNetworkTurnTerminalOwnershipGuards` | `internal/acp/handlers_test.go:775` |
| 17 | `TestACPIntegrationRequestPermissionPolicy` | `internal/acp/client_integration_test.go:101` |
| 16 | `TestEndPromptClearsActivePromptWhileEmitterIsBackpressured` | `internal/acp/types_test.go:9` |
| 14 | `TestEmitPermissionEvent` | `internal/acp/handlers_test.go:497` |
| 13 | `translateSessionUpdate` | `internal/acp/handlers.go:713` |
| 13 | `TestTerminalLifecycleHandlers` | `internal/acp/handlers_test.go:687` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/acp/handlers.go` | 1003 | Dispatch, terminal lifecycle, permission flow, session update translation, and utility helpers are co-located. |
| `internal/acp/client.go` | 955 | Driver construction, process launch, session negotiation, prompt orchestration, stop handling, and env normalization are mixed in one unit. |
| `internal/acp/permission.go` | 539 | Permission policy, path sandboxing, pending-request lifecycle, and event serialization are tightly coupled. |
| `internal/acp/types.go` | 493 | Public types, process state, prompt buffering, and byte-buffer helpers share one file. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/acp` findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/acp/handlers.go:408-418` | `internal/acp/handlers.go:477-487` | Production duplicate between terminal kill and release handlers. |
| `internal/acp/client_test.go:1061-1087` | `internal/acp/client_test.go:1075-1104` | Test-only duplicate across captured session payload decoders. |
| `internal/acp/client_test.go:1061-1073` | `internal/acp/client_test.go:1075-1087` | Test-only duplicate in shared JSON decode setup. |
| `internal/acp/client_test.go:1075-1087` | `internal/acp/client_test.go:1089-1104` | Test-only duplicate in captured set-mode decode setup. |
| `internal/acp/client_test.go:1089-1104` | `internal/acp/client_test.go:1061-1073` | Test-only duplicate in captured payload decode setup. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*AgentProcess).handleSessionUpdate` | `internal/acp/handlers.go:299` | Runs on every inbound ACP `session/update` event and allocates heavily under message streaming. | `BenchmarkHandleSessionUpdateAgentMessage` |
| `(*managedTerminal).appendOutput` | `internal/acp/handlers.go:671` | Terminal IO loop appends and truncates output on every stdout/stderr write; baseline alloc pressure is high. | `BenchmarkManagedTerminalAppendOutputOverflow` |
| `permissionPolicy.resolvePath` | `internal/acp/permission.go:139` | Every file and permission location request passes through workspace path normalization and root checks. | `BenchmarkPermissionPolicyResolvePathExistingRelative` |
| `mergeCommandEnv` | `internal/acp/handlers.go:877` | Used during terminal creation to compose environment state for spawned commands. | `BenchmarkMergeCommandEnvWithOverrides` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/acp/...`:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkHandleSessionUpdateAgentMessage` | 5176.8 | 3544 | 4955.0 | 3544 | deferred — ACP SDK JSON decode still dominates and no local one-step win was justified. |
| `BenchmarkManagedTerminalAppendOutputOverflow` | 14219.0 | 155648 | 574.3 | 0 | fixed-with-benchmark |
| `BenchmarkPermissionPolicyResolvePathExistingRelative` | 10004.0 | 4808 | 13322.4 | 4808 | not-hot-confirmed-by-benchmark — filesystem/canonicalization path is cold relative to prompt and terminal loops. |
| `BenchmarkMergeCommandEnvWithOverrides` | 963.9 | 1433 | 1879.8 | 1433 | not-hot-confirmed-by-benchmark — launch-time setup path is short and not worth deeper churn in this pass. |

Profiling notes:
- `go tool pprof -top /tmp/acp-term.mem` attributes `99.92%` of alloc space to `(*managedTerminal).appendOutput` and `trimUTF8LeadingBytes`.
- `go tool pprof -top /tmp/acp-session.mem` shows `handleSessionUpdate` dominated by repeated JSON unmarshalling through the ACP SDK, making it a weaker immediate optimization target.
- `go tool pprof -top /tmp/acp-term-after.mem` no longer shows `internal/acp` functions as meaningful alloc-space hotspots after the terminal buffering fix.

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/acp/client.go:236` | `Driver.launchAgentProcess` | `waitForExit` closes `AgentProcess.done`; tied to process lifecycle. | Tracks the launched ACP subprocess until exit. |
| `internal/acp/client.go:471` | `Driver.Prompt` | `runPrompt` returns after ACP request completes and `defer proc.endPrompt(active)` closes the prompt channel. | One goroutine per in-flight prompt turn. |
| `internal/acp/client.go:604` | `Driver.runPrompt` | Exits on either `ctx.Done()` or `close(cancellationDone)`. | Cancellation helper for cooperative `session/cancel` notification. |
| `internal/acp/handlers.go:562` | `terminalManager.create` | `managedTerminal.wait` exits when `cmd.Wait()` returns and closes `term.done`. | One goroutine per managed terminal process. |
| `internal/acp/handlers.go:977` | `watchTerminalShutdown` | Exits on `ctx.Done()` or `terminalDone`. | Manager-level terminal shutdown watcher. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/acp/client.go:222` (`AgentProcess.done`) | unbuffered | `AgentProcess` lifecycle | `waitForExit` (`internal/acp/client.go:696`) | `AgentProcess.Done`, `Driver.Stop`, `stopExecCommand` | Signals ACP subprocess exit. |
| `internal/acp/types.go:244` (`activePrompt.events`) | `promptBufferCap` | One prompt turn | `endPrompt` (`internal/acp/types.go:353`) | `Driver.Prompt` caller stream consumer | Stream of prompt events. |
| `internal/acp/types.go:245` (`activePrompt.activity`) | 1 | One prompt turn | never explicitly closed | `waitForPromptQuiescence` | Non-blocking activity pulse channel for trailing update drain. |
| `internal/acp/types.go:257` (`pendingPermission.response`) | 1 | One pending permission request | not explicitly closed; map entry removed after resolution/timeout | `handleRequestPermission` select | Carries approval decision back to request handler. |
| `internal/acp/handlers.go:105` (`managedTerminal.done`) | unbuffered | One managed terminal | `managedTerminal.wait` (`internal/acp/handlers.go:698`) | `terminalManager.wait`, `watchTerminalShutdown` | Signals terminal process completion. |
| `internal/acp/handlers.go:971` (`watcherDone`) | unbuffered | `watchTerminalShutdown` helper | watcher goroutine | currently unused by callers | Completion notification for the shutdown watcher itself. |
| `internal/acp/client.go:603` (`cancellationDone`) | unbuffered | `runPrompt` helper goroutine | `runPrompt` after ACP request (`internal/acp/client.go:643`) | cancellation helper goroutine | Prevents cancellation watcher leak after prompt completion. |
| `internal/acp/launcher.go:124` (nil-handle `done`) | unbuffered | `localProcessHandle.Done` fallback | immediate close in same function | caller of `Done()` | Safe closed channel for nil handles. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/acp/handlers.go:89` (`terminalManager.mu`) | read-heavy | `terminalManager.terminals` map | Guards terminal lookup/add/remove and close-all snapshotting. |
| `internal/acp/handlers.go:101` (`managedTerminal.mu`) | write-heavy | `managedTerminal.output`, `truncated`, `exitStatus` | Shared by output writes and snapshot/wait completion. |
| `internal/acp/types.go:207` (`toolHostMu`) | write-heavy | lazy `toolHost` initialization/access | Serializes default host construction. |
| `internal/acp/types.go:217` (`terminalOwnershipMu`) | read-heavy | `terminalOwnership` map | Tracks network-turn terminal ownership. |
| `internal/acp/types.go:220` (`waitMu`) | read-heavy | `waitErr` | Protects final wait error state. |
| `internal/acp/types.go:224` (`stopMu`) | read-heavy | `stopRequested` | Avoids racing stop/exit bookkeeping. |
| `internal/acp/types.go:226` (`promptMu`) | read-heavy | `activePrompt` pointer | Coordinates prompt lifecycle and event routing. |
| `internal/acp/types.go:229` (`pendingPermissionMu`) | write-heavy | `pendingPermissions`, `permissionRequestSeq` | Serializes pending permission registration and resolution. |
| `internal/acp/types.go:234` (`systemPromptMu`) | write-heavy | `systemPrompt`, `systemPromptSent` | Ensures one-time system prompt prepending. |
| `internal/acp/types.go:238` (`turnSourceProviderMu`) | read-heavy | `turnSourceProvider` callback | Protects provenance callback swaps/reads. |
| `internal/acp/types.go:247` (`activePrompt.sendMu`) | write-heavy | prompt channel close/send exclusion | Prevents send-on-closed-channel during prompt teardown. |
| `internal/acp/types.go:250` (`activePrompt.usageMu`) | write-heavy | accumulated prompt `TokenUsage` | Merges usage updates safely across async notifications. |
| `internal/acp/types.go:261` (`lockedBuffer.mu`) | write-heavy | stderr byte buffer | Used by buffer appends and string snapshots. |

### Concurrency — Select Audit

- `internal/acp/client.go:520` — bounded preflight check on `proc.Done()` before stop escalation; no `ctx.Done()` needed because it does not block.
- `internal/acp/client.go:563` — includes `ctx.Done()`.
- `internal/acp/client.go:582` and `internal/acp/client.go:588` — include `ctx.Done()` on nested wait/kill path.
- `internal/acp/client.go:605` — includes `ctx.Done()` and `cancellationDone`.
- `internal/acp/client.go:940` and `internal/acp/client.go:943` — bounded by timers and non-blocking timer-drain logic rather than context.
- `internal/acp/handlers.go:281` — includes `ctx.Done()` and timeout.
- `internal/acp/handlers.go:611` — includes `ctx.Done()`.
- `internal/acp/handlers.go:979` — includes `ctx.Done()` and terminal completion.
- `internal/acp/types.go:426` — non-blocking activity pulse with `default`; no blocking wait.

### Security — Threat Model

- Trust boundaries:
  - Higher-level AGH packages call exported `acp` APIs to launch agents and stream events.
  - ACP agent subprocesses send JSON-RPC requests/notifications over stdio into `handleInbound`.
  - Local filesystem and terminal operations execute through `ToolHost` implementations.
- Attacker capabilities:
  - A malicious or compromised ACP agent can send tool requests (`fs/*`, `terminal/*`, `session/request_permission`, `session/update`) with arbitrary request bodies.
  - Higher layers can supply `StartOpts` command/cwd/env/additional-dir values, but those are treated as server-controlled configuration rather than direct attacker input.
- In-scope assets:
  - Workspace filesystem boundary enforcement.
  - Terminal execution controls during agent turns, especially network-originated turns.
  - Permission request bookkeeping and event integrity for session observers.
- Out-of-scope:
  - A fully trusted operator intentionally configuring a dangerous agent command in `StartOpts`.
  - Compromise of the local OS or upstream ACP agent binary itself beyond the permission model this package enforces.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/acp/handlers.go:220` + `internal/acp/tool_host.go:87` + `internal/acp/permission.go:139` | ACP `fs/read_text_file` request path from agent | `Authorize(read)` plus `ResolvePath` canonicalization and `isWithinRoot` workspace check | `os.ReadFile` in `ReadTextFile` | LOW — attacker-controlled path is constrained to the canonical workspace root. |
| `internal/acp/handlers.go:231` + `internal/acp/tool_host.go:102` + `internal/acp/permission.go:139` | ACP `fs/write_text_file` request path and content from agent | network-turn writes are blocked; `Authorize(write)` plus `ResolvePath` canonicalization and `isWithinRoot` | `os.MkdirAll` / `os.WriteFile` in `WriteTextFile` | LOW — write capability is intentionally permission-gated and path-sandboxed. |
| `internal/acp/handlers.go:364` + `internal/acp/tool_host.go:140` + `internal/acp/handlers.go:567` | ACP `terminal/create` command, args, cwd, env from agent | permission gate, workspace path resolution for `cwd`, `execabs.LookPath`, and network-turn command allowlist | `exec.Cmd` launch in `terminalManager.create` | REJECTED — arbitrary terminal execution is the package’s intended capability for trusted agent turns; network turns are explicitly constrained by allowlist. |
| `internal/acp/handlers.go:244` + `internal/acp/permission.go:119` + `internal/acp/permission.go:354` | ACP `session/request_permission` tool metadata, locations, and option list | location paths re-resolved through `resolvePathList`; interactive approvals require explicit local decision | permission event emission and `pendingPermission.response` decision channel | LOW — surface is mediated by root-bounded path checks and human/daemon approval flow. |
| `internal/acp/handlers.go:299` + `internal/acp/handlers.go:713` | ACP `session/update` notification payload from agent | JSON decode into typed ACP structures before `AgentEvent` projection | internal event stream via `emitPromptEvent` | LOW — malformed payloads fail closed via JSON unmarshal errors; decoded content is not executed. |
| `internal/acp/client.go:699` + `internal/acp/client.go:166` + `internal/acp/launcher.go:56` | Higher-layer `StartOpts` command, cwd, env, additional dirs | `Validate`, `normalizeWorkspaceDir`, `normalizeAdditionalDirs`, and server-controlled config boundary | subprocess launch and ACP session bootstrap | REJECTED — treated as operator-controlled configuration rather than attacker-controlled input within this package’s threat model. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | extreme-software-optimization | medium | `internal/acp/handlers.go:671` | Terminal output overflow reallocated large slices and could discard buffered output when the newest chunk ended with a partial UTF-8 rune. | fixed |
| 02 | refactoring-analysis | medium | `internal/acp/client.go:166` | `client.go` remains a large multi-responsibility unit spanning launch, session negotiation, stop flow, and prompt orchestration. | deferred |
| 03 | refactoring-analysis | low | `internal/acp/handlers.go:408` | Terminal kill/release handlers are duplicated. | wontfix |
| 04 | extreme-software-optimization | low | `internal/acp/handlers.go:299` | Session-update handling remains allocation-heavy because ACP SDK JSON unmarshal dominates the path. | deferred |

## Per-Skill Notes

### refactoring-analysis

- Production complexity is moderate; only `translateSessionUpdate` appears in the top-10 because the highest cyclomatic scores are concentrated in tests.
- The large-file inventory is real, but splitting `client.go` and `handlers.go` would be broad structural churn for this pass; recorded as deferred follow-up instead of mixed into the measured runtime fix.
- The duplicate terminal kill/release handlers were left as `wontfix` for now because extracting a helper would save little code while adding indirection to two short methods.

### extreme-software-optimization

- Fixed `(*managedTerminal).appendOutput` by replacing the overflow path with a bounded-window helper that reuses a fixed-size buffer and trims only invalid leading UTF-8 prefix bytes.
- Added regression tests covering the prior failure mode where a trailing partial rune could erase the retained buffer on overflow.
- Bench result for the fixed path improved from `14219.0 ns/op, 155648 B/op` to `574.3 ns/op, 0 B/op`.
- `handleSessionUpdate` was profiled but left unchanged because the cost is dominated by ACP SDK JSON decoding, not by an isolated local helper.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No deadlock or goroutine-leak finding was confirmed after auditing the goroutine/channel/mutex/select inventories.
- Every long-lived goroutine has an explicit exit signal (`ctx.Done()`, process completion, or terminal completion), and the select statements without `ctx.Done()` are bounded or non-blocking by construction.

### security-review

- No high-confidence vulnerabilities identified.
- All attacker-controlled filesystem surfaces are mediated by permission checks plus canonical workspace-root enforcement.
- Arbitrary terminal execution is an intended capability for trusted agent turns and is further restricted by an allowlist for network-originated turns.

## Deferred Items (carry forward)

- **02** — Split `internal/acp/client.go` into smaller files if a future task can absorb higher-churn refactoring without mixing it into behavioral fixes.
- **04** — Revisit `handleSessionUpdate` only if a later task is willing to benchmark alternative ACP decode strategies or upstream SDK changes.

## `make verify`

Fresh verification command: `make verify`

```text
✓  internal/hooks (1.688s)
✓  internal/acp (4.497s)
✓  internal/session (16.036s)
✓  internal/cli (16.794s)
✓  internal/extension (18.605s)
✓  internal/daemon (20.158s)

DONE 4426 tests in 21.622s
OK: all package boundaries respected
```
