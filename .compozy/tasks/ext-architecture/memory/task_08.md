# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Wire extension boot into the daemon composition root so installed extensions start after hooks initialize, contribute hook declarations through the live hooks runtime, rebuild hooks after startup, and stop before sessions/servers on daemon shutdown.

## Important Decisions
- Kept extension boot non-fatal: `bootExtensions()` logs start or rebuild failures and continues daemon boot instead of blocking the daemon on optional extension state.
- Reused the existing hooks declaration pipeline by chaining `extensionDeclarationProvider(...)` onto the config provider instead of introducing a separate extension-only rebuild path.
- Added the minimal `GlobalDB.DB()` seam so the daemon can construct `extension.Registry` from the live global database without widening unrelated registry interfaces.
- Built the real runtime through one daemon-owned `HostAPIHandler` plus `extension.Manager` factory wiring, so Host API handlers and extension lifecycle share the same boot-time dependencies.

## Learnings
- The integration helper subprocess needs a unit-level harness check because the helper command/env builders are referenced only by the integration-tagged suite; without that explicit unit usage, golangci-lint flags them as unused in the default test build.
- Rebuilding hooks after the extension manager start attempt is still useful when startup partially fails, because healthy extensions can continue contributing hook declarations even if one extension is corrupt.

## Files / Surfaces
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/hooks_bridge.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/extension/manager.go`
- `internal/store/globaldb/global_db.go`

## Errors / Corrections
- Initial fresh `make verify` failed on `unused` helpers in `internal/daemon/daemon_test.go`; corrected by adding `TestDaemonExtensionHelperHarness` so the helper command/args/env builders are exercised in the non-tagged unit build.

## Ready for Next Run
- Verification evidence after the final code change:
  - `go test ./internal/daemon -count=1`
  - `go test -tags integration ./internal/daemon -count=1`
  - `go test -cover ./internal/daemon -count=1` → `coverage: 81.2% of statements`
  - `make verify`
- Tracking files for task 08 still need to stay aligned with the verified state if any later follow-up changes land.
