---
status: completed
title: Move dream orchestration into internal/memory/consolidation
type: refactor
complexity: high
dependencies:
  - task_06
---

# Task 08: Move dream orchestration into internal/memory/consolidation

## Overview
This task removes dream consolidation orchestration from the daemon composition root and moves it into `internal/memory/consolidation`. It separates scheduling and consolidation domain logic from daemon wiring while preserving existing dream trigger behavior and integration coverage.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- Dream scheduling, gating, lock coordination, and workspace selection logic MUST move out of `internal/daemon`.
- `internal/daemon` MUST remain the composition root and SHOULD only wire the new consolidation package after the move.
- Consolidation trigger behavior, lock semantics, and workspace resolution behavior MUST remain stable.
- This task MUST run both `make verify` and `make test-integration` because it changes daemon and memory runtime behavior.
</requirements>

## Subtasks
- [x] 8.1 Create `internal/memory/consolidation` and move daemon-owned dream orchestration into it.
- [x] 8.2 Keep `internal/memory` as the owner of consolidation services and lock semantics while narrowing daemon responsibilities.
- [x] 8.3 Update daemon wiring to depend on the new package rather than owning domain behavior.
- [x] 8.4 Migrate unit and integration tests to cover the new ownership boundary.

## Implementation Details
Use the TechSpec `Component Overview`, `Integration Points`, and `Build Order` sections. Keep daemon as wiring only. Preserve existing lock, trigger, and workspace resolution semantics. Avoid introducing background lifecycle changes that are unrelated to the package move.

### Relevant Files
- `internal/daemon/dream.go` — Current home of dream orchestration that must move.
- `internal/memory/dream.go` — Current consolidation service that the new package will coordinate with.
- `internal/memory/lock.go` — Owns consolidation locking behavior used by dream orchestration.
- `internal/daemon/daemon.go` — Composition root wiring that must stay after the move.
- `internal/workspace/resolver.go` — Used during workspace resolution for dream runs.

### Dependent Files
- `internal/daemon/daemon_integration_test.go` — Covers dream-trigger behavior and daemon runtime integration.
- `internal/httpapi/httpapi_integration_test.go` — Exercises dream trigger exposure through the HTTP API.
- `internal/udsapi/udsapi_integration_test.go` — Exercises dream trigger exposure through the UDS API.
- `internal/memory/dream_test.go` — Must continue validating consolidation gating behavior.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Establishes the move into `internal/memory/consolidation`.
- [ADR-004: Use Phased Cutovers with Same-Phase Bridge Removal and Layered Verification](../adrs/adr-004.md) — Requires runtime validation for structural phases touching daemon and memory.

## Deliverables
- `internal/memory/consolidation` package owning dream orchestration responsibilities.
- Daemon updated to wire the new package without retaining domain logic.
- Updated unit and integration tests proving stable dream trigger behavior.
- `make verify` and `make test-integration` passing for the moved runtime surface.

## Tests
- Unit tests:
  - [x] Consolidation gates still block or allow runs based on time and session thresholds.
  - [x] Explicit workspace references still resolve to the same normalized workspace target.
  - [x] Lock-unavailable scenarios still return the same already-running behavior.
- Integration tests:
  - [x] Daemon integration tests still observe dream scheduling and prompt spawning behavior after the move.
  - [x] HTTP and UDS integration flows still expose dream trigger behavior correctly through memory endpoints.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `daemon` no longer owns dream domain orchestration
- Dream consolidation behavior remains stable through runtime and API paths
