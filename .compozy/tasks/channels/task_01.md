---
status: completed
title: Introduce channel core domain types and globaldb schema
type: backend
complexity: high
dependencies: []
---

# Task 01: Introduce channel core domain types and globaldb schema

## Overview

Create the foundational domain and persistence layer for channel adapters so every later task can build against one authoritative model. This task defines the daemon-owned channel types and adds the global database tables needed for instances, secret bindings, and routes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST introduce a new `internal/channels/` package that defines the foundational channel domain types described in the TechSpec "Data Models" section, including `ChannelInstance`, `ChannelSecretBinding`, `ChannelRoute`, `RoutingKey`, `RoutingPolicy`, `InboundMessageEnvelope`, and `DeliveryEvent`.
2. MUST extend `internal/store/globaldb` with additive schema support and CRUD/query helpers for `channel_instances`, `channel_secret_bindings`, and `channel_routes`.
3. MUST validate `scope` and `workspace_id` invariants, channel status values, and stable routing-key serialization early enough that later runtime and transport layers can reuse them without duplicating shape validation.
4. SHOULD keep these types transport-agnostic so HTTP, UDS, CLI, and extension Host API layers all map to the same core model instead of creating parallel copies.
</requirements>

## Subtasks
- [x] 1.1 Create the foundational `internal/channels/` package and core structs/enums
- [x] 1.2 Add additive globaldb schema and persistence helpers for instances, bindings, routes, and ingest dedup records
- [x] 1.3 Add validation and stable key/hash helpers for scope and routing identity
- [x] 1.4 Add table-driven unit and persistence tests for the new models

## Implementation Details

Follow the TechSpec sections "Data Models", "Database schema outline", and "Impact Analysis". This task should stop at domain and persistence concerns; it should not implement session routing, delivery streaming, or transport handlers yet.

### Relevant Files
- `internal/store/globaldb/global_db.go` — Central global database access layer where new channel tables and helpers should be added
- `internal/store/globaldb/global_db_session.go` — Reference for additive session-oriented persistence helpers and query shape
- `internal/store/globaldb/global_db_test.go` — Existing global DB coverage should be extended for channel migrations and CRUD behavior
- `internal/store/globaldb/migrate_workspace.go` — Existing additive migration pattern to mirror for channel schema changes

### Dependent Files
- `internal/extension/host_api.go` — Later Host API methods will consume the channel domain types and persistence helpers
- `internal/api/contract/contract.go` — Shared transport DTOs in later tasks should map to these core types
- `internal/daemon/boot.go` — Daemon composition later depends on the channel package and store support introduced here

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Establishes the core-owned channel substrate
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — Defines the persisted instance, binding, and route models this task must create

## Deliverables
- New foundational `internal/channels/` domain package with validated core models
- Additive `internal/store/globaldb` schema and helpers for channel instances, secret bindings, routes, and ingest dedup records
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for globaldb migration and persistence behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Scope validation rejects `workspace` instances without a workspace ID and rejects `global` instances that incorrectly provide one
  - [ ] Channel status and routing-policy validation rejects unsupported enum values and malformed policy combinations
  - [ ] Stable routing-key serialization produces the same hash for repeated equivalent input values
  - [ ] Channel secret binding validation rejects empty binding names and empty vault references
  - [ ] Ingest dedup records round-trip through persistence and are excluded by TTL-based expiry filtering
- Integration tests:
  - [ ] Opening a fresh global DB applies the new channel schema and allows one `ChannelInstance` to round-trip through persistence
  - [ ] A persisted `ChannelRoute` with peer and thread dimensions survives database reopen and loads with the original session mapping intact
  - [ ] Workspace-scoped and global channel instances can coexist in the same DB without key collisions
  - [ ] Expired dedup records are excluded from lookups while unexpired records are found by idempotency key
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The repo contains one authoritative channel domain package for later tasks to build against
- Global DB can persist channel instances, bindings, and routes without breaking existing session data
