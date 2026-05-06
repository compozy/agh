# Free Iteration 038 — Task run review transport surface

## Slice

Add HTTP, UDS, CLI, and OpenAPI surfaces for task run review request, list, show, and verdict submission.

## Completed

- Added shared contract payloads for task-run review requests, read/list responses, and verdict submission responses in `internal/api/contract`.
- Added shared `internal/api/core` review handlers:
  - `POST /api/task-runs/{id}/reviews`
  - `GET /api/task-runs/{id}/reviews`
  - `GET /api/tasks/{id}/reviews`
  - `GET /api/task-reviews/{id}`
  - `POST /api/task-reviews/{id}/verdict`
- Kept transport authority delegated to `task.Service`: review request creation resolves task/run ownership through `RunDetail`, review reads use `ListRunReviews`/`GetRunReview`, and verdict submission uses `RecordRunReview`.
- Mounted the handlers on both HTTP and UDS routers and extended route coverage/binding tests.
- Added OpenAPI operations and enum mappings for review policy, review status, and review outcome.
- Added CLI UDS client methods plus `agh task review request|list|show|submit`.
- Regenerated `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and generated CLI reference pages under `packages/site/content/runtime/cli-reference/task/review/`.

## Validation

- `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/cli -run 'TestBaseHandlersTaskRunReviewEndpoints|TestTaskReviewCommandsMapRequests|TestRegisterExpandedTaskAndObserveOperations|TestRegisterRoutesCoversTechSpecEndpoints|TestRegisterTaskRoutesUseSharedHandlerBindings' -count=1` passed.
- `make codegen` passed.
- `make codegen-check` passed.
- `make cli-docs` passed.
- `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/cli -count=1` passed.
- `go test -race ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli -count=1` passed.
- `make lint` passed with `0 issues`.
- Final `make verify` passed: Bun lint/typecheck/test passed, Vitest reported 329 files and 2088 tests passed, web build passed, `golangci-lint` reported `0 issues`, Go race gate reported `DONE 8254 tests in 85.160s`, and package boundaries passed.

## Debugging notes

- The first focused core handler test returned 404 because the shared API-core test fixture did not mount the new review routes. Fixed the fixture so future shared-handler tests exercise the same route shape as HTTP and UDS.
- The initial OpenAPI enum draft used older review terminal statuses (`approved`, `rejected`, `blocked`). Corrected the contract to match the current domain status model: terminal verdicts are represented as `status=recorded` plus an `outcome`.
- A standalone `make site-build` attempt stalled in `next build` with no recent `.next` writes and zero CPU usage; the stuck validation process was terminated. The final `make verify` passed afterward and included site OpenAPI/source generation plus site typecheck through the Bun workspace gate.

## Remaining

- Notification subscription/cursor transport surfaces, task context/SSE seed surfaces, review routing diagnostics if still required by the TechSpec, web UI, site concept docs, docs memory lessons, QA pair, and CodeRabbit rounds remain incomplete.
