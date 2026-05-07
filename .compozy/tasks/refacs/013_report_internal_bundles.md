# Iteration 013 Report: `internal/bundles`

## Scope

- Package: `github.com/pedronauck/agh/internal/bundles`
- Deterministic order index: 13 from `rtk go list ./internal/...`
- Next package: `github.com/pedronauck/agh/internal/bundles/model`
- Analysis modes: `$refactoring-analysis`, `$extreme-software-optimization`, `$systematic-debugging`, `$no-workarounds`
- Subagents: read-only refactoring explorer and read-only performance explorer

## Baseline

Commands run before implementation:

```bash
rtk go test ./internal/bundles -count=1
rtk golangci-lint run ./internal/bundles
rtk proxy go test ./internal/bundles -cover -count=1
rtk go test -tags integration ./internal/bundles -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bundles -count=1
rtk proxy go test ./internal/bundles -run '^$' -bench . -benchmem -count=5
```

Observed baseline:

- Unit tests: `54 passed in 1 packages`
- Integration-tag tests: `57 passed in 1 packages`
- Race package tests: passed
- Package lint: no issues
- Coverage: `78.8% of statements`
- `BenchmarkServiceListActivationsLargeCatalog`: about `1.06-1.09 ms/op`, `1.869 MB/op`, `13075-13077 allocs/op`
- `BenchmarkServiceBuildLargeCatalog`: about `1.37-1.43 ms/op`, `2.273 MB/op`, `11958-11960 allocs/op`

## Findings

### P0: `service.go` concentrated unrelated concerns

`service.go` contained lifecycle API methods, activation resolution, lookup indexing, resource materialization, conflict validation, ID generation, clone helpers, network-settings projection, and rollback helpers. This increased change cost and made the hottest paths harder to profile.

Implemented a scoped split for the lowest-risk cohesive blocks:

- `internal/bundles/clone.go`: defensive clone helpers and bundle-profile hash helper.
- `internal/bundles/ids.go`: stable hash ID construction.
- `internal/bundles/lookup.go`: bundle-record lookup/index helpers and profile lookup.

Deferred a larger materializer/lifecycle split because it would be a broader mechanical move with limited behavioral or performance benefit after the focused fixes below. The remaining service decomposition is documented as follow-up, not hidden.

### P1: redundant internal clones in activation resolution

Profiles showed allocation pressure under `resolveActivationFromBundleLookup` and `resolveActivation` from cloning bundle/profile payloads that were only used internally and then cloned again at public boundaries.

Implemented:

- `resolveActivation` and `resolveActivationFromBundleLookup` now keep local bundle/profile values without a second internal deep clone.
- Public `Catalog`, `PreviewActivation`, `GetActivation`, and `ListActivations` still return defensive copies.
- New tests mutate returned previews and verify subsequent reads are unchanged.

### P1: `Build` re-cloned local projection state

`Build` created local state and then deep-cloned unexported slices/maps into the returned plan. The plan fields are unexported and produced from local state, so those copies were unnecessary on the hot path.

Implemented:

- `Build` now attaches local desired-state slices/maps directly to `BundleActivationResourcePlan`.
- Input-isolation tests mutate bundle records after `Build` and prove the returned plan remains stable.

### P1: aggregate slices and owner maps grew from zero

Allocation profiles showed repeated growth in desired-state aggregates and owner maps while composing large catalogs.

Implemented:

- `estimateDesiredStateCapacity` pre-sizes desired agents, souls, heartbeats, jobs, triggers, bridges, and declared channels.
- `ownedResourceMaps` now pre-counts inventory kinds and sizes per-kind owner maps up front.

### P1: bundle lookup copied large records into the exact map

The exact lookup map stored full `resources.Record[BundleResourceSpec]` values. Those records include bundle specs, so map assignment copied large structs in the hot path.

Implemented:

- `bundleRecordLookup.exact` now stores record indexes.
- The original ordered record slice remains the authoritative backing store.
- Existing case/trim lookup behavior remains covered.

### P1: desired-agent conflict validation was quadratic

`validateDesiredAgentScopeConflicts` compared each desired agent against prior agents with the same name. Large catalogs with repeated workspace-scoped agent names exercised the O(n^2) path.

Implemented:

- Replaced the scan with per-agent-name scope indexes.
- Preserved global-vs-any and workspace-vs-same-workspace conflict behavior.

### P1: `stableID` allocated intermediate normalized slices/join strings

`stableID` was visible in allocation profiles and constructed a normalized slice plus joined string before hashing.

Implemented:

- `stableID` now hashes a directly-built byte buffer with the same trimmed-part and newline-separator semantics.
- Golden ID tests cover bundle, activation, job, agent, bridge, and empty-part cases.

### P1: canceled contexts reached store methods

`Build` and `Apply` checked `ctx.Err()` after readiness, but other service methods relied on downstream stores to notice cancellation. Test stores do not necessarily respect context, and production call paths should fail before side effects when the caller already canceled.

Implemented:

