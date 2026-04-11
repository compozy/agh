# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add optional session `Space` metadata across create, persistence, session read surfaces, shared daemon contract payloads, and CLI `session new --space`.
- Keep network behavior out of `internal/session` for this task; only store and surface the metadata.
- Required validation includes create/list/stop/resume coverage and clean repo verification.

## Important Decisions
- Treat the task spec plus `_techspec.md` and ADR-005 as the source of truth: explicit opt-in only, no auto-join behavior.
- Scope excludes network join/leave wiring and prompt skill injection even though later techspec steps mention them; task 04 is metadata/persistence only.
- Plan to persist `Space` in the global session index so list/get/reconcile surfaces can expose it without ad hoc metadata reads.
- Surface `Space` in CLI human/toon outputs as well as JSON/UDS payloads so session query surfaces stay consistent across transports.
- Treat extension host session contract changes as out of scope for task 04 because the task source only requires daemon session surfaces and CLI flows.

## Learnings
- Current pre-change gap: `CreateOpts`, `SessionInfo`, `SessionMeta`, `store.SessionInfo`, `contract.CreateSessionRequest`, `contract.SessionPayload`, and `internal/cli/session.go` do not currently carry `Space`.
- The global session schema already has migration helpers in `internal/store/globaldb/migrate_workspace.go`, so adding a `sessions.space` column can follow the existing pattern instead of inventing a one-off path.
- `internal/session` can carry the opt-in cleanly without importing `internal/network`; later tasks can consume canonical session metadata instead of adding separate network-side lookup state.
- The repo verify gate includes OpenAPI freshness. After contract changes, `make verify` failed on stale `openapi/agh.json`; `make codegen` regenerated `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`, after which `make verify` passed cleanly.

## Files / Surfaces
- `internal/session/{manager.go,manager_start.go,query.go,session.go}`
- `internal/store/{types.go,meta.go}`
- `internal/store/globaldb/{global_db.go,global_db_session.go,migrate_workspace.go}`
- `internal/observe/{observer.go,reconcile.go}`
- `internal/api/{contract/contract.go,core/{handlers.go,conversions.go}}`
- `internal/cli/session.go`
- Tests across `internal/session`, `internal/store/globaldb`, `internal/api/core`, `internal/api/{httpapi,udsapi}`, and `internal/cli`
- Generated artifacts: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- Corrected hook API tests to use the real dotted hook event values such as `tool.pre_call` instead of underscored placeholders.
- Corrected stale OpenAPI artifacts by running `make codegen` before the final `make verify`.

## Ready for Next Run
- Task implementation and verification are complete. Remaining execution work is task tracking updates and the code-only local commit.
