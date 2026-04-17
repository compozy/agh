# Improvements Report — internal/environment

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 3 benchmarks in `internal/environment/daytona/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo $(rg --files internal/environment -g '*.go' -g '!**/*_test.go') | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 16 | `(*restSSHTokenSource).FetchSSHAccess` | `internal/environment/daytona/ssh.go:58` |
| 13 | `sanitizeRemoteBase` | `internal/environment/daytona/shell.go:92` |
| 12 | `main` | `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:299` |
| 12 | `NewProvider` | `internal/environment/daytona/provider.go:40` |
| 12 | `(*sidecarSession).readLoop` | `internal/environment/daytona/sidecar_transport.go:668` |
| 11 | `collectArchiveEntries` | `internal/environment/daytona/tar.go:84` |
| 11 | `archivePatternMatches` | `internal/environment/daytona/tar.go:330` |
| 10 | `runLifecycle` | `internal/environment/providertest/suite.go:40` |
| 10 | `(*managedProcess).Stop` | `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:231` |
| 10 | `(*daytonaProvider).Prepare` | `internal/environment/daytona/provider.go:127` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/environment/daytona/sidecar_transport.go` | 775 | Sidecar bootstrap, tunnel setup, health polling, launch/connect flow, and session transport all live in one unit. |
| `internal/environment/daytona/provider.go` | 602 | Provider construction, prepare/find/destroy lifecycle, network policy validation, and request shaping are concentrated together. |
| `internal/environment/daytona/ssh.go` | 521 | Token retrieval, token cache policy, SSH dialing, session lifecycle, and host-key policy are bundled in one file. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go` | 501 | Process supervision, websocket framing, HTTP handlers, and sidecar server bootstrap are concentrated in one file. |
| `internal/environment/daytona/tar.go` | 369 | Archive collection, exclusion policy, extraction, path safety, and symlink validation all share one unit. |
| `internal/environment/daytona/tool_host.go` | 305 | File operations, permission handling, terminal lifecycle, and output buffering live together. |
| `internal/environment/types.go` | 303 | Backend enums, profile resolution structs, provider contracts, sync contracts, launch contracts, and tool-host contracts are all defined in one surface file. |

### Refactoring — Duplication

Manual duplication scan (`rg`, side-by-side inspection) found these notable ≥8-line repetitions:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/environment/daytona/sync.go:14-52` | `internal/environment/daytona/sync.go:55-96` | `SyncToRuntime` and `SyncFromRuntime` repeat backend/state/root setup and differ mainly in the per-root sync function. |
| `internal/environment/daytona/state.go:73-80` | `internal/environment/local/provider.go:155-162` | The same `cloneStrings` helper is duplicated across subpackages. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `ioCopyLimit` | `internal/environment/daytona/tool_host.go:272` | Allocation-heavy terminal output capture loop used while streaming remote terminal output into the bounded transcript buffer. | `BenchmarkIOCopyLimitSlidingWindow` |
| `writeTar` | `internal/environment/daytona/tar.go:31` | Sync-to-runtime archive construction walks the workspace tree and serializes every file and header. | `BenchmarkWriteTarWorkspaceTree` |
| `extractTar` | `internal/environment/daytona/tar.go:145` | Sync-from-runtime extraction is the main local I/O loop when pulling files back from Daytona. | `BenchmarkExtractTarWorkspaceTree` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/environment/...` before production fixes.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/environment/...` in the current workspace after all changes.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkIOCopyLimitSlidingWindow` | 2419941.0 | 37753390.8 | 398432.2 | 6194015.0 | fixed-with-benchmark |
| `BenchmarkWriteTarWorkspaceTree` | 236997.4 | 224425.6 | 235840.8 | 224425.6 | not-hot-confirmed-by-benchmark — no scoped tar writer change was justified. |
| `BenchmarkExtractTarWorkspaceTree` | 827758.0 | 62437.4 | 787287.4 | 62488.0 | not-hot-confirmed-by-benchmark — extraction stayed close enough to noise that no extra change was warranted. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/environment/daytona/ssh.go:406` | `sshSession` | Cancelled by `sshSession.Close`, exits on context cancellation. | Periodic SSH keepalive loop. |
| `internal/environment/daytona/ssh.go:407` | `sshSession` | Ends when `session.Wait()` returns and closes `sshSession.done`. | Wait goroutine owns the session completion signal. |
| `internal/environment/daytona/sidecar_transport.go:594` | `sidecarSession` | Ends on websocket close or `finish`, which closes `sidecarSession.done`. | Reads launcher-sidecar websocket frames and forwards stdout/stderr/exit. |
| `internal/environment/daytona/tool_host.go:184` | `daytonaToolHost` / `remoteTerminal` | Ends after `transportSession` EOF + `Wait`, then closes `remoteTerminal.done`. | Captures bounded terminal output for later retrieval. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:153` | `managedProcess` | Ends on stdout EOF and closes the chunk queue. | Streams child-process stdout into websocket frames. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:154` | `managedProcess` | Ends on stderr EOF. | Accumulates stderr text for exit/error frames. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:155` | `managedProcess` | Ends when `cmd.Wait()` returns and closes `managedProcess.done`. | Records child exit status. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:384` | `handleStream` | Ends when `chunkQueue.Pop()` reports closed. | Streams stdout websocket frames to the client. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:386` | `handleStream` | Ends after `process.done`, then closes `exitDone`. | Sends the final exit frame after the child exits. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/environment/daytona/ssh.go:403` | 0 | `sshSession` | SSH wait goroutine | `Done`, `Wait`, `Stop` | Session completion signal for SSH-backed transports. |
| `internal/environment/daytona/sidecar_transport.go:592` | 0 | `sidecarSession` | `finishOnce` | `Done`, `Wait`, `Stop` | Session completion signal for websocket-backed launcher sessions. |
| `internal/environment/daytona/tool_host.go:172` | 0 | `remoteTerminal` | `remoteTerminal.capture` | `WaitForTerminalExit` | Terminal completion signal used by the tool host. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:150` | 0 | `managedProcess` | `managedProcess.wait` | `managedProcess.Stop`, `streamExitFrame` | Child-process lifecycle completion signal. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:385` | 0 | `handleStream` | `streamExitFrame` | `handleStream` | Sidecar websocket loop exits when the process exit frame has been sent. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/environment/daytona/ssh.go:141` | write-heavy | `sshTokenManager.tokens` | Caches SSH tokens per `apiURL+sandboxID`. |
| `internal/environment/daytona/sidecar_transport.go:58` | read-heavy | `sidecarTransport.binaries` | Guards the architecture-keyed in-memory sidecar binary cache. |
| `internal/environment/daytona/sidecar_transport.go:567` | write-heavy | websocket writes | Serializes binary websocket frame writes. |
| `internal/environment/daytona/sidecar_transport.go:571` | write-heavy | `sidecarSession.stderr` | Protects stderr accumulation/reset. |
| `internal/environment/daytona/tool_host.go:26` | write-heavy | terminal map + next terminal ID | Guards terminal registration/lookup/release. |
| `internal/environment/daytona/tool_host.go:244` | write-heavy | `remoteTerminal.output` | Guards terminal output buffer snapshots and appends. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:56` | write-heavy | `chunkQueue.chunks` + `chunkQueue.closed` | Coordinates stdout chunk queue access with a condition variable. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:115` | write-heavy | `managedProcess.stderr` | Guards stderr text accumulation. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:278` | write-heavy | session/process map | Guards sidecar process registration and lookup. |

