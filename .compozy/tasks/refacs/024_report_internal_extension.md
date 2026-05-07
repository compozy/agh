# Refacs 024: `internal/extension`

## Scope

- Package: `github.com/pedronauck/agh/internal/extension`
- Iteration: 024
- Goal: deep refactoring and performance audit for the extension manager, registry, host API helpers, and extension tool provider.
- Subagents:
  - Read-only refactoring/correctness audit for `internal/extension`.
  - Read-only performance/concurrency audit for `internal/extension`.

## Baseline

Initial package state and probes:

```bash
rtk go test ./internal/extension -count=1
rtk proxy go test ./internal/extension -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/extension -count=1
rtk golangci-lint run ./internal/extension
rtk go test -tags integration ./internal/extension -count=1
```

Observed baseline:

- Package tests passed.
- Package lint passed with no issues.
- Race package tests passed.
- Package coverage was `75.9% of statements`.
- Integration-tag package tests had pre-existing provider failures:
  - `TestTeamsProviderLaunchNegotiatesBridgeRuntime`
  - `TestTeamsProviderIngressAndDeliveryConformance`
  - `TestTelegramProviderLaunchNegotiatesBridgeRuntime`

The integration failures were observed before the local changes in this iteration. They were kept as evidence and re-probed after the scoped package work.

## Findings

### P1: extension Stop/disable leaked resource source sessions

`unregisterResources` cleared capability grants and local registration flags, but it did not reset the resource source session associated with the extension. A stopped or disabled extension could therefore leave source-scoped resource state alive past its lifecycle.

Impact: resource projections could retain stale extension source authority after a stop or failure disable path.

Root cause: source session activation had an explicit manager actor, but unregister did not have the symmetric `ResetSource` call.

### P1: Manager.Stop swallowed shutdown errors when Wait succeeded

The old shutdown path only surfaced `Shutdown` errors when `Wait` also returned an error. If `Shutdown` reported a real failure while the process had already exited cleanly, `Stop` returned nil.

Impact: lifecycle shutdown failures could be hidden from callers and tests.

Root cause: the process stop path coupled `Shutdown` error reporting to the result of a follow-up `Wait` call.

### P2: shutdown helpers detached from caller context values

The manager lifecycle and shutdown timeout helpers used `context.Background()` in places where a caller or lifecycle context already existed.

Impact: request values and future context-scoped instrumentation could be dropped across lifecycle cleanup.

Root cause: helper functions created root contexts instead of deriving from the manager lifecycle or caller context.

### P2: registry cleanup paths discarded row close errors and matched SQLite errors by string

`Registry.List` and `ensureNoActiveBundles` discarded `rows.Close()` errors. `ensureNoActiveBundles` also treated a missing `resource_records` table by matching `"no such table"` inside an error string.

Impact: cleanup errors could be lost, and table-absence handling depended on SQLite error text instead of schema state.

Root cause: the registry method shape did not preserve close errors, and the active-bundle guard did not first inspect `sqlite_master`.

### P2: managed install staging cleanup errors were ignored

`InstallLocalManaged` removed the staging directory in a deferred cleanup with `_ = os.RemoveAll(...)`.

Impact: failed cleanup could be silently ignored after install failures or after a successful move.

Root cause: cleanup was fire-and-forget even though the function already returned errors to the caller.

### P2: host API task DTO builders did extra slice growth and used loop-copy pointers

The task summary/run payload builders appended into pre-capacity slices and passed the address of loop-copy variables in the run path.

Impact: the code was harder to reason about, did avoidable growth bookkeeping, and depended on pointer-to-copy behavior instead of directly addressing the backing slice element.

Root cause: generic append-oriented DTO construction had drifted into hot host API helpers.

### P2: extension tool provider reparsed manifests across List/Resolve sequences

`ExtensionToolProvider.List` and `Resolve` both walked registry entries and reparsed manifest files. A typical caller that lists tools and then resolves one of them paid the manifest load/descriptor resolution cost twice.

Impact: repeated tool catalog operations did unnecessary manifest parsing and descriptor cloning.

