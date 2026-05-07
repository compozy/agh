# Refacs Run 005: `internal/api/httpapi`

## Package

- Import path: `github.com/pedronauck/agh/internal/api/httpapi`
- Directory: `internal/api/httpapi`
- Status: completed
- Date: 2026-05-06

## Scope

This run audited the HTTP transport package for refactoring and performance opportunities. The package is large and user-facing: it owns Gin route registration, HTTP middleware, HTTP server lifecycle, static asset serving, and HTTP-only extension/resource controls while embedding shared `internal/api/core` handlers.

The implementation stayed focused on defects and local improvements with clear behavioral proof. Broader cross-transport consolidation work was recorded as deferred because it spans `httpapi`, `udsapi`, and `core`.

## Baseline

- `rtk go test ./internal/api/httpapi -count=1`: passed before edits (`211 passed`)
- `rtk go test -tags integration ./internal/api/httpapi -count=1`: passed before edits (`260 passed`)
- `rtk proxy go test ./internal/api/httpapi -cover -count=1`: passed before edits (`80.6% of statements`)
- `rtk golangci-lint run ./internal/api/httpapi`: passed before edits
- CPU profile: dominated by test tempdir/syscall and Gin route registration; no production handler CPU hotspot justified deeper optimization.
- Memory profile: highlighted static serving and route setup allocations. Static serving had a real per-request allocation path because `serveAsset` read the whole asset with `fs.ReadFile` before `http.ServeContent`.

## Subagent Findings

### Refactoring Explorer

- Found a transport parity defect: OpenAPI/spec and UDS exposed the agent kernel routes, but HTTP only registered `/api/agent/context`, `/api/agent/soul`, and `/api/agent/soul/validate`.
- Found a CORS mismatch: HTTP declares and uses many `PATCH` routes, but preflight responses omitted `PATCH` from `Access-Control-Allow-Methods`.
- Found local duplication in loopback guards and a correctness edge case where `canonicalHost("127.0.0.1:2123")` and `canonicalHost("localhost:2123")` were not normalized before loopback checks.
- Flagged duplicated HTTP/UDS extension handler logic as real, but broader than this package-only run because the correct fix is to move shared behavior into `internal/api/core`.
- Flagged broad route/server lifecycle size and test-file size as maintainability pressure, but not a focused, low-risk run-005 change.

### Performance Explorer

- Found no production CPU hotspot worth changing.
- Recommended replacing static `fs.ReadFile` serving with `fs.Open` plus `io.ReadSeeker` so embedded/static assets can stream through `http.ServeContent` without allocating a full `[]byte` copy per request.
- Deferred route-registration allocation work because it is startup-only and dominated by Gin/spec/test initialization.

## Changes Implemented

### HTTP Agent Kernel Route Parity

HTTP now registers the spec-declared agent kernel surface:

- `GET /api/agent/me`
- `GET /api/agent/coordinator/config`
- `POST /api/agent/spawn`
- `GET /api/agent/channels`
- `GET /api/agent/channels/:channel/recv`
- `POST /api/agent/channels/:channel/send`
- `POST /api/agent/channels/reply`
- `POST /api/agent/tasks/claim-next`
- `POST /api/agent/tasks/:run_id/heartbeat`
- `POST /api/agent/tasks/:run_id/complete`
- `POST /api/agent/tasks/:run_id/fail`
- `POST /api/agent/tasks/:run_id/release`

The fix also added HTTP server wiring for `core.CoordinatorConfigResolver`, and the daemon composition root now passes `deps.CoordinatorConfig` into `httpapi`. Without that wiring, the new HTTP coordinator-config route would compile but return service-unavailable in real daemon use.

Coverage added:

- `TestHTTPAgentKernelRoutesMatchDocumentedSpecOperations`
- `TestServerHandlerConfigIncludesCoordinatorConfig`
- Existing route enumeration updated to include the new agent kernel routes.

### CORS PATCH Preflight

`corsMiddleware` now advertises `PATCH` in `Access-Control-Allow-Methods`, matching the registered HTTP API surface.

Coverage added:

- `TestCORSMiddlewareAllowsPatchPreflight`

### Loopback Host Normalization And Guard Deduplication

`canonicalHost` now normalizes host:port inputs before loopback/wildcard checks. This fixes local bind forms such as `127.0.0.1:2123`, `localhost:2123`, and `[::1]:2123`.

