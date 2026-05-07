# Iteration 008 Refactoring Report: `internal/api/udsapi`

## Scope

- Package: `github.com/pedronauck/agh/internal/api/udsapi`
- Iteration: 008
- Date: 2026-05-06
- Skills applied: `refactoring-analysis`, `extreme-software-optimization`, `systematic-debugging`, `no-workarounds`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`
- Subagents:
  - Refactoring explorer: read-only analysis of UDS transport lifecycle, route ownership, extension parity, handler wiring, and test shape.
  - Performance explorer: read-only profiling of route registration, streams, server lifecycle tests, and UDS integration lifecycle.

## Baseline

- `rtk go test ./internal/api/udsapi -count=1`: passing before this iteration (`198 passed` observed before edits).
- `rtk golangci-lint run ./internal/api/udsapi`: passing before this iteration.
- `rtk proxy go test ./internal/api/udsapi -cover -count=1`: `70.9%` statement coverage before edits.
- Package size: about 12.9k lines across UDS production and test files; tests dominate the package.

## Findings

### Implemented

1. Duplicate `Start` could unlink the active UDS socket.
   - Root cause: `Server.Start` removed the configured socket path before checking whether the server was already running.
   - Risk: a duplicate `Start` call on an active server could remove the live socket and make the daemon unreachable even though the duplicate start returned an error.
   - Fix: the server now checks lifecycle state before socket directory creation, stale socket removal, and listener creation.

2. UDS server lifecycle had only a boolean `started` flag.
   - Root cause: shutdown copied and cleared server fields before the actual HTTP shutdown, listener close, serve loop wait, prompt drain, and socket cleanup completed.
   - Risk: a concurrent `Start` could observe `started=false` while the old server was still stopping.
   - Fix: replaced the boolean with explicit `serverStateStopped`, `serverStateRunning`, and `serverStateStopping`; `Start` now rejects both running and stopping states, and `Shutdown` transitions to stopped only after the drain path completes.

3. Production cleanup discarded listener/socket errors.
   - Root cause: startup cleanup used ignored errors for listener close and socket removal on chmod/duplicate paths.
   - Risk: cleanup failures were hidden and violated AGH no-discard error discipline.
   - Fix: introduced `cleanupSocketStartFailure` and joined cleanup errors with the startup failure.

4. UDS stream lifetime was rooted in `context.Background()`.
   - Root cause: `Start` created stream cancellation state from `context.Background()`.
   - Risk: it diverged from the HTTP transport pattern and erased caller context values.
   - Fix: stream cancellation is now detached with `context.WithoutCancel(ctx)` and then explicitly canceled by `Shutdown`.

5. `Shutdown(nil)` silently used `context.Background()`.
   - Root cause: nil shutdown context fallback.
   - Risk: a caller could accidentally create a shutdown with no deadline and no cancellation.
   - Fix: `Shutdown(nil)` now returns a concrete error, matching `Start(nil)` discipline.

6. UDS extension error mapping diverged from HTTP.
   - Root cause: UDS did not map `extension.ErrExtensionExists`.
   - Risk: duplicate extension install could return `500` over UDS while HTTP returned `409`.
   - Fix: UDS now maps `ErrExtensionExists` to `409 Conflict`, with status mapping coverage.

7. Hosted MCP stream error envelopes used an untyped `map[string]any`.
   - Root cause: stream error payload construction used a dynamic map for a stable public JSON shape.
   - Risk: weaker compile-time contract and inconsistent with prior typed SSE payload cleanup in `internal/api/core`.
   - Fix: replaced the map with a typed `hostedMCPStreamErrorPayload` struct.

8. UDS-owned Gin engines emitted debug startup/route output in debug mode.
   - Root cause: `gin.New()` and route registration ran while Gin could be in global debug mode.
   - Risk: noisy daemon startup and measurable repeated allocation in tests. The performance explorer measured `gin.debugPrintRoute` at `23.68MB` cumulative allocation in the server lifecycle profile.
   - Fix: UDS-owned engine creation and route registration now temporarily suppress Gin debug mode under a package-local mutex, then restore the previous global Gin mode.

9. `shortSocketPath` was not collision-safe under parallel tests.
   - Root cause: test socket paths used `time.Now().UnixNano()` and cleanup ignored removal errors.
   - Risk: parallel tests could bind the same path or delete each other's socket, causing intermittent address-in-use and missing-socket failures.
   - Fix: the helper now uses a process-scoped atomic counter and handles cleanup errors.

10. Empty package files created false ownership signals.
    - Root cause: `memory.go` and `workspaces.go` contained only `package udsapi`.
    - Risk: low, but they suggested local UDS ownership for behavior that now lives in shared core handlers.
    - Fix: removed both empty files.

### Deferred

1. Route registration remains manually mirrored between HTTP and UDS.
   - Refactoring explorer found shotgun-surgery risk across `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, specs, and route tests.
   - Deferred because a shared declarative route registry is cross-transport architecture work and should be handled as its own iteration/spec.