### Concurrency — Select Audit

Selects without `ctx.Done()`:

| File:Line | Reason |
| --- | --- |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:246` | `managedProcess.Stop` waits either for process completion or a fixed 5-second escalation timer; it is time-bounded but not context-aware. |
| `internal/environment/daytona/cmd/agh-daytona-sidecar/main.go:396` | `handleStream` performs a non-blocking `exitDone` check after each client frame; it is input-bounded by websocket reads rather than `ctx.Done()`. |

All other production `select` statements under `internal/environment/` are context-aware.

### Security — Threat Model

- Trust boundaries:
  - The daemon and session manager call provider methods with prepared environment metadata and sync directives.
  - ACP agents call the Daytona tool host with file paths, terminal commands, environment variables, and optional terminal working directories.
  - The package crosses into the Daytona SDK/API, SSH transport, launcher sidecar websocket transport, and the local filesystem during sync extraction.
- Attacker capabilities:
  - A malicious or compromised ACP agent can supply `CreateTerminalRequest` data (`Command`, `Args`, `Env`, `Cwd`) and file-operation paths to the tool host.
  - A remote runtime can influence archive contents returned by `SyncFromRuntime`.
  - Operators can configure Daytona profile values and environment variables, but those are treated as server-controlled rather than attacker-controlled unless explicitly forwarded by an agent request.
- In-scope assets:
  - Runtime-root confinement for file and terminal operations.
  - Confidentiality of blocked Daytona secrets (`DAYTONA_API_KEY`, `DAYTONA_JWT_TOKEN`) and integrity of forwarded runtime env vars.
  - Integrity of local workspace paths during archive extraction.
- Out-of-scope:
  - Daytona control-plane correctness after requests leave this package.
  - Authentication/authorization upstream of the daemon before a request becomes a provider/tool-host call.
  - A fully trusted operator intentionally configuring unsafe profile values.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/environment/daytona/tool_host.go:149-185` | ACP `CreateTerminalRequest` (`Command`, `Args`, `Env`, `Cwd`) from the agent. | `ResolvePath` now joins relative `Cwd` values to the runtime root and rejects escapes before `remoteTerminalCommand` renders the shell launch. Command and env entries remain shell-quoted / filtered. | `remoteTerminalCommand` -> `transport.Dial` starts a remote shell in the requested directory. | LOW — fixed in this pass: terminal launches are now confined to the runtime root and relative `Cwd` values resolve correctly. |
