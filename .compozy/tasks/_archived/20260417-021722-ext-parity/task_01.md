---
status: completed
title: "Build resource persistence kernel"
type: backend
complexity: critical
dependencies: []
---

# Task 01: Build resource persistence kernel

## Overview

Create the canonical persistence layer for the shared extensibility runtime before any family migration begins. This task establishes the raw desired-state boundary, SQLite schema, authority stamping, scope rules, optimistic concurrency, and source-scoped snapshot serialization that every later task depends on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add one canonical `internal/resources` persistence kernel that owns `MutationActor`, `RawDraft`, `RawRecord`, `SourceSnapshot`, and the raw CRUD plus snapshot interfaces described in the TechSpec "Core Interfaces" section.
2. MUST persist desired-state records in `globaldb` with deterministic schema and indexes for `resource_records` and `resource_source_state`, including explicit scope, owner, source, and version columns from the TechSpec "Data Models" section.
3. MUST enforce server-authoritative scope validation, owner/source stamping, optimistic concurrency, per-source snapshot serialization, and same-transaction source reset semantics before any record is persisted.
4. MUST reject omitted scope, stale versions, stale or non-active session nonces, oversized payloads, and snapshot overwrite attempts against daemon-owned or foreign-source records.
</requirements>

## Subtasks

- [x] 1.1 Create the `internal/resources` raw persistence types and actor-aware store boundary
- [x] 1.2 Add `resource_records` and `resource_source_state` schema support to `globaldb`
- [x] 1.3 Implement scope validation, authority stamping, CAS, and source-scoped snapshot semantics
- [x] 1.4 Add contract coverage for write conflicts, source reset, and snapshot serialization

## Implementation Details

Follow the TechSpec sections "Core Interfaces", "Data Models", "Authority and Validation Rules", and "Testing Approach". This task ends at the canonical raw store and database contract; it should not introduce typed codecs, transport handlers, or family-specific projectors yet.

### Relevant Files

- `internal/resources/` — New package for raw desired-state types, actor-aware store APIs, and snapshot application logic
- `internal/store/globaldb/global_db.go` — Canonical SQLite setup and migration entry point for the new resource tables
- `internal/store/globaldb/migrate_workspace.go` — Existing migration helper pattern that the new schema work should follow
- `internal/store/globaldb/global_db_test.go` — Real SQLite coverage for schema creation, indexes, and transactional semantics

### Dependent Files

- `internal/daemon/boot.go` — Later boot wiring depends on the canonical store existing first
- `internal/api/contract/contract.go` — Transport contracts later depend on the persisted record shape and version semantics
- `internal/extension/manager.go` — Extension publication work later depends on source-scoped snapshot semantics

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Establishes the runtime as the desired-state source of truth
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires full-snapshot reconcile and snapshot-first write semantics
- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Requires daemon-side authority stamping and scope enforcement
- [ADR-007: Use Optimistic Concurrency and Serialized Source Snapshots](adrs/adr-007.md) — Defines CAS writes, per-source snapshot sequencing, and nonce handling
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Constrains raw bytes to this kernel boundary

## Deliverables

- A new `internal/resources` raw persistence kernel with actor-aware CRUD and snapshot contracts
- `globaldb` schema support for `resource_records` and `resource_source_state`
- Server-authoritative scope, owner/source, nonce, and version enforcement in the raw write path
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for SQLite persistence, snapshot serialization, and conflict handling **(REQUIRED)**

## Tests

- Unit tests:
  - [x] creating a record with `ExpectedVersion=0` succeeds, while updating or deleting with a stale version returns a conflict
  - [x] omitted scope, `global` plus non-empty `scope_id`, and `workspace` plus empty `scope_id` are all rejected before persistence
  - [x] snapshot apply rejects a non-active `session_nonce`, a stale `source_version`, and a payload that exceeds per-record or per-call limits
  - [x] owner and source fields are stamped from `MutationActor` rather than accepted from user payload
- Integration tests:
  - [x] a snapshot for one `(source_kind, source_id)` serializes correctly and cannot overwrite a daemon-owned or foreign-source record with the same `(kind, id)`
  - [x] operator-driven source reset removes both source-owned records and the matching `resource_source_state` row in the same transaction
  - [x] bootstrapping a fresh database creates the new tables and indexes deterministically in SQLite
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- The repo has one canonical persisted desired-state store for covered extensibility kinds
- CAS, scope validation, source ownership, and snapshot sequencing are enforced before any family migration begins
