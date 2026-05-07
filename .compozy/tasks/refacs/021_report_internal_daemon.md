# Refacs 021: `internal/daemon`

## Scope

- Package: `github.com/pedronauck/agh/internal/daemon`
- Iteration: 021
- Goal: deep refactoring and performance audit for daemon composition-root cleanup, shutdown, resource publication, and package-local hot paths.
- Subagents:
  - Read-only refactoring audit for `internal/daemon`.
  - Read-only performance/concurrency audit for `internal/daemon`.

## Baseline

Commands and evidence gathered before and during implementation:

```bash
rtk go test ./internal/daemon -count=1
rtk golangci-lint run ./internal/daemon
rtk proxy go test ./internal/daemon -cover -count=1
rtk go test -tags integration ./internal/daemon -run 'TestDaemonE2E.*Coordinator|TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents|TestDaemonE2ENetworkWhoisAndCapabilityExchange|TestAgentSkillResources|TestToolMCPResources|TestRestart' -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/daemon -count=1
rtk proxy go test ./internal/daemon -run '^$' -bench . -benchmem -count=3
```

Observed baseline:

- Package tests passed before scoped edits.
- Package lint passed before scoped edits.
- Coverage was `72.8% of statements`; this is below the repo's ideal package floor but was the existing state for this very large package.
- Focused integration-tag daemon tests passed.
- Race package tests passed in the performance subagent's snapshot.
- The package benchmark suite initially failed because `BenchmarkAgentSkillSourceSyncerSyncNoop` still used stale `"lookup"` tool IDs that no longer match the canonical built-in tool ID taxonomy.
- `BenchmarkResourceAgentCatalogResolveAgentWorkspaceHit` cloned the full resource catalog for a single workspace-agent lookup: about `137-176 us/op`, `393648 B/op`, `1542 allocs/op` in the focused pre-change baseline.
- `BenchmarkToolMCPSourceSyncerSyncNoop` decoded typed current records on a no-op sync: about `619-626 us/op`, `~632 KB/op`, `7892 allocs/op` in the read-only performance audit.

## Findings

### P1: daemon info and lock writers discarded cleanup/durability errors

`WriteInfo`, `syncDir`, and `writeLockPID` ignored temporary-file removal, file close, and directory close failures through underscore discards. These paths write daemon runtime state and lock ownership, so cleanup/durability failures should not be silently lost.

Impact: a failed temp cleanup or close could hide a real persistence problem during daemon startup/shutdown diagnostics.

### P1: skills watcher shutdown could wait without a shutdown deadline

The skills watcher shutdown path canceled the watcher and waited for its done channel without accepting the caller's shutdown context. If the watcher failed to observe cancellation, daemon shutdown could block beyond the configured shutdown budget.

Impact: runtime shutdown should be bounded by the daemon shutdown context, especially for background workers owned by the composition root.

### P1: boot cleanup and shutdown created detached `context.Background()` paths

The refactoring audit found runtime cleanup/shutdown code that fabricated background contexts. Some daemon workers legitimately need `context.WithoutCancel`, but shutdown cleanup still needs an explicit bounded parent so failures cannot hang indefinitely.

Impact: fabricated backgrounds weaken cancellation provenance and make bounded teardown harder to reason about.

### P1: package performance benchmark had stale tool IDs

`BenchmarkAgentSkillSourceSyncerSyncNoop` used plain `"lookup"` / `"read"` IDs while the resource validators now require canonical tool IDs. This made the benchmark suite fail before it could serve as performance evidence.

Impact: broken benchmarks remove a package-local performance guardrail and can hide regressions in sync code.

### P2: resource-agent lookup cloned the full catalog per lookup

`resourceAgentCatalog.lookupAgentRecord` used `Snapshot()`, which defensively cloned every catalog record before scanning for a single agent. The lookup is a read path for resource-agent resolution and should clone only the selected record.

Impact: workspace-hit lookups paid O(n) clone cost and allocated hundreds of KB for one result.

### P2: Tool/MCP no-op sync decoded current records unnecessarily

`toolMCPSourceSyncer.syncTools` and `syncMCPServers` listed typed stores, which decoded and validated every current managed record before comparing it with already encoded desired resources. The agent/skill sync path already had a cheaper raw-record comparison pattern.

Impact: no-op publication cycles spent avoidable CPU and allocations on typed decode/clone work.

### Deferred: native extension source error classification still uses string matching

`isExtensionSourceError` still classifies some extension loader errors by inspecting `err.Error()`. A complete root-cause fix needs typed/sentinel errors from `internal/extension` for marketplace-source failure modes. This iteration intentionally stayed within `internal/daemon`.

## Changes Made

### Cleanup and shutdown hardening

- Changed `WriteInfo` to use a named return and `errors.Join` so temp-file removal and close failures are preserved with the original write error.
- Changed `syncDir` and `writeLockPID` to join directory/file close errors instead of discarding them.
- Routed boot cleanup through the boot context and a bounded `context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)` cleanup context.
- Changed `stopSkillsWatcher` to accept a shutdown context and return the context error if the watcher does not exit in time.
- Added `daemonShutdownContext` so `Run` and `Shutdown(nil)` use an explicit bounded shutdown context instead of `context.Background()`.

### Resource lookup and sync performance

