---
status: completed
title: "Migrate tools and MCP servers to resources"
type: refactor
complexity: high
dependencies:
  - task_03
  - task_04
  - task_05
  - task_06
---

# Task 08: Migrate tools and MCP servers to resources

## Overview

Move the static and dynamic publication paths for tools and MCP servers into the canonical resource runtime. This is the smallest useful split of the original tranche-1 publication lane: it closes the real shipped tool gap, lands the shared publication pattern on two closely related kinds, and completes the resource-based replacement for `provide_tools`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add resource-backed publication for tools and MCP servers from manifests, daemon config, and extension snapshots using the static surface catalog from task 04.
2. MUST make the canonical resource runtime authoritative for these migrated families and remove manager-owned or one-off authoritative catalogs for their definitions in the same cutover phase.
3. MUST replace dynamic tool publication through `provide_tools` with `resources/snapshot` once tool records are backed by the shared runtime.
4. MUST exercise the canonical UDS CRUD path against at least one migrated tool or MCP record so tranche-1 validation proves operator writes, typed store semantics, and publication rebuild on a real migrated kind.
</requirements>

## Subtasks

- [x] 8.1 Add codecs and projectors for `tool` and `mcp_server` resource kinds
- [x] 8.2 Move manifest, config, and extension-manager publication flows for tools and MCP servers onto the canonical resource store
- [x] 8.3 Remove `provide_tools` as the authoritative dynamic tool path once tool resources are live
- [x] 8.4 Add tranche-1 coverage for static publication, dynamic snapshots, UDS CRUD smoke, and boot rebuild behavior

## Implementation Details

Follow the TechSpec sections "Data Models", "API Endpoints", "Development Sequencing", and "Technical Dependencies". This task is the second half of tranche 1 and should land alongside task 07 before automation, bridges, or bundles start migrating. Because AGH is alpha and the workspace rules reject compatibility shims, treat this as a clean cutover: do not add backfill, dual-write, or legacy compatibility paths for tool and MCP publication.

### Relevant Files

- `internal/tools/tool.go` — Tool definitions need a typed desired-state contract behind the resource runtime
- `internal/config/mcpjson.go` — MCP server declaration inputs must converge into canonical resource-backed publication
- `internal/extension/manifest.go` — Manifest resources need first-class tool publication and canonical MCP server mapping
- `internal/extension/manager.go` — Current catalog ownership and dynamic contribution logic must move behind canonical resource publication
- `sdk/typescript/src/extension.ts` — Dynamic tool contribution must stop relying on `provide_tools` once snapshot publication is authoritative

### Dependent Files

- `internal/daemon/extensions.go` — Daemon composition later depends on rehydrating tool and MCP publication from canonical records
- `internal/api/udsapi/udsapi_integration_test.go` — Tranche-1 validation should prove the UDS CRUD path against a migrated kind
- `internal/extension/manager_integration_test.go` — Extension publication coverage must prove static and dynamic contributions converge in the resource runtime

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes the shared runtime authoritative for migrated definitions
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Requires removal of legacy catalog authority in the same phase
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires tranche-1 verification before broader migration
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires rebuilt family state to come from canonical persisted snapshots
- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Keeps publication and reads constrained by daemon-side grants
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Prevents migrated family logic from reintroducing untyped payload handling

## Deliverables

- Resource-backed publication for tool and MCP server definitions
- Extension-manager and config flows migrated to the canonical runtime for these families
- Removal of `provide_tools` as the authoritative dynamic tool publication path
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for static publication, dynamic snapshots, UDS CRUD smoke, boot rebuild, and path removal **(REQUIRED)**

## Tests

- Unit tests:
  - [x] manifest tool declarations normalize into the same canonical record shape as dynamic extension snapshots
  - [x] MCP server codecs reject invalid specs before persistence and expose typed records to domain consumers
  - [x] the extension manager no longer remains the authoritative catalog owner for migrated tool and MCP definitions
  - [x] `provide_tools` is no longer advertised or required once tool records are published through `resources/snapshot`
- Integration tests:
  - [x] a static extension manifest publishes tool or MCP definitions into the canonical resource store
  - [x] an extension snapshot adds, updates, and removes tool records through `resources/snapshot` without using `provide_tools`
  - [x] UDS resource CRUD can create or update a migrated tool or MCP record and the canonical publication path rehydrates it correctly
  - [x] daemon boot rebuild rehydrates migrated tool and MCP state from persisted resource records
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Tools and MCP servers are published through one canonical runtime instead of scattered catalog paths
- `provide_tools` is removed as a one-off extensibility mechanism once resource-backed tool publication lands
