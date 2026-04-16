---
status: completed
title: "Migrate bridge instances to resource projection"
type: refactor
complexity: high
dependencies:
  - task_07
  - task_08
---

# Task 11: Migrate bridge instances to resource projection

## Overview

Move bridge instance desired state into the shared runtime without collapsing bridge delivery, health, and routing into generic records. This task is the external-state projector proof: bridge projection must compute desired runtime deltas from resources while converging live bridge runtime safely and observably.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST migrate `bridge.instance` desired state into the canonical resource runtime and make those records authoritative for bridge configuration.
2. MUST carry the richer bridge desired-state shape introduced by the bridge-adapters work, including provider-authored config metadata, `dm_policy`, `provider_config`, and delivery defaults, while validating resource writes against provider-manifest metadata such as `bridge.secret_slots` and `bridge.config_schema`.
3. MUST keep bridge delivery state, assigned-instance visibility for managed adapters, runtime status reporting, route state, health, degradation, and other operational runtime data outside the generic resource store, consistent with the TechSpec "Operational endpoints" boundary.
4. MUST preserve family-specific bridge adapter Host API methods such as `bridges/instances/list`, `bridges/instances/get`, `bridges/instances/report_state`, and `bridges/messages/ingest` as operational surfaces backed by canonical `bridge.instance` authority rather than rerouting adapters through generic same-source `resources/get|list`.
5. MUST implement bridge projection using the external-state projector rules from the TechSpec, where `Build` computes a validated delta plan off-path and `Apply` atomically swaps daemon-visible desired runtime state before converging live side effects.
6. MUST remove the legacy bridge-definition authority in the same phase once the resource-backed projector is authoritative.
</requirements>

## Subtasks

- [x] 11.1 Add codec and typed record support for `bridge.instance`, including richer bridge desired-state fields and provider-manifest validation
- [x] 11.2 Replace legacy bridge-definition authority with resource-backed projection into the bridge runtime registry
- [x] 11.3 Preserve assigned-instance visibility, status reporting, delivery, route, and health read models as operational bridge-owned state
- [x] 11.4 Add external-state projector coverage for rollback, degraded state, and boot rebuild

## Implementation Details

Follow the TechSpec sections "Core Interfaces", "Impact Analysis", "Testing Approach", and "Technical Considerations". This task should be the reference implementation for the external-state projector contract and should not shortcut bridge runtime convergence by pushing live connection state into generic resources. Because AGH is alpha and the workspace rules reject compatibility shims, treat this as a clean cutover: do not add backfill, dual-write, or legacy compatibility paths for bridge definitions. Managed bridge providers continue reading daemon-assigned instances through family-specific bridge Host APIs after cutover; this task should move the desired-state authority underneath those APIs rather than replacing them with same-source `resources/get|list`.

### Relevant Files

- `internal/bridges/types.go` — Bridge desired-state configuration needs a typed codec boundary for resource records
- `internal/bridges/registry.go` — The daemon-visible bridge registry must rebuild from canonical resource records
- `internal/extension/manifest.go` — Provider metadata such as `secret_slots` and `config_schema` must inform bridge desired-state validation
- `internal/extension/host_api_bridges.go` — Assigned-instance reads and status reporting stay operational while desired-state authority moves underneath
- `internal/store/globaldb/global_db_bridge.go` — Legacy bridge-definition persistence must be replaced or demoted during cutover
- `internal/daemon/bridges.go` — Daemon composition must wire the bridge projector and runtime rebuild path

### Dependent Files

- `internal/api/core/bridges.go` — Bridge runtime APIs must continue exposing operational state after desired-state migration
- `internal/api/httpapi/bridges_integration_test.go` — Integration coverage later proves bridge runtime behavior is preserved after cutover
- `internal/extension/host_api_bridges.go` — Bridge extension surfaces later depend on the canonical desired-state authority

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes the shared runtime authoritative for bridge desired state
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Requires removing the legacy definition authority in the same phase
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires bridge cutover verification before bundle composition starts
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires bridge desired state to rebuild from canonical snapshots
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Bridge projection runs on the shared driver with timeout and degraded-state support
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Bridge runtime code must consume typed records rather than raw JSON

## Deliverables

- Resource-backed desired-state authority for bridge instances
- Validation of resource-backed bridge desired state against provider-manifest schema and secret-slot metadata
- External-state projector logic that rebuilds bridge runtime configuration from canonical resource records
- Preservation of assigned-instance bridge Host APIs plus delivery, route, health, and degradation state outside the generic resource store
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for bridge cutover, degraded-state handling, and boot rebuild **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `bridge.instance` codec rejects invalid scope, malformed `provider_config`, invalid `dm_policy`, or illegal desired-state payloads before persistence
  - [x] provider-manifest `bridge.config_schema` and `bridge.secret_slots` metadata are enforced when validating resource-backed bridge desired state
  - [x] bridge projector `Build` computes the next desired bridge delta without opening speculative live connections
  - [x] bridge projector `Apply` degrades or rolls back cleanly when a live side effect fails after the daemon-visible registry swap
  - [x] legacy bridge-definition writes are no longer authoritative after resource-backed cutover
- Integration tests:
  - [x] an operator resource write adds, updates, and removes bridge instances through reconcile rather than legacy bridge-definition storage
  - [x] daemon boot rebuild reconstructs bridge desired state from persisted resource records
  - [x] managed bridge providers continue reading assigned instances through `bridges/instances/list|get` and reporting runtime state through `bridges/instances/report_state` after desired-state cutover
  - [x] bridge delivery, route, health, and degradation endpoints continue to work with operational state stored outside the generic resource runtime
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Bridge desired state is authoritative in the shared runtime while live bridge state remains bridge-owned
- The external-state projector contract is proven on a subsystem with real runtime side effects
