# Improvements Report — internal/network

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/network/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/network | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 24 | `TestEnumValidationAndBodyKindHelpers` | `internal/network/helpers_test.go:9` |
| 23 | `TestManagerValidationAndNilGuards` | `internal/network/manager_test.go:784` |
| 23 | `TestManagerJoinSendStatusAndLeave` | `internal/network/manager_test.go:125` |
| 22 | `TestAuditWriterRecordTaskIngress` | `internal/network/audit_test.go:265` |
| 21 | `TestRouterDirectedRecipeOpensInteractionForReceiptAndTrace` | `internal/network/router_test.go:697` |
| 20 | `TestRouterRejectsDuplicateBeforeReprocessingLifecycleState` | `internal/network/router_test.go:152` |
| 20 | `TestManagerStatusTracksWorkflowMetricsAndStructuredLogs` | `internal/network/manager_test.go:420` |
| 19 | `TestTransportLifecycleAndMethodGuards` | `internal/network/transport_test.go:58` |
| 19 | `TestManagerListsPeersAndAuditsInboundRemoteDeliveries` | `internal/network/manager_test.go:576` |
| 19 | `(*Manager).Shutdown` | `internal/network/manager.go:705` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/network/router.go` | 897 | Routing, replay-window tracking, interaction lifecycle, and generated receipt/whois reply handling are concentrated in one file. |
| `internal/network/manager.go` | 1250 | Runtime wiring, session/channel lifecycle, auditing, status, reconnect logic, and ingress policy are combined in one unit. |
| `internal/network/envelope.go` | 310 | Wire enums plus all body schema types share one file. |
| `internal/network/peer.go` | 596 | Local/remote presence tracking, channel summaries, cloning, and matching logic share one registry unit. |
| `internal/network/validate.go` | 593 | Envelope validation, body decoding, normalization, and protocol rules are bundled together. |
| `internal/network/lifecycle.go` | 356 | Interaction state validation and lifecycle transition rules live in one file. |
| `internal/network/delivery.go` | 974 | Delivery queueing, worker orchestration, message rendering, and reply guidance formatting are tightly coupled. |
| `internal/network/tasks.go` | 460 | Task-ingress authorization, channel validation, audit recording, and actor derivation live together. |
| `internal/network/audit.go` | 362 | File/store sink routing, timeline normalization, and task-ingress audit handling share one writer type. |
| `internal/network/transport.go` | 398 | Embedded NATS startup, publish/subscribe, drain/shutdown, and subject helpers share one unit. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 60 internal/network` found the following duplicates at or above the reporting threshold:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/network/manager.go:1060-1080` | `internal/network/manager.go:1082-1102` | Repeated audit-write/update/log boilerplate in `recordAuditSent` and `recordAuditReceived`. |
| `internal/network/lifecycle_test.go:159-183` | `internal/network/lifecycle_test.go:244-268` | Test-only lifecycle assertion duplication. |
| `internal/network/lifecycle_test.go:33-51` | `internal/network/lifecycle_test.go:120-138` | Test-only setup duplication. |
| `internal/network/delivery_integration_test.go:334-351` | `internal/network/delivery_test.go:725-742` | Test-only malformed-body delivery coverage duplicated across layers. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `formatNetworkMessage` | `internal/network/delivery.go:593` | Runs inside the per-session delivery worker for every accepted inbound message and performs body decode, canonical JSON render, base64 encoding, and large reply-guidance string construction. | `BenchmarkFormatNetworkMessageDirect` |
| `(*PeerRegistry).ListPeers` | `internal/network/peer.go:344` | Backing snapshot for `Manager.ListPeers` and `Manager.Status`, which are consumed by API/UI polling paths; filtered channel queries currently walk the full local and remote registry. | `BenchmarkPeerRegistryListPeersFiltered` |
| `networkLogFields` | `internal/network/manager.go:1178` | Called on every sent/received/rejected log emission and repeatedly decodes extension fields into structured slog attributes. | `BenchmarkNetworkLogFields` |

### Optimization — Benchmark Results

Baseline command before production changes: `go test -run '^$' -bench=. -benchmem -count=5 ./internal/network/...`

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkFormatNetworkMessageDirect` | 22593 | 96367 | 6794 | 11088 | fixed-with-benchmark |
| `BenchmarkPeerRegistryListPeersFiltered` | 22673 | 85560 | 10493 | 8504 | fixed-with-benchmark |
| `BenchmarkNetworkLogFields` | 1174 | 1945 | 1173 | 1945 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/network/router.go:222` | `(*Router).StartHeartbeat` | Stops on `Heartbeat.stop` close or caller `ctx.Done()`; `Heartbeat.Stop()` waits on `done`. | Per-session greet heartbeat loop created by the router. |
| `internal/network/manager.go:570` | `(*Manager).startAuditedHeartbeat` | Stops on `Heartbeat.stop` close or `m.lifecycleCtx.Done()`; `Heartbeat.Stop()` waits on `done`. | Manager-owned audited heartbeat loop per joined session. |
| `internal/network/delivery.go:281` | `(*deliveryCoordinator).trigger` | Worker exits on `lifecycleCtx.Done()`, prompt-active handoff, empty queue, or delivery failure; tracked by `wg`. | One worker goroutine per active session delivery lane. |
| `internal/network/delivery.go:447` | `(*deliveryCoordinator).retryAfterWorkerExit` | Waits on worker `state.done` or `lifecycleCtx.Done()` before optionally retriggering. | Retry helper after prompt/render failures. |
| `internal/network/transport.go:342` | `(*Transport).Shutdown` | Waits for `server.WaitForShutdown()` completion or caller `ctx.Done()`. | Short-lived waiter goroutine around embedded NATS shutdown. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/network/router.go:74` | 0 | `(*Router).StartHeartbeat` | `Heartbeat.Stop` | Heartbeat goroutine select | Stop signal for router heartbeat. |
| `internal/network/router.go:75` | 0 | `(*Router).StartHeartbeat` | Heartbeat goroutine `defer close(done)` | `Heartbeat.Stop`, `Heartbeat.Done` | Completion signal for router heartbeat. |
| `internal/network/manager.go:566` | 0 | `(*Manager).startAuditedHeartbeat` | `Heartbeat.Stop` | Manager heartbeat goroutine select | Stop signal for manager-owned audited heartbeat. |
| `internal/network/manager.go:567` | 0 | `(*Manager).startAuditedHeartbeat` | Manager heartbeat goroutine `defer close(done)` | `Heartbeat.Stop`, tests | Completion signal for manager-owned heartbeat. |
| `internal/network/delivery.go:44` | 0 | `(*deliveryCoordinator).trigger` | Worker goroutine `defer close(done)` | `retryAfterWorkerExit` | Completion signal for one delivery worker. |
| `internal/network/transport.go:85` | 0 | `NewTransport` | NATS `ClosedHandler` via `closeOnce` | `(*Transport).Drain` | Connection-closed notification for drain/shutdown. |
| `internal/network/transport.go:341` | 0 | `(*Transport).Shutdown` | Waiter goroutine `defer close(done)` | Shutdown select | Completion signal for embedded NATS shutdown wait. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/network/router.go:118` | write-heavy (`sync.Mutex`) | Replay-window `seen` map and interaction state map | Guards duplicate detection and lifecycle map mutation. |
| `internal/network/manager.go:95` | mixed (`sync.Mutex`) | Session/channel runtime maps plus connection-state flags | Coordinates join/leave, reconnect, and shutdown state. |
| `internal/network/peer.go:49` | read-heavy (`sync.RWMutex`) | Local/remote presence maps and greet interval clocked expiry | Registry lookups mix reads with expiry-backed mutations. |
| `internal/network/stats.go:29` | write-heavy (`sync.Mutex`) | Message counters and per-kind metrics | Shared runtime stats snapshot state. |
| `internal/network/delivery.go:33` | mixed (`sync.Mutex`) | Per-session queues and in-flight message map | Delivery lane registry and queue ownership. |
| `internal/network/delivery.go:48` | write-heavy (`sync.Mutex`) | One session queue’s ordered items slice | Queue push/pop/prepend/snapshot operations. |
| `internal/network/audit.go:72` | write-heavy (`sync.Mutex`) | File append critical section | Serializes JSONL appends and file sync. |
| `internal/network/transport.go:87` | write-heavy (`sync.Mutex`) | Drain/shutdown one-time flags | Prevents duplicate drain/shutdown work. |

### Concurrency — Select Audit

All long-lived blocking selects are context-aware or explicitly bounded:

| File:Line | Reasoning |
| --- | --- |
| `internal/network/router.go:229` | Heartbeat loop listens on `heartbeat.stop`, caller `ctx.Done()`, and ticker. |
| `internal/network/manager.go:577` | Manager heartbeat loop listens on `heartbeat.stop`, `m.lifecycleCtx.Done()`, and ticker. |
| `internal/network/delivery.go:406` | Prompt-event drain loop listens on `c.lifecycleCtx.Done()` while consuming the event channel. |
| `internal/network/delivery.go:448` | Retry helper waits on worker completion or `c.lifecycleCtx.Done()` before retriggering. |
| `internal/network/transport.go:305` | Drain waits on `closedCh` or caller `ctx.Done()`. |
| `internal/network/transport.go:347` | Shutdown waits on embedded-server completion or caller `ctx.Done()`. |

### Security — Threat Model

- Trust boundaries:
  - `internal/network` sits between local daemon/API/CLI callers and remote peers on the embedded NATS transport.
  - Untrusted remote-peer input enters through `Router.Receive` after NATS delivery and through task-ingress helpers when the daemon chooses to expose them.
  - Local operators configure transport/audit paths through trusted config rather than remote network messages.
- Attacker capabilities:
  - A remote peer can send arbitrary network envelopes and peer-card advertisements within the configured payload size limits.
  - A local API/CLI/extension caller can provide outbound `SendRequest` values and can read status/peer/inbox views exposed by upstream layers.
  - An authenticated remote peer with task ingress enabled can submit task specs/patches through the task-ingress helpers.
- In-scope assets:
  - Correct envelope validation and routing decisions.
  - Safe local delivery formatting of untrusted network content.
  - Integrity of task-ingress channel binding and capability checks.
  - Audit-log correctness without remote-controlled path or sink corruption.
- Out-of-scope:
  - Operator-controlled config values such as NATS port, audit-file path, and payload ceilings.
  - Cryptographic verification of peer identity or proof material beyond what this package currently preserves/opacely forwards.
  - Authorization and HTTP/CLI access control in upstream API or daemon layers.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/network/router.go:289` | Remote peer controls raw NATS payload bytes delivered to `Router.Receive`. | `ParseEnvelope` + `NormalizeEnvelope` enforce protocol/kind/body/freshness validation before routing. | Local delivery queue admission, remote presence refresh, generated receipts/whois replies. | LOW: validated before use; malformed payloads become rejected route results. |
