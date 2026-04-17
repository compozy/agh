---
status: completed
title: "Migrate bundles and activation fan-out"
type: refactor
complexity: high
dependencies:
  - task_10
  - task_11
---

# Task 12: Migrate bundles and activation fan-out

## Overview

Move bundles and bundle activations onto the shared runtime after automation and bridges already use canonical resource records. This task replaces bespoke activation inventory with owner-indexed desired-state composition and closes the last mixed-kind outlier in the migration plan.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST migrate `bundle` and `bundle.activation` to canonical resource records and make bundle activation expansion authoritative in the shared runtime.
2. MUST replace `bundle_activation_inventory` with owner-indexed resource ownership using `owner_kind` and `owner_id`, consistent with the TechSpec "Data Models" section.
3. MUST implement bundle activation as the explicit mixed-kind projector outlier, using package-local adapter code for owned `automation.*` and `bridge.instance` fan-out without creating dependency cycles through `DependsOn()`.
4. MUST write owned downstream records through the canonical store, not through operator transport APIs or bespoke inventory tables, and MUST enforce allowlisted owned kinds plus activation-scoped deletion so bundle cleanup cannot target unrelated resources.
</requirements>

## Subtasks

- [x] 12.1 Add codecs and typed store usage for `bundle` and `bundle.activation`
- [x] 12.2 Replace activation inventory with owner-indexed owned-resource composition
- [x] 12.3 Implement mixed-kind activation fan-out into automation and bridge records through package-local projector adapters
- [x] 12.4 Add cutover coverage for allowlists, owned-resource cleanup, and cycle-free activation behavior

## Implementation Details

Follow the TechSpec sections "Data Models", "Authority and Validation Rules", "Development Sequencing", and "Technical Considerations". This task must land after automation and bridge migrations so activation fan-out can target already-migrated desired-state kinds instead of inventing new compatibility paths. Because AGH is alpha and the workspace rules reject compatibility shims, treat this as a clean cutover: do not add backfill, dual-write, or legacy compatibility paths for bundle activation state.

### Relevant Files

- `internal/bundles/service.go` — Bundle activation authority and fan-out logic move onto the canonical resource runtime here
- `internal/store/globaldb/global_db_bundles.go` — Legacy activation inventory persistence must be replaced by owner-indexed resource ownership
- `internal/api/core/bundles.go` — Bundle APIs must keep preview and operational behavior aligned with the new desired-state authority
- `internal/daemon/boot.go` — Boot topology must schedule bundle activation after its dependency kinds are already migrated
- `internal/bundles/model/model.go` — Bundle desired-state model needs typed resource contracts for activation and ownership

### Dependent Files

- `internal/automation/manager.go` — Bundle activation later writes owned automation resource records through the canonical store
- `internal/bridges/registry.go` — Bundle activation later writes owned bridge-instance resource records through the canonical store
- `internal/bundles/service_test.go` — Bundle coverage must prove allowlisted fan-out and activation-scoped cleanup after cutover

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes the shared runtime authoritative for bundle composition
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Requires removing bespoke activation inventory after cutover
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires bundle cutover verification before the migration is considered complete
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires bundle composition to rebuild from canonical snapshots
- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Defines allowlisted owned kinds and activation-scoped delete safety
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Keeps `DependsOn()` ordering separate from ownership fan-out
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Makes bundle activation the explicit mixed-kind adapter outlier rather than a leaked raw-JSON pattern

## Deliverables

- Resource-backed desired-state authority for bundles and bundle activations
- Removal of `bundle_activation_inventory` in favor of owner-indexed resource ownership
- Mixed-kind activation fan-out into automation and bridge resource records with explicit allowlist enforcement
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for activation fan-out, owned-resource cleanup, and cycle-free reconcile behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `bundle.activation` expansion rejects owned kinds outside the bundle allowlist
  - [x] owner-indexed cleanup deletes only records owned by the specific activation being removed
  - [x] mixed-kind bundle adapter decodes dependency kinds explicitly without exposing raw JSON to domain code
  - [x] activation fan-out does not register a reverse dependency cycle back to `bundle.activation`
- Integration tests:
  - [x] activating a bundle creates owned automation and bridge resource records through the canonical store
  - [x] deleting or deactivating an activation removes only the owned records for that activation and leaves unrelated resources untouched
  - [x] daemon boot rebuild rehydrates bundle and activation state from canonical resource records without any activation inventory table
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Bundle activation no longer relies on bespoke inventory tables or ad hoc fan-out authority
- The migration ends with owner-indexed composition on top of already-migrated automation and bridge resource kinds