- `Service.checkReady` now returns `ctx.Err()` after nil/service/store validation.
- A focused test proves a canceled `Catalog` call does not reach `ListBundleResources`.

### P2: nested mutable bundle payloads were only shallow-cloned

Bundle clone helpers copied jobs/triggers/bridges by slice only. Nested mutable data such as trigger filters, bridge delivery defaults, bridge secret slots, and task ownership pointers could leak through public previews or plans.

Implemented:

- Deep clones for bundle jobs, triggers, bridge presets, automation jobs/triggers, bridge instances, and `JobTaskConfig.Owner`.
- Tests mutate nested preview and input payloads and verify subsequent reads/plans remain stable.

## Deferred

- `bundleProfileSpecContentHash` still allocates heavily due to JSON canonicalization and clone normalization. It is behavior-sensitive because persisted activation drift detection depends on exact hash bytes. Defer until a dedicated hash compatibility test matrix exists.
- Batch inventory listing for the real `ResourceStore.ListActivations` path remains plausible but unproven by the current benchmark, which uses `memoryStore`. Defer until a ResourceStore-backed benchmark isolates it.
- Full `service.go` decomposition into lifecycle/resolver/materializer files is still valuable, but the implemented split removed the low-risk helper clusters first. The remaining extraction should be handled in a separate structural pass if `service.go` remains a repeated edit hotspot.
- A shared owned-resource projection abstraction across projectors is broader than `internal/bundles`; defer to a cross-package refactor if similar duplication appears in more packages during later loop iterations.

## Behavior Proof

- Ordering preserved: yes. Activation iteration, declared-channel ordering, inventory sorting, and resource sort comparators were not changed.
- Tie-breaking unchanged: yes. Existing comparators and fallback lookup scans remain unchanged.
- Floating point: N/A.
- RNG: N/A.
- ID compatibility: preserved by golden tests for stable ID outputs.
- Public mutation isolation: proved by tests that mutate nested preview payloads and verify subsequent previews remain unchanged.
- Build input isolation: proved by tests that mutate bundle records after `Build` and verify returned plan contents remain unchanged.
- Context behavior: improved to fail before store access when context is already canceled.

## Performance Results

After implementation:

```bash
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(ListActivationsLargeCatalog|BuildLargeCatalog)$' -benchmem -count=5
```

Observed after results:

- `BenchmarkServiceListActivationsLargeCatalog`: about `0.93-1.01 ms/op`, `1.539 MB/op`, `10641-10644 allocs/op`
- `BenchmarkServiceBuildLargeCatalog`: about `1.01-1.08 ms/op`, `1.449 MB/op`, `9160-9162 allocs/op`

Approximate improvement from baseline:

- List activations: allocations reduced by ~333 KB/op and ~2,433 allocs/op.
- Build: allocations reduced by ~824 KB/op and ~2,800 allocs/op.

Post-change allocation profiles:

```bash
rtk proxy go test ./internal/bundles -run '^$' -bench '^BenchmarkServiceBuildLargeCatalog$' -benchmem -benchtime=200x -count=1 -memprofile /tmp/agh-bundles-013-build-after.mem -memprofilerate=1
rtk proxy go test ./internal/bundles -run '^$' -bench '^BenchmarkServiceListActivationsLargeCatalog$' -benchmem -benchtime=200x -count=1 -memprofile /tmp/agh-bundles-013-list-after.mem -memprofilerate=1
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/agh-bundles-013-build-after.mem
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/agh-bundles-013-list-after.mem
```

Remaining top nodes are dominated by bundle-profile hash canonicalization and defensive clones. Those were intentionally not rewritten in this iteration because the exact persisted hash bytes are a compatibility invariant.

## Files Changed

- `internal/bundles/service.go`
- `internal/bundles/resource_projection.go`
- `internal/bundles/service_test.go`
- `internal/bundles/clone.go`
- `internal/bundles/ids.go`
- `internal/bundles/lookup.go`
- `internal/bundles/service_refac_test.go`

## Validation

Final validation commands:

```bash
rtk go test ./internal/bundles -run 'Test(ServiceRefacs|StableIDGoldenValues|FindBundleResourceRecordIndexedNormalizesLookupKeys|BundleActivationBuildComposesTypedBundleDependency)$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bundles/service_refac_test.go
rtk go test ./internal/bundles -count=1
rtk golangci-lint run ./internal/bundles
rtk proxy go test ./internal/bundles -cover -count=1
rtk go test -tags integration ./internal/bundles -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bundles -count=1
rtk go test ./internal/bundles ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon -count=1
rtk golangci-lint run ./internal/bundles ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(ListActivationsLargeCatalog|BuildLargeCatalog)$' -benchmem -count=5
rtk make verify
```

Observed results:

- Focused refactor tests: `13 passed in 1 packages`
- Package tests: `66 passed in 1 packages`
- Integration-tag package tests: `69 passed in 1 packages`
- Race package tests: passed
- Package lint: no issues
- Dependent package tests: `1827 passed in 5 packages`
- Dependent package lint: no issues
- Coverage: `80.6% of statements`
- `make verify`: passed
