# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Mirror the HTTP settings contract on UDS, including restart actions, observability log-tail, and settings-dependent extension routes, while keeping UDS privileged and payload-compatible with shared `api/core` handlers.

## Important Decisions
- Reuse the existing HTTP settings route shape directly in `internal/api/udsapi/routes.go` instead of creating UDS-specific handler wrappers or DTOs.
- Keep transport-parity enforcement in UDS unit/integration tests instead of importing sibling transport packages directly, so package-boundary verification remains clean.

## Learnings
- Baseline gap: `internal/api/httpapi/routes.go` already registers `registerSettingsRoutes`, but `internal/api/udsapi/routes.go` currently stops after `registerExtensionRoutes`.
- `internal/api/udsapi` already wires `core.SettingsService` and `core.SettingsRestartController` through `newHandlers`, so task scope is route registration plus parity/handler tests rather than new transport plumbing.
- `make verify` enforces package-boundary rules for test imports; UDS parity coverage must avoid importing `internal/api/httpapi` directly.
- Focused validation after implementation:
  - `go test ./internal/api/udsapi`
  - `go test -cover ./internal/api/udsapi` → `84.4%`
  - `go test -tags integration ./internal/api/udsapi -run 'TestUDSTransportSettings(ReadParityMatchesHTTP|DependencyExtensionParityMatchesHTTP|MutationsRemainPrivilegedWhenHTTPIsNonLoopback)'`
  - `make verify`

## Files / Surfaces
- `internal/api/udsapi/routes.go`
- `internal/api/udsapi/server.go`
- `internal/api/udsapi/helpers_test.go`
- `internal/api/udsapi/handlers_test.go`
- `internal/api/udsapi/server_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`
- `.compozy/tasks/settings-ui/task_07.md`
- `.compozy/tasks/settings-ui/_tasks.md`

## Errors / Corrections
- Removed an initial UDS test that imported `internal/api/httpapi` after `make verify` reported a package-boundary violation; replaced that proof with transport-level parity tests that keep the same contract coverage.

## Ready for Next Run
- Task implementation and verification are complete; only commit/handoff context should remain if further follow-up is requested.
