# Iteration 007 Report: `internal/api/testutil`

## Scope

- Package: `github.com/pedronauck/agh/internal/api/testutil`
- Iteration: 007
- Date: 2026-05-06
- Skills applied: `refactoring-analysis`, `extreme-software-optimization`, `systematic-debugging`, `no-workarounds`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`
- Read-only subagents:
  - Refactoring explorer: large-module / divergent-change audit
  - Performance explorer: profile-driven hotspot audit

## Baseline

- `rtk go test ./internal/api/testutil -count=1` passed before edits.
- `rtk golangci-lint run ./internal/api/testutil` passed before edits.
- `rtk proxy go test ./internal/api/testutil -cover -count=1` showed 1.0% package coverage before edits.

## Findings

### Implemented

1. **Large module / divergent change in `apitest.go`**
   - `internal/api/testutil/apitest.go` had 1,768 lines mixing config helpers, session stubs, observer stubs, task stubs, resource stubs, automation stubs, network stubs, bridge stubs, workspace stubs, skills stubs, HTTP request helpers, SSE helpers, session fixtures, and logger helpers.
   - Risk: unrelated API test changes all touched one file, raising merge-conflict and review cost.
   - Fix: split the package into cohesive files while preserving package name, exported identifiers, method behavior, and public helper signatures.

2. **Stub declarations and interface assertions were far from their methods**
   - Several stub types were declared near the top while methods appeared much later in historical append order.
   - Fix: co-located each stub type with its methods and compile-time interface assertion.

3. **Weak package-local contract coverage for high-afferent test helpers**
   - `internal/api/testutil` is used by API core, HTTP, UDS, CLI, and daemon tests, but only one package-local behavior was covered.
   - Fix: expanded `apitest_test.go` to cover the most important stable helper contracts:
     - disabled-network home/config creation
     - `StubSessionManager.List` fallback behavior
     - `PerformRequestWithHeaders` content-type/header behavior
     - multiline/final-record SSE parsing
     - deterministic session fixtures
     - `StubResourceService.Put` JSON cloning and deterministic metadata
     - workspace stub unconfigured-method errors

4. **Repeated test home/config creation in API core tool tests**
   - The performance explorer found repeated `NewTestHomePaths(t)` calls in `internal/api/core/tools_test.go`: one call for `HomePaths` and a second call inside `ConfigWithDisabledNetwork`.
   - Fix: added `NewDisabledNetworkHomeConfig(t)` in `internal/api/testutil` and updated the four duplicate core tool test callsites to derive `HomePaths` and config from the same test home.

### Deferred

1. **Fail-fast strict stubs**
   - Many stubs intentionally return permissive zero values (`nil, nil`) when a function field is not configured.
   - Deferred because changing default stub semantics would churn many consumer tests and could mask a structural API decision inside this package-only iteration.

2. **Cross-transport fixture consolidation**
   - HTTP and UDS helper packages still duplicate some settings/envelope fixtures.
   - Deferred because it crosses package ownership and belongs in a later transport-focused iteration.

3. **Manual `httptest` request construction**
   - `PerformRequestWithHeaders` appears in allocation profiles, but most cost is canonical `httptest.NewRequestWithContext` and `engine.ServeHTTP`.
   - Deferred because replacing it would need a high-cost behavior proof for URL, host, request URI, body, header, Gin routing, and recorder semantics. Score stayed below the optimization threshold.

## Files Changed

- `internal/api/testutil/apitest.go`
- `internal/api/testutil/config.go`
- `internal/api/testutil/session_stub.go`
- `internal/api/testutil/observer_stub.go`
- `internal/api/testutil/automation_stub.go`
- `internal/api/testutil/task_stub.go`
- `internal/api/testutil/resource_stub.go`
- `internal/api/testutil/network_stub.go`
- `internal/api/testutil/bridge_stub.go`
- `internal/api/testutil/workspace_stub.go`
- `internal/api/testutil/skills_stub.go`
- `internal/api/testutil/home_helpers.go`
- `internal/api/testutil/session_fixtures.go`
- `internal/api/testutil/http_helpers.go`
- `internal/api/testutil/sse.go`
- `internal/api/testutil/logger.go`
- `internal/api/testutil/apitest_test.go`
- `internal/api/core/tools_test.go`

## Performance Evidence

Baseline from the performance explorer:

```bash
rtk go test ./internal/api/core -run 'TestTool' -count=20 \
  -cpuprofile=/tmp/agh-api-testutil-tools.cpu \
  -memprofile=/tmp/agh-api-testutil-tools.mem \
  -memprofilerate=1
```

- Before: `NewTestHomePaths` accounted for 110ms CPU and 1.29MB allocation in the focused `TestTool` profile.
- After replacing the four duplicate home/config callsites:
  - `NewTestHomePaths` accounted for 60ms CPU in `/tmp/agh-api-testutil-tools-after.cpu`.
  - `NewTestHomePaths` accounted for 681.78kB in `/tmp/agh-api-testutil-tools-after.mem`.

This change preserves handler behavior because each affected test now uses the same generated home for both `HomePaths` and config, instead of two independent generated homes.

## Validation

```bash
rtk go test ./internal/api/testutil -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/testutil/apitest_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/tools_test.go
rtk rg -n "ConfigWithDisabledNetwork\\(testutil\\.NewTestHomePaths\\(t\\)\\)" internal
rtk go test ./internal/api/core -run '^TestTool' -count=20 -cpuprofile=/tmp/agh-api-testutil-tools-after.cpu -memprofile=/tmp/agh-api-testutil-tools-after.mem -memprofilerate=1
rtk go tool pprof -list='github.com/pedronauck/agh/internal/api/testutil.NewTestHomePaths' /tmp/agh-api-testutil-tools-after.cpu
rtk go tool pprof -list='github.com/pedronauck/agh/internal/api/testutil.NewTestHomePaths' /tmp/agh-api-testutil-tools-after.mem
rtk go test ./internal/api/testutil ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1
rtk golangci-lint run ./internal/api/testutil ./internal/api/core
rtk env CGO_ENABLED=1 go test -race ./internal/api/testutil -count=1
rtk go test -tags integration ./internal/api/testutil -count=1
rtk go test ./internal/daemon -run '^TestTool' -count=1
rtk go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/daemon -count=1
rtk proxy go test ./internal/api/testutil -cover -count=1
rtk go test -tags integration ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/testutil -count=1
rtk make verify
```

Results:

- Package testutil: 14 tests passed.
- Focused `TestTool`: 140 tests passed.
- API package unit validation: 1,147 tests passed in 4 packages.
- API/core/httpapi/udsapi/cli/daemon unit validation: 2,430 tests passed in 5 packages.
- API integration-tag validation: 1,250 tests passed in 4 packages.
- Package coverage increased from 1.0% to 10.8%.
- `golangci-lint` reported no issues for `internal/api/testutil` and `internal/api/core`.
- `make verify` passed.

## Next Package

- Next package in deterministic `go list ./internal/...` order: `github.com/pedronauck/agh/internal/api/udsapi`
