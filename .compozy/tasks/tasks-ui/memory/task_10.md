# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Register the full task surface in UDS so it matches HTTP/spec coverage for point reads, live routes, aggregate observer routes, and task mutations.
- Prove parity with UDS route inventory, UDS integration coverage, and transport parity tests.

## Important Decisions
- Use normalized `api/spec.Operations()` route inventories for both documented HTTP and documented UDS task surfaces instead of importing the HTTP transport package into UDS parity tests.
- Keep the UDS parity work scoped to route registration and transport tests; shared parsing, payload shaping, and SSE framing stay in `internal/api/core`.

## Learnings
- `internal/api/udsapi/routes.go` was still exposing the older task surface; it was missing publish/live/approval/triage/dashboard/inbox/task-run detail routes that HTTP already registers.
- The shared spec had `streamTask` documented as HTTP-only even though UDS serves the same SSE route, so the task parity fix required updating the shared transport metadata before regenerating OpenAPI.

## Files / Surfaces
- `internal/api/spec/spec.go`
- `internal/api/udsapi/routes.go`
- `internal/api/udsapi/handlers_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`
- `openapi/agh.json`

## Errors / Corrections
- An initial parity-test implementation imported `internal/api/httpapi` from `internal/api/udsapi`, which violated the repository transport-boundary rule and failed `make verify`; the fix was to compare UDS router output against spec-normalized task routes without cross-package transport imports.
- An initial targeted spec patch matched the wrong `Transports` block because `spec.go` has repeated transport lists; the final correction explicitly updated only the `streamTask` operation and preserved unrelated transport metadata.

## Ready for Next Run
- Task implementation is complete and verified with `go test ./internal/api/udsapi`, targeted UDS/HTTP integration parity checks, `make web-lint`, `make web-typecheck`, and `make verify`.
