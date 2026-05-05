---
status: completed
title: Conversation Persistence, Queries, Summaries, and Audit Writes
type: backend
complexity: critical
dependencies:
  - task_04
---

# Task 05: Conversation Persistence, Queries, Summaries, and Audit Writes

## Overview

Implement the store repository that writes conversation messages and derived summaries atomically. This task replaces the flat `WriteNetworkMessage` persistence path with `WriteConversationMessage` and query APIs for threads, direct rooms, messages, work, and audit data.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and the schema task before editing.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`, and `agh-cleanup-failure-paths`.
- REFERENCE TECHSPEC for transaction boundaries and summary derivation.
- FOCUS ON durable store behavior; runtime routing and hooks are later tasks.
- TESTS REQUIRED for rollback, idempotency, query isolation, concurrent direct resolve, and audit side effects.
- NO WORKAROUNDS: do not keep patching `WriteNetworkMessage`; introduce the conversation write boundary.
</critical>

<requirements>
- MUST implement `WriteConversationMessage` or equivalent repository method that writes message, summaries, participants, work transitions, and audit side effects in one SQLite transaction.
- MUST make committed message rows the source of thread/direct summaries.
- MUST make duplicate `message_id` replay idempotent before lifecycle mutation.
- MUST implement list/show/messages APIs for threads and direct rooms at the store layer.
- MUST implement direct-room resolve with concurrency-safe insert-or-return behavior and collision detection.
- MUST implement work lookup by `work_id`.
- MUST preserve raw-token redaction in persisted body/audit rows.
</requirements>

## Subtasks

- [x] 5.1 Add conversation repository methods for thread/direct list, show, messages, direct resolve, and work lookup.
- [x] 5.2 Implement same-transaction message writes for timeline, participant rows, summaries, work rows, and audit rows.
- [x] 5.3 Add direct-room resolve logic with deterministic ID, ordered peers, concurrency safety, and collision failure.
- [x] 5.4 Add query isolation so public thread queries never return direct-room messages and direct-room queries never return public thread messages.
- [x] 5.5 Add rollback/idempotency/redaction tests.

## Implementation Details

The store must own durable truth for conversation state. Runtime code should not derive persisted summaries outside the repository.

### Relevant Files

- `internal/store/types.go` - repository request/response types.
- `internal/store/globaldb/global_db_network_messages.go` - replace or split flat timeline write behavior.
- `internal/store/globaldb/global_db_network_audit.go` - audit row writes with container fields.
- `internal/store/globaldb/global_db_network_channels.go` - channel summary counts.
- `internal/store/globaldb/tx_helpers.go` - shared transaction helpers.
- `internal/store/globaldb/global_db_network_messages_test.go` - write/query behavior.
- `internal/store/globaldb/global_db_network_audit_test.go` - audit behavior.
- `internal/store/globaldb/global_db_network_channels_test.go` - channel summary behavior.

### Dependent Files

- `internal/network/manager.go` - task_06 calls the repository.
- `internal/hooks/payloads.go` - task_07 consumes committed write results.
- `internal/api/core/network.go` - task_08 exposes query paths.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - query isolation.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - work metadata.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: repository results should provide stable payloads for Host API and hook dispatch.
- Agent manageability: store query APIs must support HTTP/UDS/CLI/native tool surfaces without duplicate SQL.
- Config lifecycle: no new config keys.

### Web/Docs Impact

- Web impact: query payload shape supports task_13 routes and query keys.
- Docs impact: task_16 documents user-visible query behavior and direct-room restrictions.

## Deliverables

- Conversation repository implementation.
- Direct-room resolve and work lookup store APIs.
- Same-transaction write path for messages, summaries, participants, work, and audit rows.
- Store tests for concurrency, rollback, idempotency, visibility, and redaction.

## Tests

- Unit tests:
  - [x] Thread message write opens the thread on first valid non-duplicate message.
  - [x] Direct-room message write updates only the matching direct room.
  - [x] Participant and message counts derive from committed message rows.
  - [x] Duplicate `message_id` replay does not increment summaries or mutate work twice.
  - [x] Raw claim tokens are rejected or redacted before persistence.
- Integration tests:
  - [x] Concurrent direct resolve returns exactly one durable room for the same pair.
  - [x] Direct ID collision with a different pair fails closed.
  - [x] Transaction rollback leaves no summary or audit side effect when message insert fails.
  - [x] Thread queries exclude direct-room messages and direct queries exclude thread messages.
  - [x] Work lookup returns the bound conversation container and state.
- Test coverage target: >=80% for touched store/globaldb packages.
- All tests must pass.

## Success Criteria

- Store writes are authoritative, atomic, and queryable by conversation container.
- No active store API relies on flat peer-room timelines or `interaction_id`.
- Later runtime and API tasks can consume store methods without duplicating SQL.
