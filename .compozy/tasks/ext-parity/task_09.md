---
status: completed
title: "Migrate agents and skills to resources"
type: refactor
complexity: high
dependencies:
  - task_08
---

# Task 09: Migrate agents and skills to resources

## Overview

Move agent and skill definitions into the canonical resource runtime after tool and MCP publication is already resource-backed. This keeps the sequential `compozy start` flow aligned with real dependency pressure: agent tool references and skill MCP sidecars can now point at canonical definitions without forcing automation or bridges to absorb the original oversized publication task.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add resource-backed publication for agent and skill definitions from daemon config, workspace discovery, and extension resources, with the canonical runtime becoming authoritative for those definitions.
2. MUST keep skill content loading, provenance verification, and MCP sidecar merge as domain logic inside `internal/skills` while moving desired-state indexing and publication authority onto resource records.
3. MUST preserve alignment with task 08 so agent tool references and skill MCP attachments resolve against the canonical resource-backed tool and MCP definitions.
4. MUST remove legacy agent and skill definition authority in the same phase after the resource-backed publication path becomes authoritative.
</requirements>

## Subtasks

- [x] 9.1 Add codecs and typed store usage for `agent` and `skill`
- [x] 9.2 Move config, discovery, and extension publication flows for agents and skills onto canonical resource records
- [x] 9.3 Keep skill content, provenance, and MCP sidecar merge in the skills subsystem while moving definition authority to the shared runtime
- [x] 9.4 Add cutover coverage for canonical reference resolution, publication rebuild, and legacy-authority removal

## Implementation Details

Follow the TechSpec sections "Data Models", "Impact Analysis", "Testing Approach", and "Development Sequencing". This task should complete the agent and skill definition cutover end to end. Because AGH is alpha and the workspace rules reject compatibility shims, treat this as a clean cutover: do not add backfill, dual-write, or legacy compatibility paths for agent or skill publication.

### Relevant Files

- `internal/config/agent.go` — Agent definition inputs need a canonical resource-backed publication path
- `internal/skills/registry.go` — Skill registry state must consume canonical resource-backed definitions
- `internal/skills/loader.go` — Skill discovery remains domain-owned while publication authority moves to resources
- `internal/skills/mcp_sidecar.go` — Skill MCP sidecar merge must stay aligned with canonical MCP server definitions from task 08
- `internal/extension/manager.go` — Extension-provided agents and skills must publish through the canonical resource runtime

### Dependent Files

- `internal/daemon/extensions.go` — Daemon composition later depends on rehydrating agent and skill publication from canonical resource records
- `internal/extension/manager_integration_test.go` — Extension publication coverage must prove agent and skill contributions converge in the resource runtime
- `internal/skills/registry_integration_test.go` — Skill registry coverage must prove canonical publication, provenance, and sidecar behavior still work after cutover

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes the shared runtime authoritative for agent and skill definitions
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Requires removal of legacy definition authority after cutover
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires agent and skill cutover evidence before later tasks rely on the new catalogs
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Requires agent and skill publication to rebuild from canonical snapshots
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Agent and skill code must consume typed resource records rather than raw payloads

## Deliverables

- Resource-backed desired-state authority for agent and skill definitions
- Canonical publication paths for config, discovery, and extension-provided agents and skills
- Explicit preservation of skill content, provenance, and sidecar logic inside the skills subsystem
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for desired-state cutover, canonical reference resolution, and boot rebuild **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `agent` and `skill` codecs reject invalid specs before persistence and expose typed records to domain consumers
  - [x] agent definitions keep tool and MCP references aligned with the canonical tool and MCP records from task 08
  - [x] skill publication preserves provenance metadata and sidecar-derived MCP attachments without leaving the resource runtime responsible for file parsing
  - [x] legacy agent and skill definition sources are no longer authoritative after the resource-backed cutover
- Integration tests:
  - [x] daemon boot rebuild reconstructs agent and skill publication from persisted resource records
  - [x] an extension or workspace discovery path publishes agent and skill definitions into the canonical resource store
  - [x] canonical agent records continue to resolve tool and MCP references against the migrated catalogs from task 08
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Agent and skill definitions are authoritative in the shared resource runtime instead of scattered config and registry paths
- Agent tool references and skill MCP attachments resolve against canonical resource-backed publication
