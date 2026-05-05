# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 05: store/globaldb conversation repository writes and queries for public threads, direct rooms, work lookup, summaries, participants, audit side effects, idempotency, rollback, isolation, and raw-token redaction.
- Scope is durable store behavior only. Runtime routing/hooks/API/web surfaces are later tasks.

## Important Decisions
- Use the PRD/TechSpec/ADR design as the approved implementation source; no new conversation package and no `internal/store/globaldb -> internal/network` dependency.
- Preserve pre-existing dirty worktree changes from earlier tasks; do not revert or normalize unrelated files.
- Add store-level sentinel errors for direct-room collision and work container/terminal rejection so later runtime/API layers can map deterministic failures without string matching.
- `WriteConversationMessage` inserts the timeline row first and treats duplicate `message_id` as idempotent before participant/work/summary/audit mutation.
- Thread/direct summaries are recomputed from committed `network_timeline_log` and `network_work` rows inside the same `BEGIN IMMEDIATE` transaction as the message write.
- Missing thread/direct/work show lookups map to `ErrNetworkConversationNotFound` while still preserving the wrapped `sql.ErrNoRows`.

## Learnings
- Shared memory says Task 04 completed migration 17, final conversation tables, store DTO validation, and direct-room/work schema constraints.
- Task 04 memory says existing runtime-facing `WriteNetworkMessage` was kept compiling after the schema hard cut, but Task 05 must introduce `WriteConversationMessage` as the authoritative durable boundary.
- Pre-change signal: `task_05.md` is pending and `rg` finds no `WriteConversationMessage`, direct-room resolve, thread/direct list/show/messages, or `GetWork` implementation under `internal/store`.
- `internal/store/globaldb` already has a task immediate transaction helper pattern; Task 05 adds a network-specific immediate helper in `tx_helpers.go` using the same rollback discipline.
- Coverage evidence: `global_db_network_conversations.go` reached 80.0%, `global_db_network_audit.go` 90.9%, and `tx_helpers.go` 84.6%; full `internal/store/globaldb` package coverage is 78.1% because unrelated pre-existing globaldb modules remain below 80.
- Final coverage evidence after redaction hardening: `global_db_network_conversations.go` 80.0%, `global_db_network_audit.go` 89.5%, and `tx_helpers.go` 84.6%; full `internal/store/globaldb` package coverage is 78.1% because unrelated pre-existing globaldb modules remain below 80.
- Final verification evidence before commit: AGH test-shape scanner passed, targeted raw-token tests passed, `go test -coverprofile=/tmp/globaldb-task05.out ./internal/store/globaldb -count=1` passed, and fresh full `make verify` exited 0 after the final hardening change.
- Commit evidence: local commit `8527c7ac feat: add network conversation persistence` contains the functional Task 05 code/test changes.
- Post-commit verification evidence: `make verify` exited 0 after commit, including frontend format/lint/typecheck/tests/build, Go lint, race tests, build, and package-boundary checks.

## Files / Surfaces
- Expected touch surfaces: `internal/store/types.go`, `internal/store/store.go`, `internal/store/globaldb/global_db_network_messages.go`, `global_db_network_audit.go`, `global_db_network_channels.go`, `tx_helpers.go`, and store/globaldb tests.
- Touched so far: `internal/store/store.go`, `internal/store/types.go`, `internal/store/globaldb/tx_helpers.go`, `internal/store/globaldb/global_db_network_audit.go`, `internal/store/globaldb/global_db_network_conversations.go`, `internal/store/globaldb/global_db_network_conversation_repository_test.go`, `.compozy/tasks/network-threads/task_05.md`, `.compozy/tasks/network-threads/_tasks.md`.

## Errors / Corrections
- Added query/lifecycle negative-path tests after initial coverage showed cursor and receipt branches were under-covered.
- Fixed show/work not-found mapping to use `ErrNetworkConversationNotFound` after the query-error test exposed raw no-row behavior.
- Tightened raw-token persistence protection so message text/preview/string fields are rejected even when body JSON is already redacted; transactional audit helper now validates audit DTOs before insert.

## Ready for Next Run
- Task 05 is complete and committed locally as `8527c7ac`; tracking and workflow memory are updated, while tracking-only files remain uncommitted per task instruction.
