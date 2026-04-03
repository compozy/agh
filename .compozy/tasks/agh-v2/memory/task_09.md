# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement `internal/httpapi` as the real TCP HTTP/SSE transport for the daemon, matching the task 09 techspec endpoints and streaming contracts.
- Verification complete: `go test -race ./internal/httpapi ./internal/daemon`, `go test -race -tags integration ./internal/httpapi ./internal/daemon`, `go test -cover ./internal/httpapi` (`80.3%`), and `make verify`.

## Important Decisions
- Kept `internal/httpapi` separate from `internal/udsapi` instead of extracting a shared transport package in this task; the duplicate surface is limited and keeps task 09 scoped to HTTP behavior.
- Made the HTTP prompt endpoint accept both the current `{message}` payload and AI SDK-style `{messages:[...]}` input, extracting the latest user text for forward compatibility.
- Mapped prompt SSE to AI SDK UI message stream parts (`start`, `text-*`, `reasoning-*`, `tool-*`, `finish`, `[DONE]`) while preserving native SSE event names like `agent_message`, `tool_call`, `tool_result`, `done` for diagnostics/tests.
- Switched `internal/daemon` from the no-op HTTP factory to the real `httpapi.New(...)` factory and updated daemon tests to allocate a free TCP port per test instead of using a fixed port.

## Learnings
- Integration-only test helpers must live behind an `integration` build tag or golangci-lint will flag them as unused during the unit-test build.
- The highest coverage gains came from branch-focused tests on prompt translation and helper/error paths, not from adding more end-to-end cases.

## Files / Surfaces
- `internal/httpapi/server.go`
- `internal/httpapi/sessions.go`
- `internal/httpapi/prompt.go`
- `internal/httpapi/stream.go`
- `internal/httpapi/agents.go`
- `internal/httpapi/observe.go`
- `internal/httpapi/daemon.go`
- `internal/httpapi/helpers_test.go`
- `internal/httpapi/helpers_integration_test.go`
- `internal/httpapi/handlers_test.go`
- `internal/httpapi/handlers_error_test.go`
- `internal/httpapi/stream_helpers_test.go`
- `internal/httpapi/server_test.go`
- `internal/httpapi/httpapi_integration_test.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`

## Errors / Corrections
- Initial `make verify` failed because three HTTP integration helpers were defined in the always-built unit test file; moved them into `helpers_integration_test.go` with the `integration` build tag instead of suppressing lint.
- Initial `go test -cover ./internal/httpapi` reported `76.7%`; added targeted prompt/observe/helper tests to bring the package to `80.3%`.

## Ready for Next Run
- Local code-only commit created as `09a6cba` (`feat: add http api server`).
- Task tracking and workflow memory were updated but intentionally left unstaged per workspace rules.