| `internal/environment/daytona/tool_host.go:56-93` | ACP file-operation paths (`ReadTextFile`, `WriteTextFile`). | `ResolvePath` joins relative paths to the runtime root and rejects escapes. | `sandbox.ReadFile` / `sandbox.WriteFile`. | LOW — rejected as safe because runtime-root checks happen before filesystem access. |
| `internal/environment/daytona/provider.go:116-165,440-467`, `internal/environment/daytona/env.go:14-49` | `PrepareRequest.AgentEnv` plus resolved profile env values. | `remoteEnvMap` forwards only `AGH_*` agent env vars, blocks Daytona secrets, and sorts final env output deterministically. | `createSandboxRequest.EnvVars` and launch env passed to the runtime. | LOW — rejected as safe because attacker-controlled env forwarding is allowlisted and secret keys are stripped. |
| `internal/environment/daytona/sync.go:113-180` | Sync roots and exclude patterns derived from daemon-controlled provider state / sync options. | Local roots are cleaned and archive writing excludes dangerous build-output trees; remote extract command is shell-quoted. | `writeTar` -> `remoteExtractCommand` -> SSH stream to the runtime. | LOW — rejected because the roots are daemon-owned provider state, not direct attacker input. |
| `internal/environment/daytona/sync.go:183-228`, `internal/environment/daytona/tar.go:145-313` | Remote archive payload returned by the runtime. | Tar extraction enforces root confinement, rejects traversal, refuses symlink overwrite, and validates symlink targets. | Local filesystem writes beneath the synced workspace root. | LOW — rejected as safe because archive extraction fails closed on traversal and symlink escape attempts. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | security-review | high | `internal/environment/daytona/tool_host.go:149` | `CreateTerminal` accepted escaped `Cwd` values and left relative working directories unresolved, so agent-supplied terminal launches could escape runtime-root confinement semantics. | fixed |
| 02 | extreme-software-optimization | medium | `internal/environment/daytona/tool_host.go:290` | `appendLimited` cloned the retained tail on every overflow, making bounded terminal capture the dominant measured allocation hotspot. | fixed |
| 03 | refactoring-analysis | medium | `internal/environment/daytona/sidecar_transport.go:1` | `sidecar_transport.go` remains a 775-LOC unit combining sidecar bootstrap, tunnel setup, websocket session handling, and transport lifecycle. | deferred — splitting transport/bootstrap/session seams would broaden this task into a larger architectural refactor. |
| 04 | refactoring-analysis | low | `internal/environment/daytona/sync.go:14` | `SyncToRuntime` and `SyncFromRuntime` repeat the same provider-state and root setup before diverging into direction-specific transfer logic. | deferred — extracting shared sync setup is worthwhile, but not without widening the sync API surface in this pass. |

