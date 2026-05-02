# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement transport-agnostic managed `HEARTBEAT.md` authoring/status services for Task 08.
- Acceptance focus: validate/write/delete/history/rollback with body-level `expected_digest`, persisted snapshots/revisions, status composition with diagnostics/config/wake/session-health, deterministic redacted errors, and no wake/session/task/lease side effects.

## Important Decisions
- Added `heartbeat.AuthoringService` / `ManagedHeartbeatAuthoringService` as the transport-neutral mutation authority for validate, write, delete, history, and rollback.
- Added `heartbeat.StatusService` / `ManagedHeartbeatStatusService` as the transport-neutral read/composition boundary for current policy, config provenance, persisted snapshots, wake state, and optional session health.
- Rollback by revision uses the append-only revision body. Rollback by `target_digest` rebuilds a canonical `HEARTBEAT.md` source from the stored snapshot frontmatter/body, then validates under current config before writing.
- Delete records a delete revision and leaves snapshots immutable; no new snapshot is created for the missing-policy state.
- Authoring errors/status errors expose deterministic redacted diagnostic codes and wrap closed sentinels for `errors.Is` consumers.

## Learnings
- Shared memory confirms Task 05 added `internal/heartbeat` resolver/config, Task 06 added Heartbeat/session-health/wake-audit storage, and Task 07 added session health read APIs. Task 08 should build on these instead of reparsing liveness or touching scheduler/task lease state.
- `agent_heartbeat_snapshots.body` stores normalized Markdown guidance, not the full source file. Digest rollback must combine `frontmatter_json` and `body` to reconstruct source; using snapshot body alone changes the digest.
- `agent_heartbeat_revisions.body` must preserve exact text for rollback. The global DB scanner now preserves text body values while continuing to trim metadata fields.

## Files / Surfaces
- Added `internal/heartbeat/authoring.go`.
- Added `internal/heartbeat/status.go`.
- Added `internal/heartbeat/authoring_status_test.go`.
- Updated `internal/store/globaldb/global_db_heartbeat.go` to preserve Heartbeat revision body text on scan.

## Errors / Corrections
- Initial rollback-by-digest implementation used `Snapshot.Body` directly and produced a different digest because snapshots store normalized guidance only. Corrected by reconstructing canonical frontmatter + body before validation/write.
- Initial global DB revision scan trimmed revision body text, which could alter rollback output. Corrected by preserving nullable text values for revision body.

## Ready for Next Run
- Verification and self-review completed for the implemented service boundary:
  - `go test ./internal/heartbeat -run 'TestManagedHeartbeat' -count=1`
  - `go test ./internal/heartbeat ./internal/store/globaldb ./internal/session -run 'TestManagedHeartbeat|TestHeartbeat|TestGlobalDBHeartbeat|TestSessionHealth' -count=1`
  - `go test ./internal/heartbeat ./internal/store/globaldb -count=1`
  - `go test -race ./internal/heartbeat ./internal/store/globaldb -run 'TestManagedHeartbeat|TestGlobalDBHeartbeat' -count=1`
  - `go test ./internal/heartbeat -cover -count=1` => 80.2%
  - `make lint` => `0 issues.`
  - `make verify` => passed; Bun tests reported 262 files / 1868 tests, Go gate reported 7621 tests, and boundaries reported `OK: all package boundaries respected`.
- Tracking updated: `task_08.md` marked completed and `_tasks.md` task 08 marked completed.
- Commit created: `8902909f` (`feat: add managed heartbeat authoring services`) with code changes only.
- Post-commit verification passed: `make verify` => Bun tests reported 262 files / 1868 tests, Go gate reported 7621 tests, and boundaries reported `OK: all package boundaries respected`.
- Remaining: no task-local implementation work.
