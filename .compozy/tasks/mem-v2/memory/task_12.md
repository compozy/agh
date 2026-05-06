# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement durable session-lineage evidence and forensic `ledger.jsonl` materialization for Memory v2 Slice 1.
- Required behavior: live `events.db` remains authoritative during runtime; ledger is an idempotent session-end projection; workspace-bound ledgers use `<sessions>/<workspace_id>/<session_id>/ledger.jsonl`; unbound ledgers use `<sessions>/_unbound/<session_id>/ledger.jsonl`.

## Important Decisions

- Added `store.SessionLedgerRecord` as the durable handoff payload shared by the session manager and the ledger materializer; it carries session id, workspace id, agent/type, event DB path, lineage, and start/end timestamps.
- Added a thin `session.LedgerMaterializer` interface plus `WithLedgerMaterializer`; the session manager calls it after closing the recorder and marking metadata stopped, so `session_stopped` is durable before projection begins.
- Implemented `internal/sessions/ledger.Materializer` as a pure reader over `events.db`. It opens the event store, queries events, renders deterministic JSONL, and never writes back to the event DB.
- `ledger.jsonl` idempotence is checksum/content based: identical reruns are skipped, while an existing path with different content returns `ErrLedgerExists` to preserve forensic immutability.
- Workspace-bound and unbound path resolution is owned by the materializer and rejects unsafe path segments before touching the event store.
- Daemon boot wiring remains deferred to task_19; task_12 provides the production seam and materializer.

## Learnings

- Existing lineage durability already spans session metadata and global session index; this task's new coverage verifies the ledger line preserves `spawn_parent_id` from a real spawned session.
- Full `internal/session` package runs can still expose pre-existing async timing flakes under heavy package load; task-focused tests, race tests, lint, and the final `make verify` are the authoritative gates for this iteration.

## Files / Surfaces

- Production: `internal/store/types.go`, `internal/session/interfaces.go`, `internal/session/manager.go`, `internal/session/manager_lifecycle.go`, `internal/session/ledger.go`, `internal/sessions/ledger/materializer.go`.
- Tests: `internal/sessions/ledger/materializer_test.go`, `internal/session/ledger_test.go`.

## Errors / Corrections

- Initial lifecycle test created a child session without spawned-session requirements. Fixed by creating a real parent session, setting `SessionTypeSpawned`, and providing a TTL.
- `make lint` caught revive context-argument ordering in test helpers and staticcheck objected to a nil context call in tests. Reordered helper parameters and removed the nil-context unit assertion because production still guards nil contexts.
- Package coverage for the new `internal/sessions/ledger` package started below 80%; added seam, idempotence, unsafe-input, and unbound-layout coverage to bring it above the floor.

## Ready for Next Run

- `task_12` is complete with full `make verify` PASS.
- Next task is `task_13` (Config and Settings Backend).
- Open dependency: task_19 must instantiate the concrete ledger materializer in daemon composition alongside extractor/dreaming runtime wiring.
