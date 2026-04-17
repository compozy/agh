---
status: completed
title: "Migrate hook bindings and wire tool/permission hooks"
type: refactor
complexity: critical
dependencies:
  - task_03
  - task_04
  - task_05
---

# Task 07: Migrate hook bindings and wire tool/permission hooks

## Overview

Move hook binding authority into the shared resource runtime and close the currently shipped hook gaps around `tool.*` and `permission.*`. The taxonomy already defines those events; the missing piece is the authoritative binding and runtime wiring path. This is the first real family cutover and the first tranche that proves the runtime, grant model, and projector-swap pattern against concrete user-visible extensibility behavior.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST migrate `hook.binding` desired state into the canonical resource runtime and make resource-backed bindings authoritative for hook dispatch.
2. MUST replace the hand-enumerated dispatch matrix in `internal/daemon/hooks_bridge.go` and any remaining manual session hook wiring with taxonomy-driven registration that includes the already-defined `tool.*` and `permission.*` events.
3. MUST implement hook-binding projection with side-effect-free build and atomic apply semantics so partial dispatch-table corruption cannot occur on projector failure.
4. MUST remove the legacy authoritative hook-binding path for migrated bindings in the same cutover phase, per the TechSpec "Development Sequencing" tranche-1 gate.
</requirements>

## Subtasks

- [x] 7.1 Add hook-binding codecs, typed store usage, and projector wiring on top of the shared resource runtime
- [x] 7.2 Replace the manual hook runtime bridge and session wiring with taxonomy-driven registration
- [x] 7.3 Wire `tool.*` and `permission.*` events end to end through the migrated hook runtime
- [x] 7.4 Add tranche-1 coverage for hook dispatch, atomic swap behavior, and legacy-authority removal

## Implementation Details

Follow the TechSpec sections "Data Models", "Development Sequencing", "Testing Approach", and "Technical Dependencies". This task should be the first concrete migrated family and should fully satisfy the tranche-1 verification gate for hooks before automation or bridge migration begins. Because AGH is alpha and the workspace rules reject compatibility shims, treat this as a clean cutover: do not add backfill, dual-write, or legacy compatibility paths for migrated hook bindings.

### Relevant Files

- `internal/hooks/dispatch.go` — Hook dispatch becomes resource-backed and must honor atomic projection updates
- `internal/hooks/events.go` — Taxonomy data drives supported families and event registration
- `internal/session/hooks.go` — Session hook composition remains thin while migrated binding authority moves to the shared runtime
- `internal/session/manager_hooks.go` — Session lifecycle dispatch must consume the migrated binding registry
- `internal/daemon/hooks_bridge.go` — Daemon-owned hook bridge must replace the hand-enumerated dispatch matrix with taxonomy-driven binding support

### Dependent Files

- `internal/session/manager_hooks_test.go` — Session hook coverage must prove `tool.*` and `permission.*` are wired end to end
- `internal/api/httpapi/hooks_integration_test.go` — Hook API coverage later depends on the migrated binding authority
- `internal/extension/manager.go` — Extension-provided hook bindings later publish through the canonical runtime

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes the runtime authoritative for hook bindings
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Requires direct removal of replaced hook authority in the same phase
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Defines the tranche-1 verification gate this cutover must satisfy
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires hook runtime rebuilds to come from canonical snapshots
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Hook binding projection runs on the shared driver
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Hook dispatch code must consume typed records rather than raw JSON

## Deliverables

- Resource-backed hook binding authority with projector wiring and atomic apply semantics
- Taxonomy-driven hook runtime wiring in session and daemon hook bridges
- End-to-end runtime wiring for `tool.*` and `permission.*` hook events
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for migrated hook dispatch, atomic swap behavior, and legacy path removal **(REQUIRED)**

## Tests

- Unit tests:
  - [x] taxonomy-driven runtime wiring includes `tool.pre_call`, `tool.post_call`, `tool.post_error`, `permission.request`, `permission.resolved`, and `permission.denied`
  - [x] hook projector `Build` computes the next dispatch table without mutating the live runtime and `Apply` swaps it atomically
  - [x] permission request patch guards still block escalation after the binding source moves to resources
  - [x] no legacy hand-enumerated hook-family path remains authoritative once resource-backed bindings are enabled
- Integration tests:
  - [x] a resource-backed hook binding fires on a real `tool.pre_call` or `tool.post_call` path through the session runtime
  - [x] a resource-backed permission hook fires on `permission.request`, `permission.resolved`, and `permission.denied` end to end
  - [x] projector failure preserves the previously applied hook dispatch table rather than leaving a partially updated runtime
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Hook bindings are authoritative in the shared runtime instead of split across bespoke paths
- `tool.*` and `permission.*` are finally wired through the shipped hook runtime end to end