- Added a catalog-level `cloneRecord` helper and reused the existing record clone helper for selected-record copies.
- Changed `resourceAgentCatalog.lookupAgentRecord` to scan under the catalog read lock and clone only the matching record.
- Changed `toolMCPSourceSyncer` to keep a `resources.RawStore` and list current managed Tool/MCP records through `ListRaw`.
- Replaced typed current-record comparison in Tool/MCP no-op sync with `sameManagedRawRecord`, matching the existing agent/skill sync strategy.
- Updated all Tool/MCP syncer construction paths, tests, and benchmarks to pass the raw resource kernel/store.

### Tests and benchmarks

- Added `TestResourceAgentCatalogLookupReturnsDefensiveCopy` to prove direct lookup still returns a defensive copy.
- Added `TestStopSkillsWatcherRespectsShutdownContext` to prove a blocked watcher returns the caller's context error and a completed watcher exits cleanly.
- Repaired stale tool IDs in daemon performance benchmarks to use canonical built-in tool IDs.
- Normalized `tool_mcp_resources_test.go` to AGH `t.Run("Should ...")` subtest shape after touching it for the raw-store constructor change.

## Performance Results

Final focused benchmark command:

```bash
rtk proxy go test ./internal/daemon -run '^$' -bench 'Benchmark(ResourceAgentCatalogResolveAgentWorkspaceHit|AgentSkillSourceSyncerSyncNoop|ToolMCPSourceSyncerSyncNoop)$' -benchmem -count=5
```

Observed final results:

- `BenchmarkResourceAgentCatalogResolveAgentWorkspaceHit`: `5790-5828 ns/op`, `752 B/op`, `8 allocs/op`.
  - Previous focused baseline: about `137-176 us/op`, `393648 B/op`, `1542 allocs/op`.
  - Result: removes the full-catalog clone from this lookup path.
- `BenchmarkToolMCPSourceSyncerSyncNoop`: `435737-441716 ns/op`, `421308-421330 B/op`, `5667 allocs/op`.
  - Previous read-only audit baseline: about `619-626 us/op`, `~632 KB/op`, `7892 allocs/op`.
  - Result: roughly one third less allocated bytes and materially fewer allocations on no-op Tool/MCP publication.
- `BenchmarkAgentSkillSourceSyncerSyncNoop`: `370310-374668 ns/op`, `325765-325854 B/op`, `5134 allocs/op`.
  - No optimization claim is made here; the benchmark was repaired and kept as a guardrail.

## Deferred / Cross-Package Notes

- Replace daemon native-extension source string matching with typed errors once `internal/extension` exposes source-not-configured/source-unavailable sentinels.
- Several broader daemon `context.Background()` sites remain outside the shutdown and boot-cleanup paths touched here; revisit with a dedicated daemon lifecycle pass if they become observable shutdown defects.
- The daemon composition root still has large data clumps in boot/runtime dependencies. Splitting them safely requires a structural design pass, not another local patch.
- HTTP/UDS wiring duplication and hook-declaration snapshot duplication remain lower-priority refactors.
- The agent/skill and Tool/MCP syncers now share the raw comparison pattern but not a generic managed-resource sync abstraction. Introduce one only when it reduces real duplication without hiding resource-kind validation details.
- The performance subagent flagged heartbeat wake/drain and harness reentry shutdown risks. They were not changed in this package iteration because no failing race/deadlock evidence was reproduced.

## Validation

Final validation commands:

```bash
rtk go test ./internal/daemon -run 'TestToolMCP|TestResourceAgentCatalogLookupReturnsDefensiveCopy|TestStopSkillsWatcherRespectsShutdownContext' -count=1
rtk go test -tags integration ./internal/daemon -run 'TestToolMCP|TestAgentSkillResources|TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents' -count=1
rtk golangci-lint run ./internal/daemon
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/agent_skill_resources_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/skills_watcher_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/perf_bench_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/tool_mcp_resources_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/tool_mcp_resources_integration_test.go
rtk proxy go test ./internal/daemon -run '^$' -bench 'Benchmark(ResourceAgentCatalogResolveAgentWorkspaceHit|AgentSkillSourceSyncerSyncNoop|ToolMCPSourceSyncerSyncNoop)$' -benchmem -count=5
rtk go test ./internal/daemon -count=1
rtk proxy go test ./internal/daemon -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/daemon -count=1
rtk rg -n "context\\.Background\\(\\)|_\\s*=" internal/daemon/boot.go internal/daemon/daemon.go internal/daemon/info.go internal/daemon/lock.go internal/daemon/agent_skill_resources.go internal/daemon/tool_mcp_resources.go internal/daemon/perf_bench_test.go internal/daemon/agent_skill_resources_refac_test.go internal/daemon/skills_watcher_refac_test.go
rtk make verify
```

Observed final results:

- Focused package tests: `16 passed in 1 packages`.
- Focused integration-tag package tests: `14 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for all touched daemon test/benchmark files.
- Focused benchmarks: passed with the final measurements above.
- Full package tests: `626 passed in 1 packages`.
- Package coverage: `72.8% of statements`.
- Race package tests: passed.
- Scoped `context.Background()` / `_ =` scan over touched daemon production/refac files: no matches.
- `make verify`: passed.
