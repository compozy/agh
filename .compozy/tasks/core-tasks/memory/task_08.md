# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete `task_08` by exposing the task/task-run API surface through both HTTP and UDS with route parity, thin transport wiring over `internal/api/core`, and transport-specific fail-fast construction when the task service dependency is missing.

## Important Decisions
- Keep route registration inside the existing grouped router structure (`registerTaskRoutes`) instead of creating a task-specific routing style.
- Require `core.TaskService` during HTTP and UDS server construction so missing daemon wiring fails immediately rather than silently shipping partial route coverage.
- Use real `task.TaskManager` instances in HTTP and UDS integration tests so route parity is verified against shared business logic, not transport-local stubs.

## Learnings
- `task_07` already provided everything the transports needed: request/response contracts, task error mapping, and thin handler entrypoints.
- The daemon boot order already created the task runtime before transport factories; the only regression was a daemon unit-test registry stub that did not satisfy the full task store interface, which caused `bootTasks` to skip manager creation and left `deps.Tasks` nil.
- HTTP and UDS package coverage both cleared the required gate once route-registration tests, constructor precondition tests, and real round-trip lifecycle integration tests were added.

## Files / Surfaces
- `internal/api/httpapi/handlers.go`
- `internal/api/httpapi/routes.go`
- `internal/api/httpapi/server.go`
- `internal/api/httpapi/helpers_test.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/httpapi/server_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/udsapi/routes.go`
- `internal/api/udsapi/server.go`
- `internal/api/udsapi/helpers_test.go`
- `internal/api/udsapi/handlers_test.go`
- `internal/api/udsapi/server_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`

## Errors / Corrections
- After making task service injection required in both transports, daemon tests that booted real servers against `recordingRegistry` started failing with `httpapi: task service is required`; corrected by extending the registry stub to satisfy the full task store surface so `bootTasks` constructs a manager before transport boot.

## Ready for Next Run
- Task implementation is complete and fully verified.
- Evidence:
  - `go test ./internal/api/httpapi ./internal/api/udsapi`
  - `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi`
  - `go test -cover ./internal/api/httpapi ./internal/api/udsapi` (`83.2%`, `83.9%`)
  - `go test -race ./internal/daemon -run TestBootRemovesStaleSocketAndCleansOrphans -v`
  - `go test -race ./internal/daemon -run TestRunShutsDownOnInjectedSignal -timeout 20s -v`
  - `go test -race ./internal/daemon`
  - `make verify`