Root cause: the provider did not cache parsed extension-host tool descriptors behind a registry/manifest fingerprint.

### P3: local package test debt remained below the 80% target

Coverage improved from `75.9%` to `76.1%`, but the package remains below the project 80% package target.

Impact: the package has a large lifecycle and provider surface with more regression risk than the package-local coverage currently reflects.

Root cause: this package owns many integration-heavy surfaces; the current iteration added focused tests for changed behavior but did not attempt a broad coverage campaign.

## Changes Made

### Manager lifecycle cleanup

- Added `stopManagedExtension` to centralize process shutdown, resource cleanup, state mutation, and logging for `Manager.Stop`.
- Preserved both `Shutdown` and `Wait` errors with `errors.Join`.
- Preserved the existing hung-process cooperative-timeout behavior by not reporting `context.DeadlineExceeded` when the process was successfully killed and waited.
- Changed `unregisterResources` to return errors and reset the extension resource source session.
- Added `extensionManagerResourceActor` and `extensionResourceSource` helpers so activation and reset use the same actor/source taxonomy.
- Cleared `sessionNonce` when resource registration is unregistered.
- Propagated unregister/reset errors through `disableExtension`.

### Context propagation

- Changed the manager lifecycle context to derive from `context.WithoutCancel(ctx)` rather than `context.Background()`, preserving values while still detaching cancellation.
- Changed `shutdownProcessWithTimeout` to require a parent context and derive its timeout from that context.
- Threaded caller/lifecycle context into launched-process cleanup and recovery shutdown paths.

### Registry and install cleanup

- Made `Registry.List` preserve `rows.Close()` errors.
- Added `sqliteTableExists` and used it in `ensureNoActiveBundles` before querying `resource_records`.
- Removed string matching on SQLite `"no such table"` errors for the active-bundle guard.
- Made `ensureNoActiveBundles` preserve row close errors.
- Made `InstallLocalManaged` join staging `os.RemoveAll` errors into the returned error.

### Host API helper cleanup

- Removed an ignored loop variable discard in `drainAgentEvents`.
- Changed task list payload builders to allocate fixed-length result slices and assign by index.
- Changed `taskRunPayloadsFromRuns` to pass `&runs[i]` instead of the address of a loop-copy variable.
- Changed `taskSummaryPayloadFromSummary` to accept `*task.Summary` and handle nil explicitly.

### Extension tool provider cache

- Added a concurrency-safe manifest-tool cache to `ExtensionToolProvider`.
- Fingerprinted registry entries plus manifest file size/modtime before reusing cached tool descriptors.
- Returned defensive clones from cache reads and writes.
- Added `BenchmarkExtensionToolProviderListAndResolve` to track the list-plus-resolve path that benefits from cached descriptor reuse.

### Tests and benchmarks

- Added `internal/extension/manager_refac_test.go` for shutdown error reporting and resource source cleanup on stop/disable.
- Added `BenchmarkExtensionToolProviderListAndResolve` to `internal/extension/perf_bench_test.go`.
- Re-ran AGH test-convention checks for the touched test/benchmark files.

## Files Changed

- `internal/extension/host_api.go`
- `internal/extension/host_api_tasks.go`
- `internal/extension/install_managed.go`
- `internal/extension/manager.go`
- `internal/extension/manager_refac_test.go`
- `internal/extension/perf_bench_test.go`
- `internal/extension/registry.go`
- `internal/extension/tool_provider.go`

## Performance Results

Final focused benchmark command:

```bash
rtk proxy go test ./internal/extension -run '^$' -bench 'Benchmark(ExtensionToolProviderListAndResolve|TaskSummaryPayloadsFromSummaries|TaskRunPayloadsFromRuns)$' -benchmem -count=5
```

Observed final results:

- `BenchmarkTaskSummaryPayloadsFromSummaries`: about `44.7-150.7 us/op`, `212992-212995 B/op`, `513 allocs/op`.
- `BenchmarkTaskRunPayloadsFromRuns`: about `63.4-111.2 us/op`, `395648-395649 B/op`, `973 allocs/op`.
- `BenchmarkExtensionToolProviderListAndResolve`: about `154.1-214.8 us/op`, `159038-159050 B/op`, `1852 allocs/op`.

