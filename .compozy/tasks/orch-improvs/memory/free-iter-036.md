# Free Iteration 036 — Task execution profile transport surface

## Slice

Add HTTP, UDS, CLI, and OpenAPI surfaces for task execution profile inspect, update, and delete.

## Completed

- Added contract payloads for task execution profile read/update responses in `internal/api/contract`.
- Added shared `internal/api/core` handlers:
  - `GET /api/tasks/{id}/execution-profile`
  - `PUT /api/tasks/{id}/execution-profile`
  - `DELETE /api/tasks/{id}/execution-profile`
- Mounted the handlers on both HTTP and UDS routers and updated transport route-binding tests.
- Added OpenAPI operations and enum customization for coordinator, worker, and sandbox profile modes.
- Added CLI UDS client methods plus `agh task profile inspect|update|delete`.
- Regenerated `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and generated CLI reference pages under `packages/site/content/runtime/cli-reference/task/profile/`.
- Ran `make cli-docs`; the generator also normalized table formatting across existing CLI reference pages.

## Validation

- `go test ./internal/api/core ./internal/cli ./internal/api/spec ./internal/api/httpapi ./internal/api/udsapi -run 'TestBaseHandlersTaskExecutionProfileEndpoints|TestTaskProfileCommandsMapRequests|TestDocumentTracksRequiredFieldsAndEnums|TestRegisterExpandedTaskAndObserveOperations|TestRegisterRoutesCoversTechSpecEndpoints|TestRegisterTaskRoutesUseSharedHandlerBindings' -count=1` passed.
- `make codegen` passed.
- `make codegen-check` passed.
- `make cli-docs` passed.
- `make site-build` passed; Next generated 1047 static pages.
- `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/cli -count=1` passed.
- `go test -race ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli -count=1` passed.
- `make lint` passed with `0 issues`.
- Final `make verify` passed: Bun lint/typecheck/test passed, Vitest reported 329 files and 2088 tests passed, web build passed, `golangci-lint` reported `0 issues`, Go race gate reported `DONE 8245 tests in 70.794s`, and package boundaries passed.

## Debugging notes

- The first `make verify` failed on `gocritic hugeParam` for passing `TaskExecutionProfileRequest` / profile payloads by value through helper/client/test surfaces.
- Fixed the root cause by changing the new profile request/helper/client/stub boundaries to pass large profile values by pointer instead of suppressing lint.

## Remaining

- Review/notification transport surfaces, web UI, site concept docs, docs memory lessons, QA pair, and CodeRabbit rounds remain incomplete.
