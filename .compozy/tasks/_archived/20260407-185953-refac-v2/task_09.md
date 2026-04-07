---
status: completed
title: Narrow consumer interfaces and remove transitional bridges
type: refactor
complexity: critical
dependencies:
  - task_07
  - task_08
---

# Task 09: Narrow consumer interfaces and remove transitional bridges

## Overview
This final convergence task aligns the remaining runtime consumers with the new package graph, removes same-phase migration bridges, and deletes validated dead compatibility code. It is intentionally last because it closes the architectural loop after the major ownership moves have landed.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- Remaining consumers in `session`, `observe`, `workspace`, `daemon`, `cli`, and related packages MUST depend on the narrowed boundaries introduced by the earlier tasks.
- Broad interfaces left behind from pre-refactor ownership, especially in persistence and API boundaries, MUST be narrowed or removed where the new structure makes that possible.
- Transitional bridges, re-exports, and compatibility shims introduced during earlier phases MUST be removed before this task completes.
- This task MUST run the full required verification gates, including `make verify` and the integration suites required by the TechSpec.
</requirements>

## Subtasks
- [x] 9.1 Update remaining consumers to the final package graph and narrowed interface surfaces.
- [x] 9.2 Remove temporary bridges and dead compatibility code left from earlier package moves.
- [x] 9.3 Delete validated dead files and stale helpers that are no longer justified by the final architecture.
- [x] 9.4 Run the final verification sweep required for the completed refactor graph.

## Implementation Details
Use the TechSpec `Impact Analysis`, `Build Order`, and `Known Risks` sections. This task exists to finish the migration, not to start new architecture work. Keep deletions evidence-based and limited to code made obsolete by the approved refactor sequence.

### Relevant Files
- `internal/store/store.go` — Shared interface surface that should be narrowed after the persistence split.
- `internal/session/manager.go` — Major consumer of store and transcript boundaries that may still need narrowing.
- `internal/observe/observer.go` — Consumer of global persistence and runtime interfaces that should align with the new structure.
- `internal/workspace/resolver.go` — Consumer of workspace persistence boundaries that may still carry broad assumptions.
- `internal/daemon/daemon.go` — Final composition root wiring after all moves are complete.

### Dependent Files
- `internal/cli/client.go` — Must use the final API and persistence-adjacent surfaces without transitional shims.
- `internal/httpapi/handlers_test.go` or migrated equivalents — Must stay green after bridge removal.
- `internal/udsapi/handlers_test.go` or migrated equivalents — Must stay green after bridge removal.
- `internal/daemon/daemon_integration_test.go` — Final runtime verification after convergence.
- `internal/cli/cli_integration_test.go` — Final verification that client-facing behavior remains stable.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Defines the final architecture this task converges on.
- [ADR-003: Split Persistence into store/sessiondb and store/globaldb](../adrs/adr-003.md) — Drives interface narrowing in persistence consumers.
- [ADR-004: Use Phased Cutovers with Same-Phase Bridge Removal and Layered Verification](../adrs/adr-004.md) — Requires removal of temporary bridges and final verification gates.

## Deliverables
- Remaining runtime consumers aligned to the final package graph and narrowed interfaces.
- Transitional bridges and dead compatibility code removed.
- Final cleanup of validated dead files introduced by the migration.
- `make verify` and required integration suites passing across the completed refactor graph.

## Tests
- Unit tests:
  - [ ] Consumer packages still compile and satisfy narrowed interfaces without transitional shims.
  - [ ] Final bridge removal does not change status mapping, DTO shapes, or transcript access behavior.
  - [ ] Dead code cleanup does not remove any still-referenced runtime paths.
- Integration tests:
  - [ ] Full runtime flows for CLI, HTTP, UDS, daemon boot, session lifecycle, and workspace operations still pass after bridge removal.
  - [ ] Memory consolidation and transcript retrieval still pass through their public API surfaces after final convergence.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- No transitional bridges remain from the refac-v2 migration
- The runtime depends on the final approved package graph without broad legacy boundaries