Interpretation:

- The tool-provider benchmark is a new regression guard for the list-plus-resolve access pattern. This iteration should not claim a numeric before/after speedup because no equivalent baseline benchmark existed before the cache was added.
- The task DTO builder changes are primarily structural and allocation-shape cleanup. The benchmark now captures the current cost so a later task-store/host-API pass can optimize it with evidence.

## Deferred / Cross-Package Notes

- Teams and Telegram provider integration tests failed in the baseline and still fail after the scoped package work:
  - `TestTeamsProviderLaunchNegotiatesBridgeRuntime` times out waiting for adapter state `ready`.
  - `TestTeamsProviderIngressAndDeliveryConformance` reaches `degraded`, expected `ready`.
  - `TestTelegramProviderLaunchNegotiatesBridgeRuntime` reaches `degraded`, expected `ready`.
- The Teams and Telegram provider package-level tests under `extensions/bridges/*` passed, and the focused WhatsApp/Slack/Linear provider launch integration probes passed. That points to provider/reference-extension harness drift rather than a regression from the manager/tool-provider refactor, but it remains open.
- `Registry` still uses a package-local `registryContext()` helper. Replacing it with context-bearing registry methods is a larger package API hard-cut and should be handled separately.
- The typed marketplace/source error hard-cut from iteration 021 remains open because the complete fix crosses `internal/extension` and `internal/daemon/native_extension_tools.go`.
- `HostAPIHandler` remains broad. Domain decomposition should be done as a dedicated structural pass rather than mixed into this lifecycle/resource cleanup iteration.

## Validation

Final validation commands:

```bash
rtk go test ./internal/extension -run 'TestManager(StopShutdownErrors|ResourceSourceCleanup|StopKillsHungSubprocessAfterTimeout)$' -count=1
rtk go test ./internal/extension -run 'TestExtensionToolProvider' -count=1
rtk go test ./internal/extension -count=1
rtk golangci-lint run ./internal/extension
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/manager_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/perf_bench_test.go
rtk env CGO_ENABLED=1 go test -race ./internal/extension -count=1
rtk proxy go test ./internal/extension -cover -count=1
rtk go test ./internal/extension ./internal/daemon ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/bundles ./internal/tools -count=1
rtk proxy go test ./internal/extension -run '^$' -bench 'Benchmark(ExtensionToolProviderListAndResolve|TaskSummaryPayloadsFromSummaries|TaskRunPayloadsFromRuns)$' -benchmem -count=5
rtk go test -tags integration ./internal/extension -run 'Test(TeamsProviderLaunchNegotiatesBridgeRuntime|TeamsProviderIngressAndDeliveryConformance|TelegramProviderLaunchNegotiatesBridgeRuntime)$' -count=3
rtk go test ./extensions/bridges/teams ./extensions/bridges/telegram -run 'Test.*InitialState|Test.*Launch|Test.*Provider' -count=1
rtk go test -tags integration ./internal/extension -run 'Test(WhatsappProviderLaunchNegotiatesBridgeRuntime|SlackProviderLaunchNegotiatesBridgeRuntime|LinearProviderLaunchNegotiatesBridgeRuntime)$' -count=1 -v
rtk make verify
```

Observed final results:

- Focused manager lifecycle tests: passed.
- Focused extension tool provider tests: `18 passed in 1 packages`.
- Full package tests: `529 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for `internal/extension/manager_refac_test.go` and `internal/extension/perf_bench_test.go`.
- Race package tests: passed.
- Package coverage: `76.1% of statements`.
- Direct dependent package set (`extension`, `daemon`, `api/core`, `api/httpapi`, `api/udsapi`, `cli`, `bundles`, `tools`): `3255 passed in 8 packages`.
- Focused final benchmarks passed with the results listed above.
- Known baseline integration probe failed in the same Teams/Telegram tests listed in the deferred notes.
- Teams and Telegram bridge package tests passed.
- WhatsApp, Slack, and Linear focused provider launch probes passed.
- `make verify`: passed.
