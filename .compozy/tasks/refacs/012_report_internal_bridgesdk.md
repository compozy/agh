# Iteration 012 Refactoring Report: `internal/bridgesdk`

## Scope

- Package: `github.com/pedronauck/agh/internal/bridgesdk`
- Iteration: 012
- Date: 2026-05-06
- Skills applied: `refactoring-analysis`, `extreme-software-optimization`, `systematic-debugging`, `no-workarounds`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`
- Subagents:
  - Refactoring explorer: read-only audit of SDK API shape, JSON-RPC runtime/peer, Host API client, inbound batching, webhook helpers, retry/error helpers, and provider-facing testability.
  - Performance explorer: read-only benchmark/profile audit of JSON-RPC round trips, instance cache snapshots, inbound batch keys, rate limiting, dedup, and runtime param decoding.

## Baseline

- `rtk go test ./internal/bridgesdk -count=1`: `64 passed` before this iteration's fixes.
- `rtk golangci-lint run ./internal/bridgesdk`: no issues.
- `rtk go test -tags integration ./internal/bridgesdk -count=1`: `67 passed`.
- `rtk proxy go test ./internal/bridgesdk -cover -count=1`: `80.6%` statement coverage.
- Initial benchmark baseline:
  - `BenchmarkInboundBatchKey`: about `73-75 ns/op`, `64 B/op`, `1 alloc/op`.
  - `BenchmarkInstanceCacheSnapshot`: about `1440-1464 ns/op`, `5488 B/op`, `14 allocs/op`.
  - `BenchmarkFixedWindowRateLimiterAllow`: about `49.9-50.5 ns/op`, `0 B/op`, `0 allocs/op`.
  - `BenchmarkPeerCallRoundTrip`: about `9357-9510 ns/op`, `2834-2837 B/op`, `61 allocs/op`.

## Findings

### Implemented

1. JSON-RPC response handling lost numeric precision and re-allocated response payloads.
   - Root cause: `rpcEnvelope.Result` decoded into `any`; `handleResponse` then marshaled that `any` back into JSON. Large JSON numbers were rounded through `float64` before callers decoded typed results.
   - Risk: provider Host API calls could corrupt large integer fields and every response paid an avoidable marshal allocation tax.
   - Fix: changed `rpcEnvelope.Result` and `rpcResult.result` to `json.RawMessage`, passed response bytes through directly, and made `sendResult` marshal results explicitly before writing frames.
   - Behavior proof: added `TestPeerResponseRefacs/Should preserve large integer response precision`.
   - Performance proof: `BenchmarkPeerCallRoundTrip` improved from roughly `9357-9510 ns/op`, `~2835 B/op`, `61 allocs/op` to `8569-9088 ns/op`, `~2411 B/op`, `54 allocs/op` after this fix.

2. JSON-RPC frame writing performed an unnecessary second encoding pass.
   - Root cause: `writeFrame` marshaled the frame into bytes and then passed those bytes to `json.Encoder.Encode(json.RawMessage(payload))`.
   - Risk: extra encoder work on every Host API/runtime request/response frame.
   - Fix: wrote the marshaled frame bytes plus newline directly under the existing write lock, preserving one JSON object per line and the existing `json.Marshal` HTML-escaping behavior.
   - Behavior proof: added `TestPeerResponseRefacs/Should write one escaped JSON frame per line`.
   - Performance proof: final `BenchmarkPeerCallRoundTrip` reports about `8763-8972 ns/op`, `2361-2364 B/op`, `54 allocs/op`.

3. Host API calls accepted invalid contexts and duplicated method strings.
   - Root cause: `HostAPIClient.Call` forwarded nil or already-canceled contexts to the transport, and typed methods hardcoded Host API method names that already exist as contract constants.
   - Risk: transports could observe nil/canceled contexts unexpectedly, and method-name drift could break provider calls.
   - Fix: added nil/canceled context guards and replaced bridge Host API method literals with `internal/extension/contract` constants.
   - Behavior proof: added `TestHostAPIClientRefacs/Should reject nil context before invoking transport` and `Should reject canceled context before invoking transport`.

4. `InboundBatcher` accounted timer callbacks after callbacks started.
   - Root cause: `flushKey` called `WaitGroup.Add(1)` inside the timer callback. `Close` could call `Wait` while a timer callback was pending but not yet accounted, and stale timer callbacks could flush a newer batch after a failed `Stop`.
   - Risk: `Close` could return before timer-owned dispatch work settled; a stale timer could dispatch the wrong batch generation.
   - Fix: introduced per-timer generation tokens, accounted each callback before `time.AfterFunc`, balanced stopped timers immediately, and made callbacks flush only when their token still owns the pending batch.
   - Validation proof: package race test passed with `rtk env CGO_ENABLED=1 go test -race ./internal/bridgesdk -count=1`.

5. Batched inbound envelopes did not fully snapshot mutable payloads.
   - Root cause: `cloneInboundEnvelope` copied attachments, metadata, command/action/reaction, but missed the `Conversation` pointer.
   - Risk: providers could mutate an envelope after `Enqueue` and change the delayed dispatch's explicit AGH Network conversation mapping.
   - Fix: cloned `NetworkConversationRef` when snapshotting inbound envelopes.
   - Behavior proof: added `TestInboundBatcherRefacs/Should isolate explicit network conversation refs before delayed dispatch`.

6. Inbound batch keys were pipe-delimited and did not mirror envelope defaulting.
   - Root cause: `InboundBatchKey` joined identity parts with raw `|` and used the raw `EventFamily` string. The bridge domain normalizes an empty plain-message family to `message`.
   - Risk: identity parts containing `|` could collide across distinct routing identities; an envelope that validates as a message could get a different batch key before and after normalization.
   - Fix: changed the key to length-prefixed parts and canonicalized the default message family.
   - Behavior proof: added `TestInboundBatcherRefacs/Should avoid delimiter collisions in routing identity keys` and `Should canonicalize empty message family like envelope validation`.
   - Performance note: this intentionally trades a small hot-path cost for correctness. Final `BenchmarkInboundBatchKey` is about `129.8-135.0 ns/op`, `80 B/op`, `1 alloc/op` versus the unsafe baseline of about `73-75 ns/op`, `64 B/op`, `1 alloc/op`.

7. Runtime initialize/shutdown lifecycle held or consumed state too aggressively.
   - Root cause: `handleInitialize` held `Runtime.mu` while invoking provider callback code; `handleShutdown` used `sync.Once`, so a failed first shutdown consumed the only callback execution and future shutdowns acknowledged without retrying.
   - Risk: initialize callbacks that call runtime accessors could deadlock; transient shutdown failures could never be retried.
   - Fix: reserve initialize state under lock, run provider initialize outside the lock, commit/rollback deliberately, and replace `sync.Once` shutdown with explicit idle/running/succeeded state.
   - Behavior proof: added `TestRuntimeRefacs/Should not hold runtime lock while initialize callback runs` and `Should retry shutdown handler after a failed first attempt`.

8. Runtime param decoding copied raw JSON just to detect empty/null payloads.
   - Root cause: `decodeParams` converted `json.RawMessage` to `string` for whitespace/null checks.
   - Risk: repeated unnecessary allocations on runtime JSON-RPC request handling.
   - Fix: switched to `bytes.TrimSpace` and `bytes.Equal`, and made shutdown's optional params check use the same byte-based pattern.
   - Behavior proof: existing `TestDecodeParamsHandlesNullAndInvalidJSON` still passes, with added `TestRuntimeRefacs/Should decode whitespace null params as empty object`.

9. Instance cache snapshot and secret lookup did extra allocation work.
   - Root cause: `Snapshot` appended managed instances into a nil slice, and `BoundSecretValue` called `Get`, which deep-cloned a whole managed instance just to return an immutable string.
   - Risk: avoidable allocations in provider sync/status paths.
   - Fix: preallocated `Snapshot.ManagedInstances` and changed `BoundSecretValue` to read under `RLock` directly.
   - Performance proof: `BenchmarkInstanceCacheSnapshot` improved from about `1440-1464 ns/op`, `5488 B/op`, `14 allocs/op` to about `948.8-971.2 ns/op`, `3280 B/op`, `11 allocs/op`.

10. Webhook body/error helpers hid errors and emitted invalid retry guidance.
    - Root cause: `readBodyWithLimit` discarded `Close` errors, and `writeWebhookError` truncated positive subsecond `RetryAfter` durations to header value `0`.
    - Risk: production code violated repo error-handling rules; retrying clients could receive `Retry-After: 0` for real positive delays.
    - Fix: join body read and close errors, and round positive retry durations up to whole seconds.
    - Behavior proof: added `TestWebhookRefacs/Should report request body close errors` and `Should round positive subsecond retry after to one second`.

11. Exported retry helper had weak guardrails.
    - Root cause: `RetryDo` accepted nil context and nil operation function, and ended with an unreachable nil error return.
    - Risk: callers could panic or receive an impossible success if future loop logic changed.
    - Fix: reject nil context/function before invoking the operation and replace the unreachable nil return with a defensive error.
    - Behavior proof: added `TestRetryDoRefacs`.

### Deferred

1. Provider-wide bound secret hardening.
   - Finding: `BoundSecretValue(instanceID, bindingName) (string, bool)` is easy for providers to misuse by ignoring the boolean, and several provider implementations currently do that.
   - Reason deferred: the correct fix should add a required-secret helper or typed resolver and migrate bridge providers in one coordinated pass. That crosses the current package boundary.

2. Dedup cache sweep throttling.
   - Finding: `DedupCache.Mark` sweeps the full map on every mark.
   - Reason deferred: no high-cardinality benchmark currently proves this is a bottleneck. A safe change needs deterministic eviction semantics and focused benchmarks.

3. `rpcIDKey` allocation reduction.
   - Finding: response/request ID normalization allocates while unquoting string IDs.
   - Reason deferred: changing the key shape needs a contract decision on whether response ID `1` should match pending request ID `"1"`.

4. Fixed-window limiter high-cardinality sweep.
   - Finding: the limiter can sweep many keys once per window.
   - Reason deferred: the current single-key benchmark is already about `50 ns/op`, `0 B/op`, `0 allocs/op`. A new high-cardinality workload should be added before changing behavior.

5. Provider test reflection cleanup.
   - Finding: some provider tests construct `bridgesdk.Session` by mutating unexported fields through reflection.
   - Reason deferred: a proper fix likely belongs in a provider/test harness pass, possibly with a focused `bridgesdktest` helper package. Adding a broad public `Session` constructor would be an API decision, not a local refactor.

6. Provider runtime setup deduplication.
   - Finding: bridge providers repeat similar `NewRuntime(RuntimeConfig{...})` setup blocks.
   - Reason deferred: the repeated code is shallow and provider-specific. A framework abstraction would be premature unless it removes initialization, health, shutdown, batching, and marker duplication together.

## Files Changed

- `internal/bridgesdk/batching.go`
- `internal/bridgesdk/cache.go`
- `internal/bridgesdk/errors.go`
- `internal/bridgesdk/hostapi.go`
- `internal/bridgesdk/peer.go`
- `internal/bridgesdk/runtime.go`
- `internal/bridgesdk/webhook.go`
- `internal/bridgesdk/batching_refac_test.go`
- `internal/bridgesdk/errors_refac_test.go`
- `internal/bridgesdk/hostapi_refac_test.go`
- `internal/bridgesdk/peer_refac_test.go`
- `internal/bridgesdk/runtime_refac_test.go`
- `internal/bridgesdk/webhook_refac_test.go`

## Validation

```bash
rtk go test ./internal/bridgesdk -run '^TestPeerResponseRefacs$' -count=1
rtk go test ./internal/bridgesdk -run '^TestPeer' -count=1
rtk proxy go test ./internal/bridgesdk -run '^$' -bench 'BenchmarkPeerCallRoundTrip' -benchmem -count=5
rtk go test ./internal/bridgesdk -run '^TestHostAPIClientRefacs$' -count=1
rtk go test ./internal/bridgesdk -run '^Test(HostAPIClientRefacs|PeerResponseRefacs)$' -count=1
rtk go test ./internal/bridgesdk -run '^Test(HostAPI|Peer)' -count=1
rtk go test ./internal/bridgesdk -run '^Test(InboundBatcherRefacs|RuntimeRefacs|WebhookRefacs|RetryDoRefacs|PeerResponseRefacs|HostAPIClientRefacs)$' -count=1
rtk go test ./internal/bridgesdk -run '^Test(RuntimeServeInitializeDeliverHealthShutdownAndSync|InboundBatcherCoalescesShortBurstAndPreservesOrdering|WebhookHandlerWritesHTTPErrorFromProviderMapping|InstanceCacheSnapshotAndListReturnClones|RetryDo)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/batching_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/runtime_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/webhook_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/errors_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/peer_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/hostapi_refac_test.go
rtk go test ./internal/bridgesdk -count=1
rtk go test -tags integration ./internal/bridgesdk -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bridgesdk -count=1
rtk golangci-lint run ./internal/bridgesdk
rtk proxy go test ./internal/bridgesdk -cover -count=1
rtk proxy go test ./internal/bridgesdk -run '^$' -bench 'Benchmark(InboundBatchKey|InstanceCacheSnapshot|FixedWindowRateLimiterAllow|PeerCallRoundTrip)$' -benchmem -count=5
rtk go test ./internal/bridgesdk ./internal/extension ./internal/extensiontest -count=1
rtk go test ./extensions/bridges/discord ./extensions/bridges/gchat ./extensions/bridges/github ./extensions/bridges/linear ./extensions/bridges/slack ./extensions/bridges/teams ./extensions/bridges/telegram ./extensions/bridges/whatsapp ./sdk/examples/telegram-reference -count=1
rtk golangci-lint run ./internal/bridgesdk ./extensions/bridges/discord ./extensions/bridges/gchat ./extensions/bridges/github ./extensions/bridges/linear ./extensions/bridges/slack ./extensions/bridges/teams ./extensions/bridges/telegram ./extensions/bridges/whatsapp ./sdk/examples/telegram-reference
```

Observed results:

- Focused refactor tests: `20 passed`.
- Existing focused regression set: `14 passed`.
- Package tests: `84 passed`.
- Integration-tag package tests: `87 passed`.
- Race package tests: passed.
- Package lint: no issues.
- Package coverage after edits: `81.2%` statements.
- Direct internal dependent set: `626 passed in 3 packages`.
- Provider/example dependent set: `191 passed in 9 packages`.
- Provider/example lint set: no issues.
- Final benchmarks:
  - `BenchmarkInboundBatchKey`: about `129.8-135.0 ns/op`, `80 B/op`, `1 alloc/op`.
  - `BenchmarkInstanceCacheSnapshot`: about `948.8-971.2 ns/op`, `3280 B/op`, `11 allocs/op`.
  - `BenchmarkFixedWindowRateLimiterAllow`: about `49.86-52.23 ns/op`, `0 B/op`, `0 allocs/op`.
  - `BenchmarkPeerCallRoundTrip`: about `8763-8972 ns/op`, `2361-2364 B/op`, `54 allocs/op`.

Full monorepo gate:

```bash
rtk make verify
```

Result: passed.

## Next Package

- `github.com/pedronauck/agh/internal/bundles`
