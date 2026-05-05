# Peer Review Incorporation Round 1

## Decision

The user selected full incorporation of all round 1 blockers and nits.

## Incorporated Blockers

- `B-001`: Added explicit extensibility, agent-manageability, extension Host API, hooks, native tools, MCP/tooling, bridge SDK, bundles/registries, skills/capabilities, and config lifecycle analysis.
- `B-002`: Added concrete SQLite migration version, final DDL, migration transaction expectations, rebuild/copy/drop behavior, delete targets, and migration tests.
- `B-003`: Added concrete Go definitions for wire models, store summaries, query DTOs, `ConversationStore`, `Work`, `WorkState`, validators, reason codes, and old-to-new symbol renames.
- `B-004`: Added deterministic direct-room resolution algorithm, direct ID hashing shape, uniqueness constraints, and concurrent resolve semantics.
- `B-005`: Added durable `network_work` table and authoritative cross-container work binding rules.
- `B-006`: Added explicit `work_id` / `task_runs` relationship: `work_id` is network-level lifecycle metadata, while `task_runs` remains the only durable queue.
- `B-007`: Added same-transaction `BEGIN IMMEDIATE` write strategy for timeline rows, work binding, participant updates, summaries, and audit rows.
- `B-008`: Added RFC 004 trust integration and signed-field set for `surface`, `thread_id`, `direct_id`, and `work_id`.

## Incorporated Nits

- `N-001`: Added explicit TanStack Router file/route map for `/network` threads and direct rooms.
- `N-002`: Added CLI commands, flags, deleted flags, and JSON/JSONL output shapes.
- `N-003`: Stated `ConversationStore` is consumed in `internal/network`, implemented in `internal/store/globaldb`, and no `internal/conversation` package is introduced.
- `N-004`: Added explicit old-to-new Go symbol rename table.
- `N-005`: Aligned MVP task 01-08 with implementation build order.
- `N-006`: Stated archived `.compozy/tasks/_archived/*` artifacts may retain old terminology.
- `N-007`: Stated expanding `direct_room` beyond two peers is a future schema migration trigger.
- `N-008`: Added metric names, units, labels, and cardinality rules.
- `N-009`: Added 80% touched-package coverage floor and Linux-Race parity expectations.
- `N-010`: Added bundled `agh-network` skill rewrite outline and example envelopes.

## Deferred Items

None.

## Files Changed

- `.compozy/tasks/network-threads/_techspec.md`
- `.compozy/tasks/network-threads/adrs/adr-001.md`
- `.compozy/tasks/network-threads/adrs/adr-002.md`
- `.compozy/tasks/network-threads/adrs/adr-003.md`
- `.codex/ledger/2026-05-04-MEMORY-network-threads.md`
- `.compozy/tasks/network-threads/qa/peer-review-incorporation-round1.md`

## Research Inputs Applied

- Current `globalSchemaMigrations` ends at version 15, so the spec names version 16 unless another migration lands first.
- Current network persistence uses flat `network_timeline_log` rows with `interaction_id`, so the spec replaces the write path with a same-transaction conversation write.
- Existing store patterns use rebuild migrations for SQLite table shape changes and `BEGIN IMMEDIATE` helpers for multi-step write transactions.
- Current HTTP/UDS/CLI surfaces expose flat channel and peer message paths, so the spec deletes those as primary conversation paths and replaces them with thread/direct endpoints.
- Current native tools expose network status/channels/inbox/peers/send, so the spec updates `network_send` and adds conversation-specific tools.
- Current hook taxonomy has no `network` family, so the spec adds async observation-only network hook events.
- RFC 004 signs the full envelope excluding only `proof.sig`, so the spec makes the new conversation fields explicit in the signed content.
