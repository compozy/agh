# Improvements Report — internal/subprocess

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/subprocess/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/subprocess | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 16 | `(*Process).Shutdown` | `internal/subprocess/process.go:342` |
| 16 | `TestInitializeBridgeRuntimeManagedInstanceHelpers` | `internal/subprocess/handshake_test.go:173` |
| 12 | `(InitializeRequest).Validate` | `internal/subprocess/handshake.go:157` |
| 12 | `(*transport).call` | `internal/subprocess/transport.go:151` |
| 11 | `(*Process).Call` | `internal/subprocess/process.go:308` |
| 10 | `validateInitializeResponse` | `internal/subprocess/handshake.go:193` |
| 10 | `(*Process).waitForExit` | `internal/subprocess/process.go:406` |
| 10 | `TestNilHelpersAndBufferUtilities` | `internal/subprocess/process_test.go:640` |
| 10 | `(*helperServer).run` | `internal/subprocess/process_test.go:797` |
| 9 | `(*transport).readLoop` | `internal/subprocess/transport.go:224` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/subprocess/process.go` | 567 | Launch, shutdown escalation, process-state bookkeeping, transport error fan-in, and stderr-tail helpers live in one unit. |
| `internal/subprocess/handshake.go` | 450 | Session DTOs, validation, normalization, cloning, and managed-instance helper methods are co-located in a single file. |
| `internal/subprocess/transport.go` | 423 | Frame IO, request dispatch, response tracking, pending-call lifecycle, and error fan-out share one transport implementation unit. |

### Refactoring — Duplication

Meaningful non-test duplicates from `dupl -plumbing -t 20 internal/subprocess/*.go`:

| Duplicate block | Notes |
| --- | --- |
| `internal/subprocess/process.go:378-396` | Repeated timeout escalation branches append the same wrapped error and `errors.Join` return shape for SIGTERM and SIGKILL phases. |
| `internal/subprocess/transport.go:352-366` | `sendResult` and `sendError` build nearly identical `rpcResponse` envelopes and forward both through `writeJSON`. |
| `internal/subprocess/handshake.go:197-217` | Three consecutive `validateSubset(...)` calls repeat the same acceptance-check pattern for actions, security, and provides. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*transport).writeJSON` | `internal/subprocess/transport.go:206` | Every outbound request and every host reply to a child request passes through this JSON framing path. | `BenchmarkTransportWriteJSONRequest` |
| `parseRPCID` numeric path | `internal/subprocess/transport.go:399` | Request/response dispatch parses an ID on every inbound frame before matching pending calls or routing host methods. | `BenchmarkParseRPCIDNumeric` |
| `CloneInitializeBridgeRuntime` | `internal/subprocess/handshake.go:357` | Bridge-capable callers clone managed runtime snapshots before handing them to long-lived manager state or provider code. | `BenchmarkCloneInitializeBridgeRuntime` |
| `(*boundedBuffer).Write` overflow path | `internal/subprocess/process.go:547` | The stderr tail buffer is the package-local streaming buffer; overflow writes are the only clearly allocation-heavy internal path. | `BenchmarkBoundedBufferWriteOverflow` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/subprocess/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/subprocess/...`

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkTransportWriteJSONRequest` | 209.3 | 176 | 209.9 | 176 | not-hot-confirmed-by-benchmark |
| `BenchmarkParseRPCIDNumeric` | 45.07 | 32 | 44.92 | 32 | not-hot-confirmed-by-benchmark |
| `BenchmarkCloneInitializeBridgeRuntime` | 3254 | 9488 | 3333 | 9488 | not-hot-confirmed-by-benchmark |
| `BenchmarkBoundedBufferWriteOverflow` | 1782 | 21760 | 1043 | 10240 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/subprocess/process.go:155` | `Launch` | Child `Wait()` completion closes `Process.done`. | Reaps child exit and folds transport/health finalization into one watcher goroutine. |
| `internal/subprocess/health.go:95` | `maybeStartHealthMonitor` | `p.lifecycleCtx.Done()` or `run.stopCh`, then `run.doneCh` closes. | Periodic probe loop; an in-flight probe currently ignores lifecycle cancellation and is a candidate bug. |
| `internal/subprocess/transport.go:126` | `(*transport).start` | `readLoop` exits on stdout EOF / decode error and closes `readerDone`. | Owns JSON-RPC frame ingestion from child stdout. |
| `internal/subprocess/transport.go:333` | `(*transport).handleRequest` | Handler goroutine relies on `t.process.lifecycleCtx` cancellation and handler cooperation. | Unbounded peer-controlled fan-out; likely resource-exhaustion surface if a child floods host requests. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/subprocess/process.go:86` | 0 | `Process` | `(*Process).waitForExit` | `Done`, `Wait`, `Shutdown`, `Call`, `waitWithContext` | Process-lifecycle completion signal. |
| `internal/subprocess/process.go:270` | 0 | `(*Process).Done` nil fallback | `(*Process).Done` immediately | External caller only | Closed sentinel returned for `(*Process)(nil)`. |
| `internal/subprocess/health.go:35` | 0 | `healthMonitorRun` | `(*healthMonitorRun).stop` via `sync.Once` | `runHealthMonitor` | Health-monitor stop signal. |
| `internal/subprocess/health.go:36` | 0 | `healthMonitorRun` | `(*healthMonitorRun).finish` | `stopHealthMonitor` | Join signal for health-monitor goroutine. |
| `internal/subprocess/transport.go:78` | 0 | `transport` | `(*transport).readLoop` | `(*transport).shutdown` | Read-loop completion signal. |
| `internal/subprocess/transport.go:153` | 1 | `(*transport).call` | `handleResponse` / `closePending` | `(*transport).call` | Per-request response delivery channel stored in `pending`. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/subprocess/process.go:87` | read-heavy | `waitErr` | Read after `Done()` closes; written once in `waitForExit`. |
| `internal/subprocess/process.go:90` | read-heavy | `stopRequested` | Checked during exit-collapse paths and shutdown initiation. |
| `internal/subprocess/process.go:91` | write-heavy | `stdin` close / `inputClosed` | Prevents double-closing stdin across shutdown and transport failure paths. |
| `internal/subprocess/process.go:94` | read-heavy | `state` | Guards process state transitions and request gating. |
| `internal/subprocess/process.go:97` | read-heavy | `transportErr` | Keeps the first transport failure for later `Wait`/`Call` propagation. |
| `internal/subprocess/process.go:542` | write-heavy | `boundedBuffer.buf` | Serializes stderr tail updates and reads. |
| `internal/subprocess/health.go:29` | read-heavy | `health.state` / `health.active` | Shares monitor state between probe loop, `HealthState`, and shutdown. |
| `internal/subprocess/transport.go:70` | read-heavy | `handlers` | Guards method registry lookups while requests stream in. |
| `internal/subprocess/transport.go:73` | write-heavy | `pending` | Tracks outstanding calls and their response channels. |
| `internal/subprocess/transport.go:76` | write-heavy | stdin frame writes | Keeps JSON-RPC frames serialized on one pipe writer. |

### Concurrency — Select Audit

- `internal/subprocess/process.go:350` is a non-blocking done probe before shutdown escalation; it never parks and does not need a `ctx.Done()` branch.
- `internal/subprocess/process.go:398` waits on `p.Done()` and `ctx.Done()`.
- `internal/subprocess/process.go:446` waits on `p.Done()`, timeout context, and `ctx.Done()`.
- `internal/subprocess/health.go:105` waits on `p.lifecycleCtx.Done()`, `run.stopCh`, and ticker delivery.
- `internal/subprocess/transport.go:172` waits on the response channel, `ctx.Done()`, and `t.process.Done()`.

All long-lived blocking selects are context- or lifecycle-aware.

### Security — Threat Model

- Trust boundaries:
  - `internal/subprocess` launches and supervises external child binaries for ACP clients, extensions, and bridge providers; those binaries communicate over stdio JSON-RPC.
  - Child stdout and stdin are the primary trust boundary inside this package; peer processes can be third-party or compromised and should not be assumed perfectly well-behaved.
  - Bridge-runtime helpers move daemon-owned instance metadata and resolved secret material into child-session initialize payloads.
- Attacker capabilities:
  - A malicious child process can emit arbitrary JSON-RPC frames on stdout, choose request cadence, params, IDs, and response bodies up to the configured max-message limit.
  - Operator-provided config controls `LaunchConfig.Command`, `Args`, `Dir`, and `Env`.
  - Daemon-resolved bridge secret values can contain arbitrary bytes.
- In-scope assets:
  - Host-process stability and bounded resource use while reading/writing protocol frames.
  - Correct lifecycle transitions and shutdown latency for managed child processes.
  - Integrity of managed bridge runtime snapshots and bound secret payloads handed to child processes.
- Out-of-scope:
  - Authenticating or attesting extension binary provenance after launch.
  - Semantic validation of method-specific JSON payloads handled by caller packages.
  - Authorization decisions in higher-level packages that decide which host methods are exposed to the child.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/subprocess/process.go:195` | Operator/caller-provided `LaunchConfig.Command`, `Args`, `Dir`, and `Env`. | `execabs.LookPath` resolves the binary path and execution uses `exec.Cmd` directly, without a shell. | `cmd.Start()` in `startManagedCommand`. | LOW — operator-controlled spawn path; no shell interpolation or relative-path execution survives. |
| `internal/subprocess/transport.go:206` | In-process caller-controlled JSON-RPC method/params. | `json.Marshal` plus `maxMessageBytes` cap before write. | Child stdin via `t.process.stdin.Write`. | LOW — same-process boundary; framing is size-bounded and never shells out. |
| `internal/subprocess/transport.go:227` | Child stdout frame bytes from a launched subprocess. | Scanner cap (`t.maxMessageBytes+1`), JSON decode, JSON-RPC version check, and `parseRPCID` validation. | `handleRequest` / `handleResponse`, including host handler fan-out at `internal/subprocess/transport.go:333`. | MEDIUM — child-controlled inbound requests currently spawn unbounded handler goroutines and can pressure host memory/CPU. |
| `internal/subprocess/health.go:121` | Child-controlled `health_check` response timing and payload. | JSON decode into `HealthCheckResponse`; probe timeout now derives from `p.lifecycleCtx` instead of `context.Background()`. | Health-state mutation and shutdown join path. | MEDIUM — fixed in-package; in-flight probes now cancel promptly during shutdown instead of waiting for the full timeout window. |
| `internal/subprocess/handshake.go:348` | Daemon-resolved bridge secret values. | Identifier fields are normalized; `Value` is currently trimmed as part of normalization. | Managed runtime cloning and delivery into child initialize payloads. | LOW — integrity-sensitive asset, but no externally exploitable source-to-sink break was proven during this pass. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `SUB-OPT-001` | extreme-software-optimization | low | `internal/subprocess/process.go:547` | `boundedBuffer.Write` no longer forces a fresh allocation before every trim; the overflow-path benchmark median dropped from `1782 ns/op` / `21760 B/op` to `1043 ns/op` / `10240 B/op`. | fixed |
| `SUB-CON-001` | deadlock-finder-and-fixer | medium | `internal/subprocess/health.go:117` | Health probes now derive from the process lifecycle context, so `stopHealthMonitor` cancels an in-flight probe instead of waiting for the full timeout window. | fixed |
| `SUB-REF-001` | refactoring-analysis | medium | `internal/subprocess/process.go:342` | `(*Process).Shutdown` remains the package’s densest production routine and repeats the same escalation/error-join structure across phases. | deferred |
| `SUB-SEC-001` | security-review | medium | `internal/subprocess/transport.go:227` | A child that floods inbound host requests can force unbounded handler goroutine fan-out through `handleRequest` at `transport.go:333`. | deferred |

## Per-Skill Notes

### refactoring-analysis

- Baseline package coverage is `82.6%`, so this pass starts above the repository floor.
- `process.go`, `handshake.go`, and `transport.go` are all well above the 300-LOC threshold and drive most of the package cyclomatic complexity.
- The duplication scan is noisy because the package has large tests, but the listed non-test clones are the only meaningful production duplicates that survived manual triage.
- `SUB-REF-001` remains deferred because refactoring `(*Process).Shutdown` safely would require a follow-up extraction pass across the lifecycle/error-collation code after the concurrency and performance fixes settle.

### extreme-software-optimization

- Added `internal/subprocess/perf_bench_test.go` before any production edit so every optimization decision starts from measured baselines.
- `boundedBuffer.Write` was the only hot-path candidate with a worthwhile package-local lever; removing the forced reallocation path cut the median overflow benchmark from `1782 ns/op` / `21760 B/op` to `1043 ns/op` / `10240 B/op`.
- `writeJSON`, `parseRPCID`, and `CloneInitializeBridgeRuntime` all stayed in roughly the same range after the functional fixes, so they are recorded as measured `not-hot-confirmed-by-benchmark` candidates rather than speculative optimization work.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute will be used.

### deadlock-finder-and-fixer

- The package has four production goroutine entry points and six channel declarations, so concurrency review is material even though there is no classical mutex deadlock in the current tests.
- `TestStopHealthMonitorCancelsInFlightProbe` now proves the fixed contract: once shutdown cancels the process lifecycle, the in-flight probe tears down promptly and the pending response channel is removed.
- The deferred concurrency/security risk is the unbounded inbound host-request fan-out from `handleRequest`, which needs a transport-level backpressure design rather than a local micro-patch.

### security-review

- The highest-confidence package-local security concern is resource exhaustion from child-controlled inbound request fan-out.
- Secret-payload normalization is recorded in the surface inventory because the package handles daemon-owned secret material, but it did not reach a confirmed exploitable finding in this pass.
- No additional HIGH-confidence vulnerability survived the post-fix review beyond the deferred fan-out issue.

## Deferred Items (carry forward)

- **`SUB-REF-001`** — Extracting the repeated shutdown escalation/error-join phases out of `(*Process).Shutdown` would improve maintainability, but it should be handled as a dedicated lifecycle refactor after this pass to avoid mixing structural change with correctness/perf fixes.
- **`SUB-SEC-001`** — Bounded backpressure for child-originated host requests needs a transport design that does not starve child responses behind queued requests on the same stdout stream. That is larger than this package pass and should be tackled with caller/transport contract review.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/subprocess (5.331s)
DONE 4501 tests in 21.432s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0`.