| `internal/network/router.go:251` | Local caller controls `SendRequest` fields, including body/ext IDs and targeting. | `buildEnvelope` trims identifiers, validates channel/peer refs/body/freshness, and checks target presence before publish. | Transport publish to broadcast/direct NATS subjects. | LOW: validated request path; no sink reached without protocol validation. |
| `internal/network/delivery.go:593` | Remote/local envelope body content is rendered into agent-visible delivery text. | Body preview is XML-escaped; canonical body is base64-encoded; wrapper explicitly marks content `trust="untrusted"`. | Prompt message sent to `PromptNetwork`. | LOW: untrusted content is encoded/escaped before prompt inclusion. |
| `internal/network/tasks.go:105` | Authenticated remote peer controls task create/update/cancel/enqueue payloads and requested channel values. | `resolveTaskPeerContext`, `validateRequestedTaskChannel`, and `enforceBoundTaskChannel` restrict peer/capability/channel alignment before task-manager calls. | Task-service create/update/cancel/enqueue entry points plus audit records. | LOW: channel/capability gates are explicit; no direct sink without validation. |
| `internal/network/audit.go:92` | Local caller provides session ID, direction, envelope, and optional reason for audit rows. | `NormalizeAuditEntry` validates normalized fields; file path is operator-configured rather than remote-controlled. | JSONL append path and optional persistent audit/message store. | LOW: sink path is trusted config and payload fields are normalized before persistence. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | refactoring-analysis | medium | `internal/network/manager.go:1060` | Sent/received audit helpers duplicate the same write-update-log flow. | wontfix: two explicit helpers remain clearer than introducing a shared abstraction for a single duplicate pair. |
| 02 | extreme-software-optimization | medium | `internal/network/peer.go:370` | Channel-filtered peer snapshots walked every local and remote entry instead of using the existing per-channel indexes. | fixed: `ListPeers` now uses the channel index fast path and benchmarked lower latency/allocation cost. |
| 03 | extreme-software-optimization | medium | `internal/network/delivery.go:618` | Delivery rendering allocated heavily while rebuilding reply-guidance strings for every inbound message. | fixed: reply-guidance rendering now reuses static fragments and a pre-grown builder with benchmarked wins. |

