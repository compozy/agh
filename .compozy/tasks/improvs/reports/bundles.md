# Improvements Report — internal/bundles

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 2 benchmarks in `internal/bundles/perf_bench_test.go`, before/after numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo $(rg --files internal/bundles -g '*.go' -g '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 13 | `NewResourceStore` | `internal/bundles/resource_store.go:65` |
| 13 | `(*Service).Activate` | `internal/bundles/service.go:215` |
| 12 | `(*Service).materializeBridge` | `internal/bundles/service.go:834` |
| 11 | `syncOwnedResources` | `internal/bundles/resource_store.go:517` |
| 8 | `(*Service).ListActivations` | `internal/bundles/service.go:287` |
| 8 | `(*Service).Build` | `internal/bundles/resource_projection.go:51` |
| 8 | `(*ResourceStore).ListBundleActivationInventory` | `internal/bundles/resource_store.go:222` |
| 7 | `(*Service).resolveWorkspace` | `internal/bundles/service.go:902` |
| 7 | `(*Service).materializeActivationResources` | `internal/bundles/service.go:781` |
| 7 | `(*Service).UpdateActivation` | `internal/bundles/service.go:353` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/bundles/resource_store.go` | 672 | Resource-backed activation CRUD, inventory reads, owned-resource sync, equality checks, and reconcile triggering all live in one file. |
| `internal/bundles/service.go` | 1238 | Catalog reads, activation CRUD, reconcile orchestration, workspace resolution, materialization, hashing, logging, and clone helpers are concentrated in one monolith. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/bundles` notable production findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/bundles/resource_store.go:300-338` | `internal/bundles/resource_store.go:340-377` | Owned job/trigger sync setup repeats the same owner-to-desired grouping and generic dispatch pattern. |
| `internal/bundles/resource_store.go:340-377` | `internal/bundles/resource_store.go:379-416` | Owned trigger/bridge sync setup repeats the same grouping and generic dispatch shape. |
| `internal/bundles/resource_store.go:418-449` | `internal/bundles/resource_store.go:451-482` | Upsert logic for owned jobs and triggers is structurally mirrored. |
| `internal/bundles/resource_store.go:451-482` | `internal/bundles/resource_store.go:484-515` | Upsert logic for owned triggers and bridges is structurally mirrored. |
| `internal/bundles/service.go:792-804` | `internal/bundles/service.go:805-817` | Inventory append loops for materialized jobs and triggers still mirror each other. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*Service).ListActivations` | `internal/bundles/service.go:287` | Baseline API list flow resolved each activation preview by revisiting bundle catalog data for every activation in the result set. | `BenchmarkServiceListActivationsLargeCatalog` |
| `(*Service).Build` / activation resolution from bundle records | `internal/bundles/resource_projection.go:51`, `143`, `196` | Baseline boot/reconcile projection flow repeatedly searched the same bundle catalog slice while iterating activation records. | `BenchmarkServiceBuildLargeCatalog` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/bundles/...` before any package fix, followed by the same command after the bundle-record lookup change:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkServiceListActivationsLargeCatalog` | 2015702.60 | 10800779.60 | 756464.00 | 1520727.80 | fixed-with-benchmark |
| `BenchmarkServiceBuildLargeCatalog` | 1013961.60 | 1638785.80 | 918897.00 | 1734592.00 | fixed-with-benchmark — runtime improved even though the indexed lookup adds one map allocation per build pass. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `go` statements exist under `internal/bundles/`; package concurrency is mutex-based only. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| none | — | — | — | — | No production channels are declared in `internal/bundles/`. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/bundles/service.go:102` | write-heavy | Serialized activation mutation and reconcile entry points | `opMu` guards `Activate`, `UpdateActivation`, `Deactivate`, and `Reconcile`. |
| `internal/bundles/service.go:103` | read-heavy | Cached `NetworkSettings` snapshot | `settingsMu` protects reads/writes around `applyNetworkSettings` and `NetworkSettings`. |

### Concurrency — Select Audit

No production `select` statements exist in `internal/bundles/`.

### Security — Threat Model

- Trust boundaries:
  - Local daemon API layers call `Service` methods with activation requests and read bundle catalog/activation state through HTTP and UDS routes.
  - The daemon bundle-resource wiring calls `NewResourceStore`, `Build`, and reconcile flows with typed bundle and activation records.
  - Workspace resolution crosses into `internal/workspace` when callers provide workspace-scoped bundle activation references.
- Attacker capabilities:
  - A local caller can control activation request fields: extension name, bundle name, profile name, scope, workspace reference, and default-channel binding flag.
  - A local caller can control activation IDs on update/deactivate/read paths.
  - Operator-managed bundle resource specs can be malformed, but they are configuration-controlled rather than public-user-controlled.
- In-scope assets:
  - Integrity of persisted bundle activation records and owned automation/bridge resources.
  - Correct workspace scoping and workspace registration behavior.
  - Integrity of effective default-channel computation and declared channel inventory.
- Out-of-scope:
  - Authentication and authorization in higher API layers before requests reach this package.
  - Correctness of external extension manifests and daemon-owned resource reconciliation inputs beyond this package’s validation.
  - Malicious code already executing inside a trusted daemon process.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/bundles/service.go:198`, `511`, `902` | Local preview/activate request controls extension, bundle, profile, scope, and workspace reference. | `Scope.Normalize`, `resolveWorkspace`, `Activation.Validate`, bundle/profile lookup, and downstream materialization validations fail closed. | `PreviewActivation` / `Activate` resolve bundle definitions, materialize owned resources, and persist activation state via `Store`. | LOW — caller-controlled fields are validated and normalized before persistence or resource fan-out. |
| `internal/bundles/service.go:353`, `393` | Local update/deactivate/read requests control activation IDs and default-channel binding. | `Store.GetBundleActivation`, `validatePrimaryChannelClaim`, and activation validation reject missing or conflicting state. | Activation record update/delete plus reconcile application. | LOW — invalid or missing activation references fail closed through store lookups and validation. |
| `internal/bundles/resource.go:80`, `109` | Bundle and activation resource specs from daemon-owned resource writes. | Typed resource codecs normalize strings and validate required fields, scopes, and manifest-backed bundle structure. | Canonical `bundle` and `bundle.activation` resource stores. | REJECTED — this is operator/daemon-controlled configuration input, not an external attacker boundary for this package. |
| `internal/bundles/service.go:902-931` | Workspace-scoped activation path can carry a user-provided workspace path or ID. | `aghconfig.ResolvePath` normalizes path-like refs and resolver methods return canonical workspace IDs. | `workspaceResolver.Resolve` / `ResolveOrRegister`. | LOW — the package delegates resolution to the workspace subsystem and does not directly open files or execute path content. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | extreme-software-optimization | medium | `internal/bundles/service.go:287`, `internal/bundles/resource_projection.go:148`, `internal/bundles/resource_projection.go:196` | Large-catalog activation list/build flows repeatedly searched the same bundle-record slice per activation; bundle lookup is now indexed once per pass. | fixed |
| 02 | refactoring-analysis | medium | `internal/bundles/service.go:576`, `internal/bundles/resource_projection.go:143` | Desired-state accumulation was duplicated between runtime reconcile and resource-projection build flows. | fixed |
| 03 | refactoring-analysis | medium | `internal/bundles/service.go:1` | `service.go` remains a 1238-LOC multi-responsibility unit spanning catalog reads, activation CRUD, reconcile orchestration, workspace resolution, and materialization helpers. | deferred |
| 04 | refactoring-analysis | medium | `internal/bundles/resource_store.go:300` | Owned job/trigger/bridge sync setup and upsert helpers remain heavily mirrored across three resource kinds. | deferred |

## Per-Skill Notes

### refactoring-analysis

- The highest-value production duplicate was the desired-state accumulation loop shared between `collectDesiredState` and `collectDesiredStateFromBundleRecords`. `collectDesiredState` now delegates through the shared helper instead of carrying its own copy.
- `service.go` and `resource_store.go` still dominate both file size and structural complexity for this package.
- The remaining meaningful duplication is concentrated in the owned-resource sync/setup helpers and the mirrored inventory append loops in `materializeActivationResources`.

### extreme-software-optimization

- Added `internal/bundles/perf_bench_test.go` so the package now has co-located large-catalog benchmarks for the activation list and build paths.
- Added `newBundleRecordLookup` plus `resolveActivationFromBundleLookup`, letting large-catalog list/build/reconcile flows index bundle records once per pass instead of re-scanning the same slice for every activation.
- `BenchmarkServiceListActivationsLargeCatalog` improved from `2015702.60 ns/op, 10800779.60 B/op` to `756464.00 ns/op, 1520727.80 B/op`.
- `BenchmarkServiceBuildLargeCatalog` improved from `1013961.60 ns/op, 1638785.80 B/op` to `918897.00 ns/op, 1734592.00 B/op`.
- The build path now trades a modest map allocation increase for lower end-to-end runtime on large catalogs; I kept the change because the lookup shift is shared with the much larger list-path win and removes repeated O(n) scans from both flows.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No production goroutine, channel, or select-based deadlock surface exists in this package.
- The only production synchronization primitives are `opMu` and `settingsMu`, with no nested lock ordering inversions observed in the current code paths.

### security-review

- No high-confidence vulnerabilities identified within this package’s threat model.
- The main externally influenced inputs are activation requests and workspace references, both of which are validated or delegated to typed subsystems before use.

## Deferred Items (carry forward)

- **03** — Split `internal/bundles/service.go` when a follow-up task can absorb broader structural churn around catalog reads, CRUD, reconcile, and materialization responsibilities.
- **04** — Consolidate the mirrored owned-resource sync and upsert helpers in `internal/bundles/resource_store.go` once a follow-up is willing to refactor all three resource kinds together.

## `make verify`

pass

Captured output excerpt from the final clean pass of `make verify`:

```text
$ make verify
# github.com/golangci/golangci-lint/v2/cmd/golangci-lint
ld: warning: -bind_at_load is deprecated on macOS
0 issues.
✓  internal/bundles (cached)
✓  internal/daemon (cached)
✓  internal/cli (cached)
✓  internal/extension (cached)
DONE 4431 tests in 0.900s
OK: all package boundaries respected
```
