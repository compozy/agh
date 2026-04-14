---
status: pending
title: "Persist task dependencies, audit trail, and idempotency"
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Persist task dependencies, audit trail, and idempotency

## Overview
Extend the storage layer so the task domain can safely represent bounded dependency graphs, immutable audit history, and multi-writer idempotency. This task closes the persistence gaps that make cross-surface coordination and replay-safe writes possible.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. The store MUST persist dependency edges, task events, and idempotency records needed by the TechSpec and the ADR set.
2. Dependency writes MUST enforce bounded graph rules and cycle detection transactionally at write time rather than via asynchronous cleanup.
3. Audit and idempotency persistence MUST preserve immutable technical origin data and apply payload caps before rows are stored.
</requirements>

## Subtasks
- [ ] 3.1 Add schema and storage methods for dependency edges between tasks.
- [ ] 3.2 Add immutable task-event persistence for lifecycle and audit records.
- [ ] 3.3 Add idempotency-key persistence and lookup for multi-writer ingress surfaces.
- [ ] 3.4 Implement transactional dependency-edge writes with cycle detection and graph-limit enforcement.
- [ ] 3.5 Add query helpers needed for dependency inspection, audit reads, and duplicate-ingress detection.

## Implementation Details
Use the TechSpec "Data Models", "Run Authority and Attachment Rules", "Guardrails and Limits", and "Monitoring and Observability" sections. The transaction shape called out in the revised spec matters here: dependency-edge creation should use the documented transactional boundary instead of optimistic post-write repair.

### Relevant Files
- `internal/store/globaldb/global_db.go` — Shared transaction and DB helper patterns.
- `internal/store/globaldb/global_db_network_audit.go` — Reference for immutable audit/event persistence patterns.
- `internal/store/globaldb/global_db_automation.go` — Reference for idempotency-adjacent list/read patterns from a multi-writer subsystem.
- `internal/task/` — Source of graph limits, payload caps, and domain validation rules.
- `.compozy/tasks/core-tasks/_techspec.md` — Source of the required transactional and guardrail rules.

### Dependent Files
- `internal/task` manager implementation — Will depend on dependency, event, and idempotency reads/writes.
- `internal/network/` — Will consume idempotency and audit capabilities for network-originated writes.
- `internal/observe/` — Will consume persisted task events for projections and health views.

### Related ADRs
- [ADR-002: Support Global and Workspace Task Scope with Explicit Hierarchy and Bounded Dependencies](../adrs/adr-002.md) — Governs bounded dependency graphs and cycle prevention.
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Drives immutable lifecycle event recording and manager-owned transitions.
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Influences channel-related audit semantics.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Requires immutable origin metadata in persisted audit records.

## Deliverables
- Dependency-edge schema and store methods with transactional cycle checks.
- Immutable task-event and idempotency persistence.
- Read/query helpers for dependency inspection, audit history, and duplicate detection.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for graph writes and multi-writer idempotency **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify dependency-edge validation rejects self-dependency, duplicate edges, and limit overflows before persistence.
  - [ ] Verify event payload writes reject oversize payloads and preserve immutable actor/origin metadata.
  - [ ] Verify idempotency lookups return the original result for duplicate keys from the same origin scope.
- Integration tests:
  - [ ] Verify creating an edge that would introduce a cycle fails transactionally and does not leave partial graph state behind.
  - [ ] Verify two duplicate non-human writes with the same idempotency key are deduplicated against the same persisted record.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The task store can represent bounded dependency graphs, audit records, and idempotent writes safely
- Downstream manager and ingress work can rely on transactionally correct dependency and audit persistence
