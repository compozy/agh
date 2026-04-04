# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement `internal/observe` as the `session.Notifier` consumer for global `agh.db` writes, cross-session queries, health metrics, and boot-time reconciliation.
- Deliver unit + integration tests with `-race` compatibility and >=80% package coverage, then update task tracking and create one local commit.

## Important Decisions
- Treat the PRD/techspec/ADRs as the approved design contract; do not reopen scope beyond implementation choices.
- Reuse `store.GlobalDB` query/write APIs instead of re-implementing SQL in multiple layers.
- Use `store.ReadSessionMeta` over session directories for reconciliation input and mark missing DB rows / orphaned rows through `store.GlobalDB.ReconcileSessions`.
- Resolve the effective permission mode once on `OnSessionCreated` and cache it per live session so permission audit rows can record `policy_used` without changing the notifier interface.
- Normalize recovered `meta.json` state to `stopped` during reconciliation so crash-left active sessions do not stay globally indexed as active after daemon restart.

## Learnings
- `session.Manager` already normalizes and persists raw `acp.AgentEvent` payloads, then calls `Notifier` for every prompt event and final crash error.
- Permission audit data is available directly on `acp.EventTypePermission` events via `Action`, `Resource`, and `Decision`.
- Token usage should be aggregated from `AgentEvent.Usage`, primarily on `done` events and sometimes on `usage` events, with nullable fields preserved.
- `store.GlobalDB.UpdateTokenStats` is additive, so `observe/` must aggregate only final per-turn usage (`done`) instead of every interim usage-bearing event to avoid double counting turns.

## Files / Surfaces
- `internal/session/interfaces.go`
- `internal/session/manager.go`
- `internal/store/store.go`
- `internal/store/global_db.go`
- `internal/store/schema.go`
- `internal/observe/observer.go`
- `internal/observe/health.go`
- `internal/observe/query.go`
- `internal/observe/reconcile.go`
- `internal/observe/*_test.go`
- `.compozy/tasks/agh-v2/_techspec.md`
- `.compozy/tasks/agh-v2/adrs/adr-006.md`
- `.compozy/tasks/agh-v2/adrs/adr-008.md`
- `.compozy/tasks/agh-v2/adrs/adr-009.md`

## Errors / Corrections
- Existing unrelated worktree changes were present in `.compozy/tasks/**`; keep scope tight and avoid reverting or broadening those files.
- `make verify` initially failed on staticcheck because tests passed literal `nil` contexts; replaced those with non-linting coverage paths and reran verification cleanly.

## Ready for Next Run
- Verified evidence:
  - `go test -race -cover ./internal/observe` → pass, coverage `81.6%`
  - `go test -race -tags integration ./internal/observe` → pass
  - `make verify` → pass
- Remaining follow-up lives in daemon/API wiring, not in `internal/observe`.
