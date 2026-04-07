# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the `/api/workspaces` REST surface to `internal/httpapi`, inject a resolver-backed workspace service into HTTP handlers, and move the session HTTP contract to the TechSpec shape: `workspace` for registered refs, `workspace_path` for explicit paths, and `GET /api/sessions?workspace=` filtering.
- Keep the change greenfield-clean: no compatibility shim for the old `{"workspace":"/path"}` request shape.

## Important Decisions
- The HTTP layer will consume a workspace interface defined in `internal/httpapi`, implemented by the resolver, so handlers stay testable without importing daemon wiring details.
- Session listing will support `?workspace=` by resolving workspace IDs/names through the injected workspace service and then filtering the returned session infos by `WorkspaceID`.
- Session JSON now exposes `workspace_id` and `workspace_path` explicitly instead of overloading `workspace` with a path response field.
- Workspace detail routes use `Resolve(...)` for live agent/skill/session summaries, while list/filter/update/delete paths use `Get(...)` so registrations can still be addressed without forcing a fresh filesystem resolve.

## Learnings
- Current handler tests and integration tests still assume the old create payload (`workspace` carrying a filesystem path) and will need coordinated fixture updates.
- `SessionInfo` already carries both `WorkspaceID` and resolver-derived `Workspace`, so the HTTP layer can expose both without changing session runtime models.
- Integration verification confirmed that the HTTP layer surfaces canonical resolver paths (`/private/...` on macOS temp dirs), not necessarily the raw path alias submitted by the client.
- The daemon runtime needed both the narrow `workspace.WorkspaceResolver` interface and the concrete resolver instance so the HTTP factory could consume CRUD methods without widening other runtime dependencies.

## Files / Surfaces
- `internal/httpapi/server.go`
- `internal/httpapi/workspaces.go`
- `internal/httpapi/sessions.go`
- `internal/httpapi/stream.go`
- `internal/httpapi/helpers_test.go`
- `internal/httpapi/handlers_test.go`
- `internal/httpapi/handlers_error_test.go`
- `internal/httpapi/httpapi_integration_test.go`
- `internal/daemon/daemon.go`
- `internal/store/store.go`
- `internal/store/global_db.go`
- `internal/store/global_db_test.go`

## Errors / Corrections
- Initial daemon compilation failed because `httpapi` now needs CRUD methods that are not present on the narrow `workspace.WorkspaceResolver` interface; corrected by threading the concrete resolver through `internal/daemon` only for the HTTP factory while keeping the narrower interface for other runtime consumers.
- Initial HTTP integration assertions assumed the raw temp-dir alias would round-trip unchanged; corrected after verification showed the resolver canonicalizes `workspace_path` values.

## Ready for Next Run
- Implementation and verification are complete. Remaining closeout is task tracking updates, code-only commit creation, and final handoff.
