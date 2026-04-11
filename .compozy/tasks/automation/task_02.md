---
status: pending
title: Add automation persistence and overlay storage in globaldb
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Add automation persistence and overlay storage in globaldb

## Overview

Create the SQLite persistence model for jobs, triggers, runs, and the config-backed enabled overlays described in the hardened TechSpec. This task gives the runtime one authoritative store for CRUD operations, history queries, scope-aware uniqueness, and restart-safe fire-limit evaluation inputs.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add additive automation tables, indexes, and foreign keys to the global database schema without disturbing existing session, workspace, observe, or extension tables.
2. MUST persist scope-aware uniqueness, nullable workspace ownership for global entries, stable webhook identifiers, and separate overlay tables for config-owned `enabled/disabled` state.
3. MUST expose CRUD, list, history, and fire-limit query capabilities that later runtime and API tasks can call without embedding SQL elsewhere.
4. MUST preserve the TechSpec ownership model: config-backed definitions remain definition-owned by TOML, while runtime overlays store only operational enabled state.
</requirements>

## Subtasks
- [ ] 2.1 Extend the global database schema with automation jobs, triggers, runs, and overlay tables
- [ ] 2.2 Add store methods for create, update, delete, list, and history queries across jobs, triggers, and runs
- [ ] 2.3 Add scope-aware uniqueness and webhook identifier lookup behavior
- [ ] 2.4 Add overlay read and write behavior for config-backed enabled state
- [ ] 2.5 Add restart-safe run window queries that later dispatcher logic can use for fire-limit evaluation

## Implementation Details

Follow the TechSpec sections "Database Schema", "Development Sequencing", and "Testing Approach". Keep SQL ownership inside `internal/store/globaldb` and expose Go methods that the automation runtime can consume instead of leaking raw queries upward.

### Relevant Files
- `internal/store/globaldb/global_db.go` — Owns schema initialization and is the right place to add new automation tables
- `internal/store/globaldb/global_db_session.go` — Existing CRUD/query patterns in the global DB are the best reference for new automation accessors
- `internal/store/globaldb/global_db_test.go` — Current database tests show the repo's preferred schema and query verification style
- `internal/store/sqlite.go` — Shared SQLite setup and checkpoint behavior should remain the same for automation data

### Dependent Files
- `internal/automation/` — Scheduler, trigger engine, manager, and dispatcher will all depend on the store surface created here
- `internal/api/core/` — Later handler tasks will consume list/detail/history methods rather than issuing direct SQL
- `internal/extension/host_api.go` — Host API automation methods will eventually route through this persistence layer

### Related ADRs
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Requires shared persistence backing both schedules and triggers
- [ADR-004: Configurable Per-Job Retry with Fire Limits](adrs/adr-004.md) — Requires persisted runs to support restart-safe fire-limit evaluation

## Deliverables
- Global database schema updates for automation tables and indexes
- `internal/store/globaldb` query methods for automation CRUD, overlays, and run history
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for persistence, reopening, and query behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Opening the global DB creates all automation tables and expected uniqueness indexes
  - [ ] Creating a global job and a workspace job with the same human name succeeds, while duplicating the name inside the same scope fails
  - [ ] Webhook trigger lookup by stable `webhook_id` returns the correct trigger even when the slug text differs
  - [ ] Config-backed overlay writes update only the enabled override and do not mutate the stored definition payload
  - [ ] Run history queries filter correctly by job, trigger, status, and time window
- Integration tests:
  - [ ] Reopening the database preserves jobs, triggers, overlays, and run history
  - [ ] Persisted run-window queries still see recent runs after process restart so fire-limit evaluation data survives reboot
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Automation persistence lives entirely inside `internal/store/globaldb`
- Store queries can answer scope-aware CRUD, history, overlay, and fire-limit inputs without raw SQL escaping the package

