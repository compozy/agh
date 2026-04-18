# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Completed the shared `internal/api/core` task surface for publish, run detail, timeline, task stream, tree, dashboard, inbox, approval, and triage flows so HTTP and UDS can share one handler/parsing/conversion layer.

## Important Decisions
- Extended `core.Observer` with `QueryTaskDashboard` and `QueryTaskInbox` and aligned `TaskService` with `task.Manager` so shared handlers can depend on manager/live/read-model surfaces without transport-specific seams.
- Kept task query parsing, actor/workspace resolution, domain error mapping, payload shaping, and task-native SSE framing inside `internal/api/core`; later transport tasks should only register routes.
- Refactored the dashboard payload converter into focused helpers and switched it to pointer input to satisfy lint (`funlen`, `gocritic`) without changing response shape.

## Learnings
- Expanding `core.Observer` also requires updating daemon test doubles, because `internal/daemon.Observer` embeds the core interface.
- Focused unit tests on draft filtering, dashboard status-breakdown conversion, nil payload branches, and new read-handler error paths were enough to push `internal/api/core` coverage from `79.3%` to the required threshold.

## Files / Surfaces
- `internal/api/core/interfaces.go`
- `internal/api/core/parsers.go`
- `internal/api/core/conversions.go`
- `internal/api/core/sse.go`
- `internal/api/core/tasks.go`
- `internal/api/core/test_helpers_test.go`
- `internal/api/core/tasks_surface_internal_test.go`
- `internal/api/core/tasks_surface_test.go`
- `internal/api/core/tasks_surface_integration_test.go`
- `internal/api/testutil/apitest.go`
- `internal/daemon/daemon_test.go`

## Errors / Corrections
- `make verify` initially failed because daemon `fakeObserver` no longer satisfied the expanded observer interface; added stub `QueryTaskDashboard` / `QueryTaskInbox` methods.
- `golangci-lint` then failed on `funlen`, `lll`, and `gocritic` in the new shared handler work; fixed by helper extraction, line wrapping, and pointer-based dashboard conversion.
- Fresh verification evidence:
  - `go test ./internal/api/core`
  - `go test -tags integration ./internal/api/core`
  - `go test ./internal/api/core -coverprofile=/tmp/task08_api_core.cover && go tool cover -func=/tmp/task08_api_core.cover | tail -n 1` => `80.0%`
  - `make verify` => pass
  - local commit `5bc345ee` (`feat: add shared task api handlers`) created after code-only staging; `make verify` rerun on committed state => pass

## Ready for Next Run
- Task_09 and task_10 should wire the existing shared handlers into HTTP and UDS route tables and add parity coverage, not duplicate task parser/conversion/SSE logic in transport packages.
