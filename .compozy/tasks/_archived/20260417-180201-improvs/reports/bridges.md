# Improvements Report — internal/bridges

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 4 benchmarks in `internal/bridges/perf_bench_test.go`, before/after numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 $(rg --files internal/bridges -g '*.go' -g '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 18 | `(BridgeInstance).Validate` | `internal/bridges/types.go:338` |
| 18 | `(*Service).UpdateInstance` | `internal/bridges/registry.go:252` |
| 18 | `(*Broker).enqueueEventLocked` | `internal/bridges/delivery_broker.go:785` |
| 15 | `(DeliveryEvent).validateTypedFields` | `internal/bridges/types.go:1081` |
| 14 | `validateBridgeInstanceDesiredFields` | `internal/bridges/resource.go:218` |
| 14 | `sameProjectedBridgeInstance` | `internal/bridges/resource_projection.go:235` |
| 14 | `canTransitionInstanceState` | `internal/bridges/lifecycle.go:59` |
| 14 | `(DeliveryEvent).Validate` | `internal/bridges/types.go:700` |
| 14 | `(*Service).UpdateInstanceState` | `internal/bridges/registry.go:324` |
| 14 | `(*Broker).ProjectEvent` | `internal/bridges/delivery_broker.go:376` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/bridges/delivery_broker.go` | 1159 | Route-worker lifecycle, queueing, retries, projection, snapshotting, cleanup, and delivery metrics live in one large stateful unit. |
| `internal/bridges/types.go` | 1119 | Domain enums, validation, normalization, inbound envelopes, outbound events, secret bindings, and dedup models are packed into one file. |
| `internal/bridges/registry.go` | 661 | CRUD, lifecycle transitions, route canonicalization, route persistence, and delivery-target resolution are bundled together. |
| `internal/bridges/managed_sync.go` | 540 | Resource-backed sync and direct-store sync implementations plus equality helpers share one file. |
| `internal/bridges/resource.go` | 491 | Resource codec validation, provider metadata checks, delivery-default normalization, and projection conversion are co-located. |
| `internal/bridges/delivery_types.go` | 488 | Request/snapshot/registration DTOs are mixed with validation, normalization, and clone helpers. |
| `internal/bridges/resource_projection.go` | 319 | Projection build/apply logic, semantic JSON equality, cloning, and sorting live in one file. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/bridges` notable production findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/bridges/delivery_broker.go:595-606` | `internal/bridges/delivery_broker.go:607-618` | Near-identical request-build branches for queued `start` and `delta` events. |
| `internal/bridges/delivery_broker.go:607-618` | `internal/bridges/delivery_broker.go:619-630` | Near-identical `delta` and `terminal` request-build branches in `prepareRequest`. |
| `internal/bridges/delivery_types.go:440-447` | `internal/bridges/types.go:823-830` | Trim/normalize helpers for projection and binding payloads repeat the same field-cleanup pattern. |
| `internal/bridges/registry.go:42-59` | `internal/bridges/types.go:313-330` | Create-request fields largely mirror `BridgeInstance` persistence fields. |
| `internal/bridges/managed_sync.go:491-503` | `internal/bridges/resource_projection.go:238-250` | Managed-sync and resource-projection equality checks duplicate bridge field comparisons. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `ProjectEvent` / turn-index lookup path | `internal/bridges/delivery_broker.go:376`, `394`, `1125` | Every projected session output event resolves the `(session_id, turn_id)` key before dedup and queueing. | `BenchmarkBrokerProjectEventTurnLookup` |
| `(*Broker).enqueueEventLocked` | `internal/bridges/delivery_broker.go:785` | Core ordered-delivery queue path for every projected outbound event. | `BenchmarkBrokerEnqueueEventLockedDelta` |
| `(*Broker).prepareRequest` | `internal/bridges/delivery_broker.go:582` | Route worker assembles negotiated requests on the IO loop before every transport call. | `BenchmarkBrokerPrepareRequestDelta` |
| `(*Broker).DeliveryMetrics` | `internal/bridges/delivery_broker.go:183` | Health and observability surfaces aggregate backlog and counters under the broker lock. | `BenchmarkBrokerDeliveryMetricsSnapshot` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/bridges/...` before any package fix, followed by the same command after the scoped changes:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkBrokerProjectEventTurnLookup` | 26.17 | 24 | 9.78 | 0 | fixed-with-benchmark |
| `BenchmarkBrokerEnqueueEventLockedDelta` | 145.78 | 320 | 145.36 | 320 | not-hot-confirmed-by-benchmark — queue path stayed flat after the turn-index fix, so no separate queue-path change landed. |
| `BenchmarkBrokerPrepareRequestDelta` | 210.52 | 512 | 210.76 | 512 | not-hot-confirmed-by-benchmark — request assembly stayed flat and the remaining duplication is deferred as refactoring work. |
| `BenchmarkBrokerDeliveryMetricsSnapshot` | 6024.40 | 11608 | 6079.40 | 11608 | not-hot-confirmed-by-benchmark — snapshot cost stayed effectively unchanged and did not justify a safe micro-optimization in this pass. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/bridges/delivery_broker.go:518` | `Broker.ensureRouteLocked` | `Broker.Close()` cancels `lifecycleCtx`, `runRouteWorker` exits on `ctx.Done()`, and `WaitGroup` joins. | One worker per routing-key hash; no fire-and-forget goroutines outside the broker. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/bridges/delivery_broker.go:513` (`wakeCh`) | 1 | `routeWorker` / `Broker` | none explicit; worker exits when broker lifecycle context is canceled | `runRouteWorker` | Buffered wake-up channel used only for route-worker nudges; not used as a work payload queue. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/bridges/delivery_broker.go:90` | write-heavy | Transport pointer, active deliveries, turn/session indices, route queues, and delivery metrics | Single broker lock serializes ordered queueing and worker bookkeeping. |

### Concurrency — Select Audit

- `internal/bridges/delivery_broker.go:528` waits on `wakeCh` or `lifecycleCtx.Done()`.
- `internal/bridges/delivery_broker.go:539` waits on retry timer or `lifecycleCtx.Done()`.
- `internal/bridges/delivery_broker.go:543` is a timer-drain cleanup select with `default`, so `ctx.Done()` is not required.
- `internal/bridges/delivery_broker.go:1115` is a non-blocking wake-up send with `default`.
- No blocking production `select` in `internal/bridges/` was found without either context cancellation or explicit non-blocking semantics.

### Security — Threat Model

- Trust boundaries:
  - Higher daemon layers (`internal/api`, `internal/extension`, `internal/daemon`, `internal/store`) pass bridge instance, route, secret binding, and delivery-target requests into this package.
  - Bridge adapters and extension runtimes can originate normalized inbound envelope data that is validated by `internal/bridges` types before persistence or routing.
  - Resource-managed bridge definitions arrive through daemon-owned resource reconciliation.
- Attacker capabilities:
  - A local API caller can submit bridge instance create/update payloads, routing identities, and delivery-target overrides.
  - A remote sender, through a bridge adapter, can influence `InboundMessageEnvelope` fields such as ids, sender metadata, content, attachments, provider metadata, and idempotency keys.
  - An operator-managed resource definition can carry malformed bridge resource specs, but that source is configuration-controlled rather than public-user-controlled.
- In-scope assets:
  - Bridge instance configuration integrity, including scope/workspace binding and delivery defaults.
  - Canonical routing identities and persisted route/session bindings.
  - Secret-binding references and ingest dedup state.
  - Outbound delivery ordering and resumable broker state.
- Out-of-scope:
  - Authentication/authorization in higher API layers before requests reach this package.
  - Secret backend enforcement outside the stored `VaultRef`/binding metadata.
  - Malicious code already executing inside a trusted daemon or extension process.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/bridges/registry.go:200`, `584` | Local create request for bridge instance metadata, provider config, and delivery defaults | `CreateInstanceRequest.toInstance()` trims fields, normalizes JSON, assigns defaults, then `BridgeInstance.Validate()` enforces lifecycle/scope invariants | `RegistryStore.InsertBridgeInstance` via `Service.CreateInstance()` | LOW — local caller-controlled config is validated before persistence; authz belongs to higher layers. |
| `internal/bridges/registry.go:83`, `252` | Local update request for mutable bridge fields | `UpdateInstanceRequest.Validate()` checks optional field shapes; `Service.UpdateInstance()` re-normalizes JSON and re-validates the full instance before persist | `RegistryStore.UpdateBridgeInstance` | LOW — invalid fields fail closed before storage. |
| `internal/bridges/target.go:57`, `78` | Local delivery-target override request (`peer_id`, `thread_id`, `group_id`, `mode`) | `ResolveDeliveryTargetRequest.Validate()` plus `BuildDeliveryTarget()` merge request overrides with validated instance defaults and re-run `DeliveryTarget.Validate()` | Returned `DeliveryTarget` consumed by higher delivery layers | LOW — the package enforces target invariants before the value leaves the domain layer. |
| `internal/bridges/types.go:583` | Adapter-provided inbound envelope and provider metadata from remote users/platforms | `InboundMessageEnvelope.Validate()` enforces scope/workspace binding, event-family-specific payload rules, and JSON validity for provider metadata | Higher ingest, dedup, and routing layers that consume the validated envelope | LOW — this package validates structure and leaves policy decisions to callers; no direct execution or path/file sink exists here. |
| `internal/bridges/resource.go:105` | Operator-managed resource spec for bridge instances | `validateBridgeInstanceResourceSpec()` binds scope, validates desired fields, and checks provider metadata against installed manifests | Canonical `bridge.instance` resources and projected runtime bridge state | REJECTED — operator-controlled config is out of the external attacker model for this package. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | refactoring-analysis | medium | `internal/bridges/resource.go:376` | Delivery-default normalization diverged between resource-managed specs and registry/runtime validation, rejecting provider-specific string defaults that the runtime path and existing tests already accept. | fixed |
| 02 | extreme-software-optimization | medium | `internal/bridges/delivery_broker.go:104` | Broker turn lookup keyed `(session_id, turn_id)` through concatenated strings, allocating on every register/project/remove path in the ordered-delivery hot path. | fixed |
| 03 | refactoring-analysis | medium | `internal/bridges/delivery_broker.go:595` | `prepareRequest` still carries three near-identical request-build branches for start/delta/terminal queue items. | deferred — consolidating those branches is structural churn without benchmark evidence that it is the right next change. |
| 04 | refactoring-analysis | medium | `internal/bridges/managed_sync.go:491` | Bridge equality checks remain duplicated between managed sync and resource projection. | deferred — deduping spans multiple files and would exceed this package pass’s scoped fix budget. |

## Per-Skill Notes

### refactoring-analysis

- Large-file pressure is still concentrated in `delivery_broker.go`, `types.go`, `registry.go`, and `resource.go`.
- The fixed correctness issue was a genuine validation divergence caused by having multiple delivery-default normalization paths. I unified the registry, runtime, and resource entry points on `NormalizeDeliveryDefaultsJSON`, which keeps provider-specific string fields while still rejecting invalid field types or invalid `mode`.
- `go test -cover ./internal/bridges/...` now reports `80.9%` package coverage after the added regression coverage.
- I left the `prepareRequest` branch duplication and the cross-file equality duplication deferred because both are maintainability problems, but neither carried stronger value than the measurable turn-index hot-path fix in this pass.

### extreme-software-optimization

- Added `internal/bridges/perf_bench_test.go` so every selected hot-path candidate has a benchmark co-located with the package.
- Replaced the broker turn-index string key with a structured `turnIndexKey`, removing lookup-key string construction from `RegisterPromptDelivery`, `ProjectEvent`, and delivery cleanup.
- `BenchmarkBrokerProjectEventTurnLookup` improved from `26.17 ns/op, 24 B/op` to `9.78 ns/op, 0 B/op`.
- `BenchmarkBrokerEnqueueEventLockedDelta`, `BenchmarkBrokerPrepareRequestDelta`, and `BenchmarkBrokerDeliveryMetricsSnapshot` all stayed effectively flat, so I did not land speculative micro-optimizations there.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No production deadlock or goroutine-leak finding was confirmed after auditing the package-owned worker, wake channel, and broker lock.
- `runRouteWorker` exits on broker lifecycle cancellation and is joined through the broker `WaitGroup`.
- The buffered `wakeCh` only nudges workers, and every blocking wait includes either `ctx.Done()` or explicit non-blocking semantics.

### security-review

- No high-confidence vulnerabilities identified after the threat model and per-surface inventory.
- The highest-risk inputs are local create/update payloads and adapter-supplied inbound envelopes; both are validated before persistence or domain handoff.
- The fixed delivery-default divergence was correctness hardening, not a direct exploit path, because the affected inputs remain local configuration rather than execution-bearing data.

## Deferred Items (carry forward)

- **03** — Consolidate `prepareRequest` branch duplication in `internal/bridges/delivery_broker.go` once a follow-up task can absorb the structural churn and re-benchmark the full delivery loop.
- **04** — Collapse duplicate bridge-instance equality logic across `internal/bridges/managed_sync.go` and `internal/bridges/resource_projection.go` when both reconciliation paths can be refactored together.
- **OPT-02** — Revisit `DeliveryMetrics` snapshot cloning only if future end-to-end profiling shows observability reads on the broker lock are material.

## `make verify`

Command: `make verify`

Supplemental check: `go vet ./internal/bridges/...`

Exit code: `0`

Excerpt from the clean pass:

```text
0 issues.
✓  internal/bridges (1.087s)
✓  internal/store/globaldb (7.817s)
✓  internal/extension (8.132s)
✓  internal/daemon (8.411s)
✓  internal/cli (8.524s)

DONE 4428 tests in 10.693s
OK: all package boundaries respected
```
