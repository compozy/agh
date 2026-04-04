# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `internal/session` as the orchestration layer between config/agent definitions, per-session SQLite event storage, and ACP-backed runtime processes.
- Keep scope to lifecycle orchestration only: create, prompt, stop, resume, get, list, state/meta updates, and notifier fan-out.

## Important Decisions
- Keep `session/` interfaces local to the package, but add an ACP adapter so the package still composes directly with `internal/acp` without forcing tests to construct native ACP process structs.
- Use `config.Load(...WithWorkspaceRoot(workspace))` plus `config.LoadAgentDef(...)` by default so create/resume resolve provider overrides and merged MCP servers exactly as task 01/03 implemented them.
- Treat `session/` as the sole owner of per-session `events.db` writes and `meta.json` state transitions; observers only receive notifier callbacks.
- Track active sessions and pending reservations separately so `max_sessions` enforcement remains race-safe under concurrent create/resume calls.

## Learnings
- `internal/acp.Driver.Start` already implements the required resume behavior: if `ResumeSessionID` is set, it attempts `session/load` and falls back to `session/new`, returning the effective ACP session id and capabilities.
- `internal/store.OpenSessionDB` seeds sequence numbers from the on-disk DB, which makes stop/resume event continuity straightforward as long as the same `events.db` path is reopened.
- `store.SessionMeta` already matches the task’s persisted session state fields and validates against `store.SessionInfo`.
- Unit coverage for `internal/session` reached 80.0% after adding explicit branch tests for cleanup-on-failure, adapter/process helpers, constructor normalization, and prompt/stop edge cases.
- The task tracks only active sessions in-memory; stopped sessions are removed from the manager map and reconstructed from `meta.json` plus `events.db` on resume.

## Files / Surfaces
- `internal/acp/types.go`
- `internal/acp/client.go`
- `internal/store/store.go`
- `internal/store/meta.go`
- `internal/store/session_db.go`
- `internal/config/config.go`
- `internal/config/provider.go`
- `internal/config/agent.go`
- `internal/config/home.go`
- `internal/session/interfaces.go`
- `internal/session/session.go`
- `internal/session/manager.go`
- `internal/session/session_test.go`
- `internal/session/manager_test.go`
- `internal/session/additional_test.go`
- `internal/session/manager_integration_test.go`

## Errors / Corrections
- Coverage initially landed at 63.2%, then 76.2%; addressed by adding focused tests for constructor defaults/errors, create/resume cleanup, helper paths, and adapter/process wrappers instead of weakening the scope.

## Ready for Next Run
- Verification evidence:
  - `go test -race ./internal/session`
  - `go test -race -tags integration ./internal/session`
  - `go test -cover ./internal/session` → 80.0%
  - `make verify`
- Tracking files were updated after verification; keep them out of the automatic code commit because `_tasks.md` already had unrelated working-tree edits.
- Code commit created: `7463af5` (`feat: add session lifecycle manager`).