## Per-Skill Notes

### refactoring-analysis
- Initial scan found one production duplicate block pair in the audit helper methods; no production duplicate beyond that threshold appeared in `router.go`, `delivery.go`, or `peer.go`.
- The duplicate in `recordAuditSent` and `recordAuditReceived` was left as `wontfix`: both helpers remain short, diverge at the event type/log message level, and a shared helper would add indirection without eliminating a broader pattern.

### extreme-software-optimization
- Baseline medians from `go test -run '^$' -bench=. -benchmem -count=5 ./internal/network/...`:
- `BenchmarkFormatNetworkMessageDirect`: `22593 ns/op`, `96367 B/op`, `220 allocs/op`
- `BenchmarkPeerRegistryListPeersFiltered`: `22673 ns/op`, `85560 B/op`, `172 allocs/op`
- `BenchmarkNetworkLogFields`: `1174 ns/op`, `1945 B/op`, `28 allocs/op`
- Post-fix medians from the same command:
- `BenchmarkFormatNetworkMessageDirect`: `6794 ns/op`, `11088 B/op`, `47 allocs/op`
- `BenchmarkPeerRegistryListPeersFiltered`: `10493 ns/op`, `8504 B/op`, `172 allocs/op`
- `BenchmarkNetworkLogFields`: `1173 ns/op`, `1945 B/op`, `28 allocs/op`
- `formatNetworkMessage` and filtered `ListPeers` were fixed because the after numbers show material improvement; `networkLogFields` stayed effectively flat and is therefore treated as not-hot rather than a fix target.

### ubs
- `not-run` due to a missing callable skill runner in this environment.

### deadlock-finder-and-fixer
- Inventory review did not uncover a concrete leak or deadlock path in the delivery retry waiter.
- `retryAfterWorkerExit` is bounded by either the worker `done` channel or `lifecycleCtx.Done()`, and the waiter only retriggers while queue state still indicates pending work.
- No package-local concurrency fix was justified from the evidence gathered here.

### security-review
- Threat-model and surface-inventory review found no HIGH or MEDIUM package-local vulnerability.
- Remote task-ingress capability trust remains a broader design limitation, but inside `internal/network/` the current path still gates by bound channel and declared capability before reaching task-manager sinks, so it does not meet the threshold for a package-local fix finding in this pass.

## Deferred Items (carry forward)

- None.

## `make verify`

Final command: `make verify`

Exit code: `0`

Excerpt from the clean pass:

```text
0 issues.
✓  internal/network (1.246s)
✓  internal/hooks (1.591s)
✓  internal/acp (4.389s)
✓  internal/extension (7.223s)
✓  internal/daemon (7.675s)
✓  internal/cli (8.396s)

DONE 4486 tests in 10.113s
OK: all package boundaries respected
```

Non-blocking environment warnings observed during the same run:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.
