---
status: completed
title: Session Lineage And Spawn Metadata
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
  - task_03
  - task_05
---

# Task 12: Session Lineage And Spawn Metadata

## Overview
Add the session metadata required for safe autonomous spawning: parent/root lineage, depth, role, budget, TTL, and policy. This task records lineage and exposes it in contracts, but does not yet add agent-facing spawn commands.

<critical>
- ALWAYS READ `_techspec.md`, ADR-006, ADR-009, ADR-010, and ADR-011 before changing session metadata
- DO NOT IMPLEMENT SPAWN API IN THIS TASK - task_13 consumes this foundation
- LINEAGE MUST BE DURABLE AND QUERYABLE after daemon restart
- MANUAL USER SESSIONS REMAIN FIRST-CLASS root sessions
- TESTS REQUIRED - schema, manager creation, read models, restart, and contract generation must be covered
- NO WORKAROUNDS - do not encode lineage as opaque JSON if typed columns/read models are required
</critical>

<requirements>
- MUST add typed session metadata for parent session, root session, depth, session role/type, TTL deadline, budget, and policy where needed by the TechSpec.
- MUST persist lineage metadata in the global session catalog and expose it through session info/read DTOs.
- MUST distinguish user, system, coordinator, and spawned sessions without breaking current session creation.
- MUST keep manual sessions as root sessions with no parent.
- MUST emit typed spawn/session metadata hooks only through the taxonomy from task_03 when externally meaningful.
- MUST update generated contracts and web types if session DTOs change.
</requirements>

## Subtasks
- [x] 12.1 Extend session creation options and canonical session info with lineage/policy metadata.
- [x] 12.2 Update global session schema/store mapping for lineage, TTL, role, and budget fields.
- [x] 12.3 Update manager creation/list/get flows to preserve manual root sessions and spawned-child metadata.
- [x] 12.4 Update contracts/OpenAPI/generated web types for public session read models.
- [x] 12.5 Add tests for root sessions, child metadata, restart reads, invalid depth/policy, and DTO compatibility.
- [x] 12.6 Document task_13 integration points in completion notes.

## Implementation Details
Keep the model explicit enough for task_13 to enforce spawn caps and permission narrowing without parsing free-form metadata. Do not add the reaper yet, but make TTL/budget fields available for it.

### Relevant Files
- `internal/session/session.go` - canonical session info and type/role fields.
- `internal/session/manager.go` - `CreateOpts`, manager validation, and session catalog writes.
- `internal/store/globaldb/global_db_session*.go` - session schema and persistence.
- `internal/api/contract/sessions.go` - public session DTOs and generated contract source.
- `internal/api/udsapi/session*.go` - session list/get handler output.
- `internal/api/httpapi/*session*` - HTTP session read surfaces if present.
- `.resources/paperclip/doc/plans/2026-02-19-agent-mgmt-followup-plan.md` - reference for agent management metadata.
- `.resources/paperclip/doc/plans/2026-03-10-workspace-strategy-and-git-worktrees.md` - reference for workspace-aware session policy.
- `.resources/hermes/environments/hermes_base_env.py` - reference for environment/session lifecycle boundaries.

### Dependent Files
- `internal/session/manager.go` - task_13 adds spawn validation and reaper behavior.
- `internal/daemon/*coordinator*` - task_14 creates coordinator sessions with typed metadata.
- `web/src/generated/agh-openapi.d.ts` - regenerated if session DTOs change.

### Related ADRs
- [ADR-006: Safe Spawn With Lineage And Permission Narrowing](adrs/adr-006.md) - lineage and policy model.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - spawn/session hook payload boundaries.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual sessions remain root sessions.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - generated contract discipline.

## Deliverables
- Durable session lineage and policy metadata.
- Updated session manager creation/list/get behavior.
- Public session read models and generated contracts updated where needed.
- Unit tests with 80%+ coverage for session metadata validation/mapping **(REQUIRED)**.
- Store/API integration tests for restart and read-model behavior **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] Manual user session creation produces a root session with depth 0 and no parent.
  - [ ] Child/coordinator session creation validates parent/root/depth/TTL/budget fields.
  - [ ] Invalid depth, missing root for child, expired TTL, and malformed policy return wrapped validation errors.
  - [ ] Session DTO conversion exposes lineage without leaking internal-only policy details.
  - [ ] Existing session creation tests pass for user, dream, and system sessions.
- Integration tests:
  - [ ] Session lineage persists and reloads correctly after reopening the global DB.
  - [ ] Listing sessions can filter or display coordinator/spawned roles if the contract exposes that data.
  - [ ] Generated OpenAPI/web types and web tests pass after session DTO changes.
  - [ ] Manual session start via CLI/API remains unchanged except for additional root metadata.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Safe spawn enforcement can be implemented without schema refactors.
- Manual and autonomous sessions share one coherent session catalog.

## Completion Notes
- Added typed lineage/budget/policy model surfaces in `store.SessionLineage`, `session.CreateOpts.Lineage`, `session.Info`, and public session conversion paths.
- Persisted lineage through explicit global catalog columns for parent/root/depth/role/TTL/auto-stop plus typed JSON columns for budget and permission policy.
- Kept manual user/dream/system sessions as root rows while adding explicit `coordinator` and `spawned` session types for future autonomous behavior.
- Task 13 integration points: use `session.SessionTypeSpawned`, `session.SessionTypeCoordinator`, `store.SessionLineage`, and globaldb filters for `SessionType`, `ParentSessionID`, `RootSessionID`, and `SpawnRole`; implement spawn API/reaper enforcement there, not in this task.
