# Peer Review Incorporation Round 2

## Decision

The user selected full incorporation of all round 2 nits. Round 2 had no blockers and was marked `READY`.

## Incorporated Blockers

None.

## Incorporated Nits

- `N-001`: Moved store-facing conversation references to `store.NetworkConversationRef` and kept `internal/network.ConversationRef` as runtime validation state, preventing `globaldb` from importing `internal/network`.
- `N-002`: Added implementation-time migration-version verification against `globalSchemaMigrations`; version 16 remains the current planning snapshot, not a stale mandate.
- `N-003`: Documented existing SQLite foreign-key enablement through DSN/configuration and required migration/tests to assert `PRAGMA foreign_keys = ON`.
- `N-004`: Removed the undefined terminal-duplicate carve-out; exact `message_id` replay is idempotent before lifecycle handling, and all new post-terminal writes are rejected.
- `N-005`: Defined the zero-row-after-`INSERT OR IGNORE` direct-room collision case as `ErrDirectRoomCollision` with required test coverage.
- `N-006`: Defined the thread-opening rule and `thread_id` / `direct_id` / `work_id` grammar.
- `N-007`: Added best-effort, fire-and-forget, post-commit hook delivery semantics with no replay log and required deduplication guidance.
- `N-008`: Added explicit symmetric validation rules for `surface`, `thread_id`, and `direct_id`.
- `N-009`: Added foreign keys from `network_work` to `network_threads` / `network_direct_rooms` with `ON DELETE RESTRICT` to prevent dangling work rows.
- `N-010`: Added canonical `task_runs.metadata_json.network_work_id` correlation metadata and clarified it never participates in queue ownership.
- `N-011`: Defined channel-level web composer behavior as a "New public thread" affordance that creates a root thread message and redirects to thread detail.
- `N-012`: Clarified that `cy-create-tasks` must still append the required `$qa-report` + `$qa-execution` pair; Task 08 is only the verification scope definition.
- `N-013`: Added RFC 004/JCS nullable-field marshal/unmarshal contract and unit-test requirements for absent vs zero-valued fields.

## Deferred Items

None.

## Files Changed

- `.compozy/tasks/network-threads/_techspec.md`
- `.compozy/tasks/network-threads/qa/peer-review-incorporation-round2.md`
- `.codex/ledger/2026-05-04-MEMORY-network-threads.md`

## ADR Updates

No ADR update was required. The round 2 findings clarified implementation contracts and tests; they did not change the accepted architecture decisions:

- public threads and direct rooms remain separate conversation containers
- `direct` remains a surface, not a kind
- `work_id` remains lifecycle metadata, not a queue
