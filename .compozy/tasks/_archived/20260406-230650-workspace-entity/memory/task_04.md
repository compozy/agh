# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Refactor `session.Manager` create/resume to require `workspace.WorkspaceResolver`, resolve startup state from `workspace.ResolvedWorkspace`, and persist `WorkspaceID` as the authoritative session metadata key.

## Important Decisions
- `session.CreateOpts` now distinguishes registered workspace references (`Workspace`) from explicit filesystem roots that should auto-register (`WorkspacePath`).
- `session.Session.WorkspaceID` and `store.SessionMeta.WorkspaceID` are the durable persisted fields; runtime `Session.Workspace` remains as a resolver-derived root path to avoid widening downstream API churn inside task_04.
- Session query/status paths now resolve stopped-session workspace roots best-effort from the resolver using stored `WorkspaceID`.

## Learnings
- Session tests needed a resolver fake that can answer both `Resolve` and `ResolveOrRegister`; older config/agent loader seams were no longer sufficient once `ResolvedWorkspace` became the source of truth.
- `internal/observe` reconciliation can convert normalized `store.SessionMeta` directly into `store.SessionInfo` because those structs now share the same workspace-ID-backed shape.
- Downstream daemon, HTTP, UDS, and integration tests needed coordinated updates once session creation stopped accepting an implicit current-directory fallback.

## Files / Surfaces
- `internal/session/manager.go`
- `internal/session/session.go`
- `internal/session/query.go`
- `internal/session/manager_test.go`
- `internal/session/additional_test.go`
- `internal/session/query_test.go`
- `internal/session/transcript_test.go`
- `internal/session/manager_integration_test.go`
- `internal/session/manager_stop_integration_test.go`
- `internal/store/store.go`
- `internal/store/meta_test.go`
- `internal/store/store_helpers_test.go`
- `internal/observe/reconcile.go`
- `internal/observe/helpers_test.go`
- `internal/observe/reconcile_test.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/httpapi/sessions.go`
- `internal/httpapi/handlers_test.go`
- `internal/httpapi/httpapi_integration_test.go`
- `internal/udsapi/handlers.go`
- `internal/udsapi/handlers_test.go`
- `internal/udsapi/udsapi_integration_test.go`
- `internal/cli/cli_integration_test.go`

## Errors / Corrections
- First `make verify` failed because `internal/observe` tests still wrote the removed `store.SessionMeta.Workspace` field, `internal/observe/reconcile.go` used a verbose struct literal that Staticcheck rejected, and `internal/session/manager_test.go` still had an unused helper. All three issues were fixed before the final verification pass.

## Ready for Next Run
- Task 08 can rely on resolver-derived workspace roots already flowing through session create/resume when wiring ACP additional directories.
- Tasks 10-12 should revisit the external session creation contract so callers can pass registered workspace IDs/names explicitly instead of the current compatibility mapping of API `workspace` input into `WorkspacePath`.