## Per-Skill Notes

### refactoring-analysis

- The dominant structural signal in this package is concentration: large Daytona files (`sidecar_transport.go`, `provider.go`, `ssh.go`, `tar.go`, `tool_host.go`) continue to mix multiple responsibilities.
- The duplicated sync setup in `sync.go` is real, but untangling it cleanly would require reshaping the sync helper boundary rather than a small local edit.
- I deferred the structural work because this pass produced two higher-value in-scope fixes with clear correctness/performance payoff.

### extreme-software-optimization

- Added `internal/environment/daytona/perf_bench_test.go` so every selected hot path has a co-located benchmark.
- The benchmark signal was decisive for terminal output capture: `appendLimited` was rebuilding a new retained slice every time the output window overflowed.
- Fixed that by trimming the existing `bytes.Buffer` in place with `Next(...)`, while preserving the same last-`N` bytes behavior and the same unlimited-path semantics when `limit <= 0`.
- Isomorphism proof for the perf change:
  - Ordering preserved: yes — the buffer still retains the most recent bytes in arrival order.
  - Tie-breaking unchanged: yes — when a single chunk exceeds the limit, the last `limit` bytes of that chunk still win.
  - Floating-point: N/A.
  - RNG seeds: N/A.
  - Golden outputs: existing terminal-output tests plus the full `make verify` gate remained green.
- `BenchmarkIOCopyLimitSlidingWindow` improved from `2419941.0 ns/op, 37753390.8 B/op` to `398432.2 ns/op, 6194015.0 B/op`.
- `writeTar` and `extractTar` stayed effectively flat enough that no extra tar-path optimization was justified in this pass.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- The package has real concurrency surfaces, but they are explicit and owned: every goroutine in the Daytona transport, tool host, and sidecar command has a corresponding `done` channel, wait path, or close path.
- The two selects without `ctx.Done()` are bounded by a fixed escalation timer or websocket input loop, so I did not classify them as deadlock bugs in this pass.
- No goroutine/channel/mutex deadlock findings were confirmed from the inventories.

### security-review

- Fixed a high-confidence confinement bug in terminal creation by resolving agent-supplied `Cwd` values through `ResolvePath` before rendering the remote shell command.
- Source -> sink trace for the fixed issue:
  - Source: ACP `CreateTerminalRequest.Cwd` in [tool_host.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/environment/daytona/tool_host.go:149>).
  - Sink before the fix: `remoteTerminalCommand(...)` -> `transport.Dial(...)` in [tool_host.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/environment/daytona/tool_host.go:165>).
  - Fix: confine and normalize the requested working directory with `ResolvePath` in [tool_host.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/environment/daytona/tool_host.go:160>) before launching the terminal session.
- Added `TestDaytonaToolHostCreateTerminalResolvesCwdWithinRuntimeRoot` in [provider_test.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/environment/daytona/provider_test.go:465>) to cover both the escape rejection and the relative-path resolution path.
- No other high-confidence vulnerabilities were identified after tracing file operations, env forwarding, sync-to-runtime, and archive extraction paths.

## Deferred Items (carry forward)

- **03** — Split `internal/environment/daytona/sidecar_transport.go` across bootstrap, websocket session transport, and launcher tunnel responsibilities when a future task can absorb a larger Daytona transport refactor.
- **04** — Consolidate the repeated setup in `SyncToRuntime` and `SyncFromRuntime` if a follow-up task is willing to reshape the sync helper boundary.

## `make verify`

Final gate command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
DONE 4466 tests in 8.640s
OK: all package boundaries respected
```

Additional toolchain noise during the successful run:

- repeated Node warnings that `NO_COLOR` was ignored because `FORCE_COLOR` is set
- macOS linker warning from the lint toolchain: `ld: warning: -bind_at_load is deprecated on macOS`
