# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Register the expanded task HTTP surface in `internal/api/httpapi` so the documented task point-read, live, approval/triage, run-detail, and observer-backed routes exist under `/api` and behave through shared `api/core` handlers.

## Important Decisions
- Kept `routes.go` thin: all new HTTP task endpoints bind directly to existing shared `BaseHandlers` methods instead of adding transport-specific parsing or payload logic.
- Used the OpenAPI contract from `internal/api/spec.Operations()` as the parity source for the HTTP task route family, normalizing `{param}` segments to Gin `:param` paths in the parity test instead of maintaining a second task-only route list.

## Learnings
- The existing HTTP integration runtime already exercises real task manager and observer wiring, so route verification could stay transport-focused by extending that harness rather than adding new stubs or HTTP-only semantics.
- Task inbox/archive/approval assertions are stable with the real runtime because observer reads query durable task state directly; no polling or synthetic stream joins were needed.

## Files / Surfaces
- `internal/api/httpapi/routes.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/httpapi/transport_parity_integration_test.go`

## Errors / Corrections
- Initial baseline confirmed the spec/router drift by inspection because no existing HTTP test covered the new task surface yet; added parity coverage before broad verification to prevent that blind spot from recurring.

## Ready for Next Run
- Verification evidence:
  - `go test ./internal/api/httpapi`
  - `go test -tags integration ./internal/api/httpapi -coverprofile=/tmp/task09_httpapi_postcommit.cover && go tool cover -func=/tmp/task09_httpapi_postcommit.cover | tail -n 1` -> `84.6%`
  - `make web-lint`
  - `make verify` (post-commit rerun passed after the formatting hook)
- Task is complete. Local commit: `c9984ba5` (`feat: wire http task routes`).
- Tracking and workflow memory were updated for task_09 and intentionally left out of the automatic code commit.
