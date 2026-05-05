Goal (incl. success criteria):

- Complete network-threads task_04: add SQLite conversation schema migration, store DTO validation foundation, required migration/DTO tests, tracking updates, clean make verify, and one local commit.

Constraints/Assumptions:

- Follow user/system/developer instructions, repo AGENTS.md/CLAUDE.md/internal guidance, task_04.md, \_techspec.md, ADRs, and workflow memory.
- No destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit permission.
- Must read workflow memory before code edits and update task memory as decisions/learnings/touched surfaces change.
- Must use required skills: cy-workflow-memory, cy-execute-task, cy-final-verify; task also requires agh-schema-migration, agh-code-guidelines, golang-pro, agh-test-conventions, testing-anti-patterns.
- Automatic commit is enabled only after clean verification, self-review, and tracking updates.

Key decisions:

- Use global schema migration version 17; the registry currently ends at version 16.
- Preserve the package boundary: `internal/store` / `internal/store/globaldb` must not import `internal/network`; store validation will mirror the durable grammar it needs.
- Do not keep `interaction_id` storage compatibility; existing timeline/audit methods may keep compiling names temporarily but must use the new columns.
- Migration 17 owns final conversation DDL/index creation; migration 1 must not create final conversation indexes because legacy databases may still have stale `network_audit_log` / `network_timeline_log` when migration 1 runs.

State:

- Implementation and tracking updates are complete; final post-tracking verification is next.

Done:

- Identified implementation target repo as /Users/pedronauck/Dev/compozy/agh2.
- Scanned existing ledgers for network-threads overlap; relevant prior ledgers include network-threads, network-wire-model, and network-work-primitives.
- Read workflow memory, required skills, root/internal guidance, `_techspec.md`, ADR-001/002/003, and current store/globaldb code.
- Captured pre-change signal: fresh schema still has `network_timeline_log.interaction_id`, old flat timeline indexes, and no conversation side tables.
- Added store DTOs/validation and unit tests for conversation refs, summaries, direct rooms, work rows, messages, and queries.
- Added migration 17 with final `network_threads`, `network_thread_participants`, `network_direct_rooms`, `network_work`, revised timeline/audit schema, indexes, and migration tests.
- Updated timeline/audit writes/scans to persist `surface`, `thread_id`, `direct_id`, and `work_id`.
- Fixed migration-order bug by deferring final conversation DDL/index creation from migration 1 to migration 17.
- Validation so far: `go test ./internal/store/... ./internal/network ./internal/api/core -count=1`; `go test -race ./internal/store/... ./internal/network ./internal/api/core -count=1`; new test files pass `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py`.
- First `make verify` failed on task-local lint issues (`funlen`, `lll`); fixed and reran.
- Full `make verify` then passed with `DONE 8139 tests` and `OK: all package boundaries respected`.
- Updated task_04 status/checklists, the current compact `_tasks.md` table, task memory, and shared workflow memory.

Now:

- Rerun full `make verify` on the final tracked state, self-review, and commit only task-scoped files.

Next:

- Commit after clean final verification and staged-file review.

Open questions (UNCONFIRMED if needed):

- Whether existing non-task dirty tracking changes affect final staging; current plan is to avoid touching unrelated changes.

Working set (files/ids/commands):

- Repo: /Users/pedronauck/Dev/compozy/agh2
- Task files: .compozy/tasks/network-threads/task_04.md, \_tasks.md, \_techspec.md, adrs/
- Workflow memory: .compozy/tasks/network-threads/memory/MEMORY.md, memory/task_04.md
- Code/test surfaces: internal/store/types.go, internal/store/network_conversation_types_test.go, internal/store/globaldb/global_db.go, internal/store/globaldb/global_db_network_conversations_test.go, internal/store/globaldb/global_db_network_messages.go, internal/store/globaldb/global_db_network_audit.go, internal/network/audit.go, related globaldb tests.
