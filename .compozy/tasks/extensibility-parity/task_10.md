---
status: completed
title: "Migrate automation definitions to resource projection"
type: refactor
complexity: high
dependencies:
  - task_07
  - task_09
---

# Task 10: Migrate automation definitions to resource projection

## Overview

Move automation desired state into the shared resource runtime while keeping runs, history, and runtime execution state in the automation subsystem. This task proves the spec's desired-state versus operational-state split on a subsystem that already has persistence, scheduling, and trigger fan-out complexity.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST migrate `automation.job` and `automation.trigger` definitions into the canonical resource runtime and make those resource records authoritative for desired-state automation configuration.
2. MUST keep execution history, runs, runtime locks, and other operational automation state in the existing automation subsystem rather than moving them into generic resource records.
3. MUST implement automation projection with side-effect-free build and atomic apply semantics so failed reconcile preserves the previously applied scheduler and trigger runtime state.
4. MUST remove the legacy automation-definition authority in the same phase after the resource-backed projector path becomes authoritative.
</requirements>

## Subtasks

- [x] 10.1 Add codecs and typed store usage for `automation.job` and `automation.trigger`
- [x] 10.2 Replace legacy definition authority with a projector that rebuilds automation desired state from canonical records
- [x] 10.3 Keep runs, history, and other operational automation state on the existing automation runtime tables and APIs
- [x] 10.4 Add cutover coverage for projector safety, boot rebuild, and runtime-state preservation

## Implementation Details

Follow the TechSpec sections "Data Models", "Impact Analysis", "Testing Approach", and "Development Sequencing". This task should complete the automation cutover end to end, including removal of replaced definition authority, while explicitly leaving runtime execution data on automation-owned storage and APIs. Because AGH is alpha and the workspace rules reject compatibility shims, treat this as a clean cutover: do not add backfill, dual-write, or legacy compatibility paths for automation definitions.

### Relevant Files

- `internal/automation/types.go` — Desired-state automation specs need a stable typed contract for resource codecs
- `internal/automation/manager.go` — Automation runtime rebuild must switch from legacy definitions to resource-backed projection
- `internal/automation/persistence.go` — Existing automation persistence boundaries need the desired-state versus operational-state split clarified
- `internal/store/globaldb/global_db_automation.go` — Legacy automation definition storage must be replaced or demoted during cutover
- `internal/api/core/automation.go` — API behavior must keep operational automation actions family-specific after the definition cutover

### Dependent Files

- `internal/daemon/daemon.go` — Daemon startup later depends on automation runtime rebuild from canonical resource records
- `internal/api/udsapi/udsapi_integration_test.go` — UDS coverage later exercises operator writes that fan out into automation runtime changes
- `internal/automation/manager_integration_test.go` — Automation integration coverage must prove runs survive the desired-state cutover

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes the shared runtime authoritative for automation desired state
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Requires removal of legacy automation definition authority after cutover
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires automation projector and runtime evidence before the next tranche
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires automation desired state to rebuild from canonical snapshots
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Automation projection runs on the shared driver
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Automation code must consume typed resource records rather than raw payloads

## Deliverables

- Resource-backed desired-state authority for automation jobs and triggers
- Automation projector logic that rebuilds runtime configuration from canonical resource records
- Explicit preservation of automation runs and operational runtime data outside the generic resource store
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for desired-state cutover, boot rebuild, and runtime-state preservation **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `automation.job` and `automation.trigger` codecs reject invalid scope, malformed filters, and other invalid specs before persistence
  - [x] automation projector `Build` computes the next scheduler and trigger plan without mutating the live runtime
  - [x] failed projector `Apply` preserves the previously applied automation runtime state
  - [x] legacy automation definition writes are no longer authoritative after the resource-backed cutover
- Integration tests:
  - [x] an operator resource write creates, updates, and deletes automation jobs and triggers through reconcile rather than through legacy definition storage
  - [x] daemon boot rebuild reconstructs automation desired state from persisted resource records
  - [x] automation runs and history remain readable and intact after desired-state definitions move to resources
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Automation definitions are authoritative in the shared resource runtime while runs remain automation-owned
- The automation subsystem cleanly separates desired state from runtime execution state
