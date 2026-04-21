---
status: pending
title: Global Session Index Migration and Legacy Provider Repair
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 03: Global Session Index Migration and Legacy Provider Repair

## Overview

Extend the global session index to persist provider state and converge old local session metadata to the new model exactly once. This task is the storage-risk boundary for the feature: the SQLite schema must gain `sessions.provider`, global register/scan/reconcile paths must preserve it, and inactive legacy sessions with blank provider metadata must be repaired deterministically before resume or reconcile continues.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and task_02 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Data Models", "Testing Approach", "Build Order", and "Known Risks"
- THIS IS THE ONLY ALLOWED LEGACY HANDLING FOR THE FEATURE - implement the one-time repair from ADR-005 and do not introduce perpetual fallback semantics
- MIGRATIONS MUST BE IDEMPOTENT - both in-place `ALTER TABLE` and copy-style rebuild paths must preserve `provider`
- GLOBAL INDEX AND SESSION META MUST CONVERGE - do not leave repaired sessions blank in one storage layer and populated in another
- GREENFIELD: nao aceitar compat code infinito; metadata antiga deve convergir para o novo estado e parar de ser especial
</critical>

<requirements>
- MUST add `provider TEXT NOT NULL DEFAULT ''` to the global SQLite `sessions` table
- MUST extend `migrateSessionColumns` and any copy-style `sessions` table rebuild path to create and populate the new column
- MUST update global register, scan, list, get, and reconcile paths to read and write `provider`
- MUST implement the one-time repair rule for inactive legacy metadata with blank provider state before resume or global reconcile proceeds
- MUST persist the repaired provider immediately and fail explicitly if the stored agent or provider can no longer be resolved
- MUST avoid any ongoing fallback from blank provider after repair; repaired sessions should become ordinary sessions
- SHOULD log successful and failed repair attempts with the provider and phase fields described in the TechSpec
</requirements>

## Subtasks
- [ ] 3.1 Add `sessions.provider` to the global DB schema migration paths
- [ ] 3.2 Update global session register/scan/reconcile helpers to round-trip provider
- [ ] 3.3 Implement one-time blank-provider repair for inactive legacy session metadata
- [ ] 3.4 Fail explicitly when legacy repair cannot resolve the stored agent/provider anymore
- [ ] 3.5 Add migration and repair coverage for idempotence, reconcile, and explicit failure paths

## Implementation Details

See TechSpec "Data Models", "Testing Approach", and ADR-005. The storage invariant after this task is simple: every live or queryable session has a concrete provider in session metadata and in the global `sessions` index, except for the narrow transient moment before a legacy repair is persisted.

### Relevant Files
- `internal/store/globaldb/global_db.go` - global DB initialization and migration wiring
- `internal/store/globaldb/migrate_workspace.go` - session column migration path that must add `provider`
- `internal/store/globaldb/global_db_session.go` - session register, scan, and reconcile helpers
- `internal/store/globaldb/global_db_session_test.go` - natural home for provider-aware register/scan tests
- `internal/store/globaldb/global_db_test.go` - migration and index behavior coverage
- `internal/store/globaldb/global_db_extra_test.go` - copy-style or edge-case migration coverage
- `internal/session/resume_repair.go` - legacy metadata repair logic
- `internal/session/resume_repair_test.go` - targeted repair and failure tests

### Dependent Files
- `internal/api/core/conversions.go` - task_04 depends on the global read model exposing provider
- `internal/cli/session.go` - task_04 will show provider in list/detail output sourced from the global index
- `web/src/generated/agh-openapi.d.ts` - later generated surfaces assume provider is present in session payloads consistently
- `.compozy/tasks/session-driver-override/task_08.md` - QA execution must prove migration and repair behavior end to end

### Related ADRs
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) - defines persisted-provider semantics
- [ADR-005: Migrate Session Provider State In Place And Repair Legacy Metadata Once](adrs/adr-005.md) - governs the migration and one-time repair behavior

## Deliverables
- Global DB schema migration that adds and preserves `sessions.provider`
- Provider-aware global register, scan, and reconcile behavior
- One-time legacy blank-provider repair with immediate persistence
- Explicit failure path when repair cannot resolve the stored agent/provider **(REQUIRED)**
- Migration and repair coverage for idempotence and reconcile behavior **(REQUIRED)**
- Test coverage >=80% for the touched `internal/store/globaldb` and `internal/session` package(s) **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `migrateSessionColumns` adds `provider` idempotently
  - [ ] Copy-style migrations preserve `provider` values when rebuilding `sessions`
  - [ ] `scanSessionInfo` reads provider from the global index
  - [ ] `registerSession` upserts provider correctly
  - [ ] Legacy blank-provider repair persists the resolved provider exactly once
  - [ ] Repair fails explicitly when the stored agent or provider can no longer be resolved
- Integration tests:
  - [ ] Opening an existing global DB migrates `sessions.provider` without dropping data
  - [ ] Reconcile persists repaired providers into the global index and stops leaving blanks behind
  - [ ] Resuming a legacy session after repair uses the persisted provider deterministically
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The global session index persists provider for every post-feature session
- Legacy blank-provider sessions converge once and then behave like ordinary sessions
