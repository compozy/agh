# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Mirror the HTTP workspace CRUD/resolve routes and updated session create/list contract into `internal/udsapi`, including workspace resolver injection and payload parity (`workspace` ref vs `workspace_path`, response `workspace_id` + `workspace_path`).
- Keep scope tight to UDS plus any small CLI transport/test updates required by the new contract.

## Important Decisions
- Reuse the HTTP workspace/session helper logic where possible so UDS validation and error mapping stay aligned instead of drifting.
- Treat the HTTP implementation from task 10 as the source-of-truth contract for UDS route names, payloads, and status codes.
- Introduce a small shared `internal/apisupport` package for session/workspace transport validation and status mapping instead of duplicating helpers across `internal/httpapi` and `internal/udsapi`.
- Fix the daemon boot regression at the composition root by passing `udsapi.WithWorkspaceResolver(deps.WorkspaceService)` into the default UDS factory; do not weaken daemon tests or make the UDS constructor optional again.

## Learnings
- UDS now mirrors the HTTP workspace surface: `/api/workspaces` CRUD/resolve routes exist, `POST /api/sessions` accepts either a registered `workspace` ref or an explicit `workspace_path`, and `GET /api/sessions` can filter by workspace ref through resolver lookup.
- UDS and CLI payloads now treat `workspace_id` and `workspace_path` as distinct response fields; human-facing CLI output should prefer the path when present and fall back to the ID.
- The first full `make verify` exposed a real production-path gap: `internal/daemon` was still creating the real UDS server without a workspace resolver, which broke daemon boot tests until the default UDS factory was updated.

## Files / Surfaces
- `internal/apisupport/session_workspace.go`
- `internal/udsapi/server.go`
- `internal/udsapi/routes.go`
- `internal/udsapi/handlers.go`
- `internal/udsapi/workspaces.go`
- `internal/udsapi/helpers_test.go`
- `internal/udsapi/handlers_test.go`
- `internal/udsapi/handlers_error_test.go`
- `internal/cli/client.go`
- `internal/cli/session.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`

## Errors / Corrections
- Initial full verification failed in `internal/daemon` with `daemon: create uds server: udsapi: workspace resolver is required`; the root cause was missing workspace-service injection in the default UDS factory, fixed in `internal/daemon/daemon.go` with a regression assertion in `internal/daemon/daemon_test.go`.

## Ready for Next Run
- Task implementation and verification are complete. Remaining execution step is the local code-only commit after tracking updates.
