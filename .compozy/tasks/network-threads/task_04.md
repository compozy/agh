---
status: pending
title: SQLite Conversation Schema and Store DTO Foundation
type: backend
complexity: critical
dependencies:
  - task_02
  - task_03
---

# Task 04: SQLite Conversation Schema and Store DTO Foundation

## Overview

Add the durable schema foundation for public threads, direct rooms, work metadata, participants, and revised timeline/audit storage. This task creates the store-level data model and numbered migration without yet migrating all runtime callers.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and the `agh-schema-migration` skill before editing.
- ACTIVATE `agh-schema-migration`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for table shapes, constraints, and side-table decisions.
- FOCUS ON schema, migration, DTOs, and constraints; message write orchestration is task_05.
- TESTS REQUIRED for fresh DB, migrated DB, reopen-after-restart, foreign keys, constraints, and legacy column deletion.
- NO WORKAROUNDS: do not add compatibility columns or fallback readers for old conversation rows.
</critical>

<requirements>
- MUST add the next free global schema migration version. `globalSchemaMigrations` already includes version 16 in this checkout, so use the next available version at implementation time, currently expected to be 17.
- MUST create durable `network_threads`, `network_thread_participants`, `network_direct_rooms`, and `network_work` schema.
- MUST revise `network_timeline_log` and `network_audit_log` to carry conversation fields and remove `interaction_id`.
- MUST enforce direct-room uniqueness on `(channel, peer_a, peer_b)` with lexicographic order and `CHECK(peer_a < peer_b)`.
- MUST enforce `network_work` references to exactly one existing thread or direct room.
- MUST prevent deleting a conversation container while work references it.
- MUST add store DTOs with validation for summaries, messages, direct rooms, work rows, and conversation refs.
</requirements>

## Subtasks

- [ ] 4.1 Add store DTOs and validation helpers for conversation references and summaries.
- [ ] 4.2 Add numbered migration for conversation tables, revised timeline, audit fields, and indexes.
- [ ] 4.3 Rebuild old flat `network_timeline_log` according to the TechSpec, preserving only non-conversation `greet` and `whois` rows where required.
- [ ] 4.4 Add foreign key, uniqueness, and check constraints for direct rooms and work rows.
- [ ] 4.5 Add fresh DB, migrated DB, and reopen-after-restart tests.

## Implementation Details

The migration must be transactional and must not rely on `EnsureSchema`-style reconciliation for column changes. Use `BEGIN IMMEDIATE` patterns consistent with existing global DB migrations.

### Relevant Files

- `internal/store/types.go` - conversation DTOs and validation types.
- `internal/store/globaldb/global_db.go` - migration registry and schema statements.
- `internal/store/globaldb/tx_helpers.go` - transaction helper reuse.
- `internal/store/globaldb/global_db_network_messages.go` - existing timeline schema reference.
- `internal/store/globaldb/global_db_network_audit.go` - audit schema/query updates.
- `internal/store/globaldb/global_db_network_channels.go` - channel summary dependencies.
- `internal/store/globaldb/global_db_test.go` - migration registry and reopen tests.
- `internal/store/globaldb/global_db_network_messages_test.go` - timeline migration tests.

### Dependent Files

- `internal/network/manager.go` - task_06 will consume the new store APIs.
- `internal/api/core/network.go` - task_08 will consume query DTOs.
- `web/src/generated/agh-openapi.d.ts` - task_08 codegen depends on final DTO fields.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - conversation table split.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - `network_work` boundary.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: durable store DTOs become reusable by Host API, hooks, native tools, and bridge mapping.
- Agent manageability: no public routes yet; storage must support later CLI/HTTP/UDS queries.
- Config lifecycle: no new config keys; tests must prove default config does not gain conversation controls.

### Web/Docs Impact

- Web impact: no web code in this task, but DTO names must anticipate generated contract payloads.
- Docs impact: task_16 documents final schema-visible behavior, not internal table names.

## Deliverables

- Numbered schema migration using the next free version.
- Store DTOs for conversation refs, thread summaries, direct rooms, messages, and work rows.
- Foreign-key and uniqueness constraints for conversation/work integrity.
- Migration tests for fresh and upgraded DBs.

## Tests

- Unit tests:
  - [ ] Store DTO validation rejects invalid surface/container combinations.
  - [ ] Direct-room pair normalization enforces ordered peers.
  - [ ] Work DTOs reject dangling or dual-container refs.
- Migration tests:
  - [ ] Fresh DB contains the final conversation tables and indexes.
  - [ ] Reopen-after-restart preserves schema and records the selected next migration version.
  - [ ] Old `interaction_id` column and old flat timeline indexes are absent after migration.
  - [ ] Unique direct-room constraint exists.
  - [ ] `network_work` rejects missing referenced containers.
  - [ ] Referenced thread/direct containers cannot be deleted.
  - [ ] `PRAGMA foreign_keys = 1` is asserted for cascade/restrict tests.
- Test coverage target: >=80% for touched store/globaldb packages.
- All tests must pass.

## Success Criteria

- SQLite schema can represent public threads, direct rooms, messages, and work metadata without legacy columns.
- Migration versioning is correct for the current registry.
- Later runtime/store write tasks can use a typed conversation foundation.