2. Handler dependency wiring is structurally duplicated.
   - `Server`, `handlerConfig`, `handlerConfig()`, and `newHandlers` still mirror many dependencies.
   - Deferred because a direct `core.BaseHandlerConfig` composition change would touch broad construction paths and belongs in a dedicated refactor.

3. Prompt streaming logic is duplicated with HTTP.
   - UDS and HTTP prompt streaming are similar but have real contract differences.
   - Deferred until lifecycle semantics are stable across both transports.

4. Large test files still need structural splitting.
   - `handlers_test.go`, `udsapi_integration_test.go`, and `transport_parity_integration_test.go` remain large and include pre-existing test-shape/no-discard debt.
   - This iteration touched only focused tests/helpers needed for the lifecycle and parity fixes.

5. Caching `apispec.Operations()` inside route-parity tests was not kept.
   - Performance explorer identified test allocation from repeated defensive copies.
   - The change would touch large parity files with pre-existing convention debt; it was deferred to avoid expanding this package iteration into a broad test-normalization pass.

## Files Changed

- `internal/api/udsapi/server.go`
- `internal/api/udsapi/server_test.go`
- `internal/api/udsapi/server_env_test.go`
- `internal/api/udsapi/helpers_test.go`
- `internal/api/udsapi/extensions.go`
- `internal/api/udsapi/extensions_additional_test.go`
- `internal/api/udsapi/hosted_mcp.go`
- `internal/api/udsapi/hosted_mcp_test.go`
- `internal/api/udsapi/memory.go` removed
- `internal/api/udsapi/workspaces.go` removed

## Validation

```bash
rtk go test ./internal/api/udsapi -run 'Test(ServerStart|HostedMCPStreamErrorData|ExtensionStatusCodeMappings|RegisterNetworkRoutesMatch|UDSTransportTaskSurface)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/server_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/server_env_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/extensions_additional_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/hosted_mcp_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/helpers_test.go
rtk go test ./internal/api/udsapi -count=1
rtk golangci-lint run ./internal/api/udsapi
rtk go test -tags integration ./internal/api/udsapi -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/udsapi -count=1
rtk proxy go test ./internal/api/udsapi -cover -count=1
rtk go test ./internal/api/core ./internal/api/httpapi ./internal/api/spec ./internal/api/udsapi ./internal/cli ./internal/daemon -count=1
rtk go test -tags integration ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1
rtk go test ./internal/api/udsapi -run '^TestServerStartRejectsRestartDuringShutdown$' -count=20
rtk go test ./internal/api/udsapi -run '^TestServerStartDuplicateKeepsActiveSocket$' -count=20
rtk proxy go test ./internal/api/udsapi -run '^(TestNew|TestPath|TestServer|TestEnsure|TestSocket)' -count=20 -memprofile=/tmp/udsapi-server-after.mem -memprofilerate=1
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/udsapi-server-after.mem
```

Observed results:

- Focused UDS lifecycle/parity tests: `24 passed`.
- Full UDS package tests: `213 passed`.
- UDS integration-tag package tests: `246 passed`.
- UDS race package tests: passing.
- Direct dependent package tests: `2599 passed in 6 packages`.
- Direct integration dependent package tests: `1251 passed in 3 packages`.
- Coverage after edits: `70.4%` statements.
- After the Gin quieting change, the server lifecycle memory profile no longer shows `gin.debugPrintRoute` in the top allocation nodes; the remaining top allocator is route trie construction (`gin.(*node).addRoute`), which is expected one-time setup.

Full monorepo gate:

```bash
rtk make verify
```

Result: passed.

## Next Package

- `github.com/pedronauck/agh/internal/automation`