The duplicated `loopbackAPIGuard` and `loopbackMutationGuard` bodies were consolidated through a shared `loopbackGuard` helper while preserving distinct error messages.

Coverage added:

- `TestCanonicalHostNormalizesBoundHostPorts`
- `TestLoopbackGuardsHandleBoundHostPorts`

### Static Asset Serving Allocation Reduction

`serveAsset` now opens assets from the configured `fs.FS` and uses `http.ServeContent` directly when the file implements `io.ReadSeeker`. This avoids the previous unconditional whole-file `[]byte` allocation for seekable embedded/static assets. A fallback remains for non-seekable test or custom file systems.

Close errors are handled and logged instead of discarded.

Post-change static memprofile evidence:

- `rtk proxy go test ./internal/api/httpapi -run '^TestStaticRoutes' -count=1 -memprofile /tmp/httpapi-static-after.mem`
- `rtk go tool pprof -top /tmp/httpapi-static-after.mem`
- The post-change profile no longer shows a direct `fs.ReadFile`/whole-file copy path in `serveAsset`; remaining static-route allocation is dominated by `http.ServeContent` MIME initialization in the small focused test run.

### Server Lifecycle Cleanup

`Server.Start` now creates the stream context with `context.WithoutCancel(ctx)` plus an explicit cancel function. This keeps the intended detached stream lifetime while avoiding a raw `context.Background()` fork and preserving request-scoped context values.

Duplicate-start listener cleanup now returns a joined error if closing the extra listener fails, instead of discarding the close error.

## Deferred Findings

- Consolidate duplicated HTTP/UDS extension management behavior into `internal/api/core`. This is the right fix for transport drift, but it is cross-package and should be done as a dedicated core/transport refactor.
- Rework shared approve/cancel/session-prompt drain logic across HTTP and UDS. The current duplication is visible, but not isolated to `httpapi`.
- Split the largest legacy test files (`httpapi_integration_test.go`, `handlers_test.go`, `server_test.go`) into cohesive files. This is useful but high-churn and should be a test-structure run rather than mixed into route correctness work.
- Optimize Gin route-registration allocations only if startup profiling shows user-visible startup cost. Current profiles show startup/test initialization, not a request hot path.

## Validation

```bash
rtk go test ./internal/api/httpapi -run 'Test(CanonicalHostNormalizesBoundHostPorts|LoopbackGuardsHandleBoundHostPorts|CORSMiddlewareAllowsPatchPreflight|HTTPAgentKernelRoutesMatchDocumentedSpecOperations|ServerHandlerConfigIncludesCoordinatorConfig|StaticRoutes)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/httpapi/middleware_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/httpapi/routes_refac_test.go
rtk golangci-lint run ./internal/api/httpapi
rtk go test ./internal/api/httpapi -count=1
rtk go test ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/daemon -run 'Test(HTTPAgentKernelRoutesMatchDocumentedSpecOperations|ServerHandlerConfigIncludesCoordinatorConfig|CoordinatorConfig|RegisterTaskRoutesUseSharedHandlerBindings|UDSTransportTaskSurfaceMatchesHTTPAndDocumentedSpecOperations)' -count=1
rtk proxy go test ./internal/api/httpapi -run '^TestStaticRoutes' -count=1 -memprofile /tmp/httpapi-static-after.mem
rtk go tool pprof -top /tmp/httpapi-static-after.mem
rtk go test -tags integration ./internal/api/httpapi -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/httpapi -count=1
rtk go test ./internal/daemon -count=1
rtk proxy go test ./internal/api/httpapi -cover -count=1
rtk make verify
```

Observed results:

- Focused HTTP tests: `14 passed`
- Unit package: `222 passed`
- Integration package: `271 passed`
- Race package: passed
- Daemon package: `613 passed`
- Coverage: `81.4% of statements`
- `golangci-lint`: no issues
- `rtk make verify`: passed

## Files Changed

- `internal/api/httpapi/handlers.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/httpapi/middleware.go`
- `internal/api/httpapi/middleware_refac_test.go`
- `internal/api/httpapi/routes.go`
- `internal/api/httpapi/routes_refac_test.go`
- `internal/api/httpapi/server.go`
- `internal/api/httpapi/static.go`
- `internal/daemon/daemon.go`

## Next Package

`github.com/pedronauck/agh/internal/api/spec`
