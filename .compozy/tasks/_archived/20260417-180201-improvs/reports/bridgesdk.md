# Improvements Report — internal/bridgesdk

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmark inventory and results below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 $(rg --files internal/bridgesdk -g '*.go' -g '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 19 | `NewWebhookHandler` | `internal/bridgesdk/webhook.go:145` |
| 15 | `(*Peer).Call` | `internal/bridgesdk/peer.go:101` |
| 12 | `RetryDo` | `internal/bridgesdk/errors.go:312` |
| 12 | `(*Peer).Serve` | `internal/bridgesdk/peer.go:161` |
| 11 | `(*InboundBatcher).Enqueue` | `internal/bridgesdk/batching.go:93` |
| 10 | `(*Peer).dispatchRequest` | `internal/bridgesdk/peer.go:218` |
| 8 | `(*Runtime).handleInitialize` | `internal/bridgesdk/runtime.go:238` |
| 8 | `(*InboundBatcher).FlushAll` | `internal/bridgesdk/batching.go:159` |
| 7 | `(*Runtime).handleShutdown` | `internal/bridgesdk/runtime.go:329` |
| 7 | `(*Runtime).Serve` | `internal/bridgesdk/runtime.go:82` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/bridgesdk/runtime.go` | 400 | Runtime bootstrap, session lifecycle, request decoding, host-api wiring, and handler dispatch sit in one file. |
| `internal/bridgesdk/peer.go` | 381 | JSON-RPC framing, request routing, response correlation, transport error fan-out, and ID parsing are bundled together. |
| `internal/bridgesdk/errors.go` | 378 | Provider error taxonomy, classification, recovery mapping, retry policy, and retry timing helpers are co-located. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/bridgesdk` surfaced only a test-only duplicate in `runtime_integration_test.go`; no production duplication meeting that threshold was detected.

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `InboundBatchKey` | `internal/bridgesdk/batching.go:217` | Runs on every inbound envelope enqueued for short-burst batching and currently builds a composite routing key string. | `BenchmarkInboundBatchKey` |
| `(*InstanceCache).Snapshot` | `internal/bridgesdk/cache.go:65` | Allocation-heavy clone path used whenever providers expose the negotiated managed-instance runtime snapshot. | `BenchmarkInstanceCacheSnapshot` |
| `(*Peer).Call` | `internal/bridgesdk/peer.go:101` | Core Host API round-trip path for provider runtimes sharing the stdio JSON-RPC transport. | `BenchmarkPeerCallRoundTrip` |
| `(*FixedWindowRateLimiter).Allow` | `internal/bridgesdk/webhook.go:78` | Executes on every guarded webhook request and maintains attacker-influenced per-key limiter state. | `BenchmarkFixedWindowRateLimiterAllow` |

### Optimization — Benchmark Results

Baseline and post-fix averages from `go test -bench=. -benchmem -count=5 ./internal/bridgesdk/...`:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkInboundBatchKey` | 207.12 | 176 | 74.97 | 64 | fixed-with-benchmark |
| `BenchmarkInstanceCacheSnapshot` | 3564.80 | 12016 | 1437.00 | 5488 | fixed-with-benchmark |
| `BenchmarkPeerCallRoundTrip` | 8721.20 | 2836 | 8779.20 | 2836 | not-hot-confirmed-by-benchmark — the panic fix changed correctness only; round-trip transport cost stayed effectively flat. |
| `BenchmarkFixedWindowRateLimiterAllow` | 47.72 | 0 | 51.11 | 0 | not-hot-confirmed-by-benchmark — the stale-key eviction hardening added a small constant check but preserved allocation-free steady-state behavior. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/bridgesdk/peer.go:197` | `Peer.Serve` | Request goroutines inherit the serve context; `Peer.Serve` closes pending response channels and waits on the `WaitGroup` before returning. | One explicit goroutine per inbound request frame. Timer callbacks in `InboundBatcher` are runtime-managed via `time.AfterFunc`, not explicit `go` statements. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/bridgesdk/peer.go:56` (`pending` map values) | 1 per call | `Peer.Call` | `handleResponse`, `closePending`, or context-canceled caller path | Waiting `Peer.Call` invocations | Response correlation channels live only for one in-flight request id. |
| `internal/bridgesdk/peer.go:116` (`responseCh`) | 1 | `Peer.Call` | `handleResponse` or `closePending` | The originating `Peer.Call` select | Buffered single-response channel prevents responder goroutines from blocking on result delivery. |
| `internal/bridgesdk/webhook.go:59` (`sem`) | configured limit | `InFlightLimiter` | never explicitly closed; tokens released through `Release` | `TryAcquire` / `Release` | Buffered semaphore bounds concurrent webhook handling. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/bridgesdk/batching.go:52` | write-heavy | Pending batches, close/error state, and timer coordination | Shared by enqueue, flush, and close paths. |
| `internal/bridgesdk/cache.go:17` | read-heavy | Managed-instance runtime snapshot and metadata fields | Snapshot/list/get use `RLock`, reset/sync use `Lock`. |
| `internal/bridgesdk/dedup.go:16` | write-heavy | TTL dedup map | Every mark/clear operation takes the mutex. |
| `internal/bridgesdk/peer.go:52` | read-heavy | Registered JSON-RPC handlers | Write on registration, read on dispatch. |
| `internal/bridgesdk/peer.go:55` | write-heavy | Pending response-channel registry | Guards request correlation lifecycle. |
| `internal/bridgesdk/peer.go:58` | write-heavy | Encoder writes to the shared transport | Serializes outbound frame writes. |
| `internal/bridgesdk/peer.go:62` | write-heavy | Transport error accumulation | Guards the terminal transport error chain. |
| `internal/bridgesdk/runtime.go:46` | read-heavy | Runtime peer/session pointers | Session reads are concurrent with initialize/serve bootstrap. |
| `internal/bridgesdk/webhook.go:45` | write-heavy | Fixed-window per-key counter map | Guards request counts and eventual expiry state. |

### Concurrency — Select Audit

- `internal/bridgesdk/errors.go:344` waits on `ctx.Done()` or the retry timer.
- `internal/bridgesdk/peer.go:134` waits on `ctx.Done()` or the per-call response channel.
- `internal/bridgesdk/peer.go:170` is a non-blocking serve-loop cancellation check with `default`.
- `internal/bridgesdk/webhook.go:125` is a non-blocking semaphore acquire with `default`.
- `internal/bridgesdk/webhook.go:138` is a non-blocking semaphore release with `default`.
- No blocking production `select` was found without either `ctx.Done()` handling or explicit non-blocking semantics.

### Security — Threat Model

- Trust boundaries:
  - The daemon/runtime transport feeds JSON-RPC frames into `Peer` and `Runtime`.
  - Remote webhook senders reach `NewWebhookHandler` over HTTP and control the raw request body, method, headers, and request metadata seen by the guard pipeline.
  - Provider runtimes call back into the daemon through `HostAPIClient`, but those calls originate inside trusted provider code.
- Attacker capabilities:
  - A remote webhook sender can control HTTP method, headers, content type, body size/content, and any value a caller-derived `RequestKey` extracts from the request.
  - A malicious or buggy upstream daemon/runtime peer could send malformed JSON-RPC frames or invalid initialize/delivery/shutdown payloads over stdio.
  - An attacker can attempt request-flooding and high-cardinality webhook keys to stress in-memory limiter state.
- In-scope assets:
  - Provider runtime availability and memory safety.
  - Integrity of managed-instance session state and Host API request/response correlation.
  - Webhook ingress correctness, rate limiting, and bounded concurrency.
- Out-of-scope:
  - Authentication and authorization decisions in higher daemon layers before bridge-runtime requests reach this package.
  - Provider-specific webhook signature semantics implemented by caller-supplied `VerifySignature`.
  - Compromise of already-trusted daemon or provider process code.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/bridgesdk/webhook.go:173-220` | Remote HTTP webhook request body, method, headers, and `RemoteAddr` | Allowed method/content-type checks, `MaxBodyBytes`, optional signature verification, rate limiting, and in-flight limiting | Provider-supplied `WebhookHandler` receives `WebhookRequest` | LOW — ingress checks fail closed before provider mapping and no direct code-execution or filesystem sink exists here. |
| `internal/bridgesdk/webhook.go:79-123`, `186-193` | Caller-derived request key or default remote host used for rate limiting | Key trimming plus per-window stale-key eviction after the fix | `FixedWindowRateLimiter.counts` map | MEDIUM (fixed) — before the fix, high-cardinality request keys could accumulate stale counters indefinitely across windows. |
| `internal/bridgesdk/peer.go:183-204` | Daemon/runtime JSON-RPC frames over stdio | JSON decode, JSON-RPC version check, method lookup, per-call response correlation | Registered RPC handlers or pending response channels | REJECTED — this boundary is daemon-controlled in the threat model, and malformed frames already fail closed with decode/version errors. |
| `internal/bridgesdk/runtime.go:243-285` | Daemon initialize payload | `decodeParams`, `InitializeRequest.Validate`, and explicit `request.Runtime.Bridge != nil` guard | `Session` and `InstanceCache` state | REJECTED — trusted daemon input is validated before any state mutation. |
| `internal/bridgesdk/runtime.go:294-352` | Daemon delivery / shutdown payloads | `decodeParams`, `DeliveryRequest.Validate`, optional shutdown decode | Provider `Deliver`/`Shutdown` handlers | REJECTED — trusted daemon input is validated before reaching provider callbacks. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | refactoring-analysis | medium | `internal/bridgesdk/peer.go:101` | `Peer.Call` encoded params through a panic-based helper, so unmarshalable inputs crashed the runtime and left stale pending-call state behind. | fixed |
| 02 | extreme-software-optimization | medium | `internal/bridgesdk/batching.go:216` | `InboundBatchKey` built its routing key with `fmt.Sprintf`, costing 176 B/op and 8 allocs/op on the inbound batching hot path. | fixed |
| 03 | extreme-software-optimization | medium | `internal/bridgesdk/cache.go:65` | `InstanceCache.Snapshot` deep-cloned already-cloned managed runtime state, doubling snapshot allocation cost. | fixed |
| 04 | security-review | medium | `internal/bridgesdk/webhook.go:79` | `FixedWindowRateLimiter` retained expired request-key counters indefinitely, allowing attacker-driven high-cardinality key growth across windows. | fixed |
| 05 | refactoring-analysis | medium | `internal/bridgesdk/runtime.go:1` | Runtime bootstrap, session lifecycle, request decoding, and handler dispatch remain concentrated in a 400-line file. | deferred |
| 06 | refactoring-analysis | medium | `internal/bridgesdk/errors.go:1` | Error taxonomy, recovery mapping, and retry policy remain packed into one 378-line mixed-responsibility unit. | deferred |

## Per-Skill Notes

### refactoring-analysis

- Large-file pressure remains concentrated in `runtime.go`, `peer.go`, and `errors.go`.
- The fixed `Peer.Call` issue was structurally rooted in `mustRawJSON`, which converted ordinary bad input into a process panic. The call path now marshals params explicitly and returns an error before registering pending response state.
- `go test -cover ./internal/bridgesdk/...` now reports `79.0%` package coverage, up from `78.5%`.
- I deferred file-splitting work in `runtime.go` and `errors.go` because those are structural refactors without stronger correctness or benchmark pressure than the scoped fixes above.

### extreme-software-optimization

- Added `internal/bridgesdk/perf_bench_test.go` so every selected hot-path candidate has a co-located benchmark.
- Replaced `InboundBatchKey`'s `fmt.Sprintf` path with a pre-sized `strings.Builder`, improving the benchmark from `207.12 ns/op, 176 B/op` to `74.97 ns/op, 64 B/op`.
- Removed the redundant deep clone at the tail of `InstanceCache.Snapshot`, improving the benchmark from `3564.80 ns/op, 12016 B/op` to `1437.00 ns/op, 5488 B/op`.
- `BenchmarkPeerCallRoundTrip` stayed effectively flat because the correctness fix only changed error handling for bad params.
- `BenchmarkFixedWindowRateLimiterAllow` stayed allocation-free and only regressed slightly in steady-state nanoseconds because the security hardening now performs a once-per-window stale-key sweep check.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No production deadlock or goroutine-leak finding was confirmed after auditing the package-owned goroutine, response channels, mutexes, and selects.
- `Peer.Serve` remains the only explicit production goroutine site, and it waits for in-flight request handlers before returning.
- Every blocking production `select` either listens for `ctx.Done()` or is intentionally non-blocking via `default`.

### security-review

- No high-confidence remote code execution, injection, path traversal, or secret-exposure vulnerability was identified within this package's threat model.
- The one medium-confidence issue was the stale-key retention bug in `FixedWindowRateLimiter`, where attacker-influenced request keys could accumulate across expired windows. The fix now evicts expired counters before each new window sweep.
- The JSON-RPC/runtime payload surfaces remain trusted-daemon boundaries and already fail closed on invalid payloads.

## Deferred Items (carry forward)

- **05** — Split `internal/bridgesdk/runtime.go` when a follow-up can absorb the structural churn around runtime/session bootstrap, request decoding, and handler dispatch.
- **06** — Split `internal/bridgesdk/errors.go` into smaller classification/recovery/retry units when a follow-up pass targets structural maintainability instead of scoped bug/perf fixes.
- **OPT-03** — Revisit `Peer` transport round-trip costs only if end-to-end profiling shows JSON-RPC framing is material; this pass did not justify deeper transport reshaping.

## `make verify`

Command: `make verify`

Supplemental check: `go vet ./internal/bridgesdk/...`

Exit code: `0`

Excerpt from the clean pass:

```text
0 issues.
✓  internal/bridgesdk (1.064s)
✓  internal/extension (7.588s)
✓  internal/daemon (8.365s)

DONE 4430 tests in 10.006s
OK: all package boundaries respected
```
