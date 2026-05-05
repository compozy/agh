---
status: completed
title: Work Lifecycle and Direct-Room Identity Primitives
type: backend
complexity: critical
dependencies:
  - task_02
---

# Task 03: Work Lifecycle and Direct-Room Identity Primitives

## Overview

Create the runtime primitives that make direct rooms deterministic and keep work lifecycle separate from conversation identity. This task narrows `work_id` to lifecycle-bearing work inside exactly one conversation container and provides pure direct-room identity helpers before store persistence exists.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-002, ADR-003, and `internal/CLAUDE.md`.
- ACTIVATE `nats`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for direct ID derivation and work lifecycle transitions.
- FOCUS ON primitives and state rules; durable SQLite rows are task_04/task_05.
- TESTS REQUIRED for deterministic direct IDs, collision signaling, work transitions, and idempotency.
- NO WORKAROUNDS: do not keep interaction aliases or map old names to new names.
</critical>

<requirements>
- MUST rename `InteractionState`, `Interaction`, `OpenInteraction`, `ApplyInteractionEnvelope`, and `ReasonCodeInteractionClosed` to work-specific names.
- MUST bind work lifecycle to a `store.NetworkConversationRef`-compatible container concept without importing store into runtime packages unless the TechSpec-approved package boundary supports it.
- MUST implement deterministic direct-room ID derivation from `channel + sorted(peer_a, peer_b)` using domain-separated SHA-256.
- MUST reject same-peer direct rooms.
- MUST expose collision/error states so store resolution can fail if a deterministic ID maps to another pair.
- MUST ensure exact duplicate `message_id` replay is idempotent before lifecycle handling and new post-terminal messages are rejected.
</requirements>

## Subtasks

- [x] 3.1 Rename interaction lifecycle types/functions/reason codes to work lifecycle names.
- [x] 3.2 Add a conversation reference type suitable for work binding.
- [x] 3.3 Implement pure direct-room ID derivation and pair normalization.
- [x] 3.4 Enforce lifecycle state transitions, terminal behavior, and cross-container rejection.
- [x] 3.5 Add unit tests for ID derivation, same-peer rejection, collision signaling, lifecycle transitions, and duplicate replay.

## Implementation Details

The direct-room helper should be deterministic and side-effect free. Durable unique constraints and concurrent resolve behavior are added in store tasks.

### Relevant Files

- `internal/network/lifecycle.go` - work lifecycle rename and transition rules.
- `internal/network/envelope.go` - work/container fields consumed by lifecycle code.
- `internal/network/validate.go` - direct-room ID grammar and lifecycle prerequisites.
- `internal/network/lifecycle_test.go` - transition and idempotency coverage.
- `internal/network/validate_test.go` - ID grammar and same-peer coverage.

### Dependent Files

- `internal/store/types.go` - task_04 adds durable conversation DTOs.
- `internal/store/globaldb/global_db_network_work.go` - likely task_04/task_05 destination for persisted work rows.
- `internal/network/router.go` - task_06 uses work lifecycle during routing.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - direct-room identity.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - lifecycle semantics.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: work state transitions become the source for future Host API and hook events.
- Agent manageability: no public commands yet; later surfaces must expose these semantics without queue ownership.
- Config lifecycle: no new config keys.

### Web/Docs Impact

- Web impact: no direct web change; web will display server-owned direct IDs later.
- Docs impact: task_16 must document deterministic direct-room identity and restricted visibility.

## Deliverables

- Work lifecycle naming across `internal/network`.
- Direct-room ID derivation helper with deterministic output.
- Lifecycle/container validation primitives for later store and router tasks.
- Unit tests covering lifecycle and direct-room edge cases.

## Tests

- Unit tests:
  - [x] Direct ID derivation is peer-order independent.
  - [x] Direct ID derivation is channel-scoped.
  - [x] Same-peer direct room requests fail closed.
  - [x] Work creation binds a work ID to exactly one conversation reference.
  - [x] Cross-container work continuation is rejected.
  - [x] Exact duplicate `message_id` replay returns duplicate before lifecycle handling.
  - [x] New post-terminal work messages are rejected.
- Integration tests:
  - [x] Runtime lifecycle tests compile with no remaining interaction type aliases.
- Test coverage target: >=80% for touched package.
- All tests must pass.

## Success Criteria

- Work lifecycle and conversation identity are separate in runtime code.
- Direct room identity is deterministic and ready for store-backed resolve.
- No active runtime lifecycle type uses `interaction` naming.
