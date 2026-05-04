---
status: completed
title: Align Discovery, Peer Details, and API Contracts with Unified Capabilities
type: backend
complexity: high
dependencies:
  - task_01
  - task_03
---

# Task 04: Align Discovery, Peer Details, and API Contracts with Unified Capabilities

## Overview

Align brief discovery, rich discovery, peer details, and API-visible contracts so every surfaced capability shape now reflects the unified model. This task is where the runtime stops exposing a split mental model across `greet`, `whois`, peer-card details, and daemon API payloads.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "System Architecture", "Data Models", "API Endpoints", and "Testing Approach"
- PRESERVE THE DISCOVERY SPLIT - brief discovery stays in `greet`, rich discovery stays in `whois`, transfer stays in `kind:"capability"`
- KEEP API CONTRACTS IN LOCKSTEP WITH RUNTIME TYPES - avoid ad-hoc response shaping that diverges from the canonical model
- TESTS REQUIRED - peer details, contract serialization, filtering, and envelope-size behavior need explicit verification
- GREENFIELD: remove split-model terminology from API payloads instead of layering new fields on top of old names
</critical>

<requirements>
- MUST update brief capability discovery, rich capability catalogs, and peer detail payloads to use the unified capability schema
- MUST ensure `peer_card.capabilities`, `agh.capabilities_brief`, and explicit rich capability discovery remain internally consistent
- MUST update daemon API contracts and handlers so they expose the unified model without recipe-specific fields or terminology
- MUST preserve `whois` filtering semantics and envelope-size protections while switching to the new canonical capability content
- MUST keep the same discovery transport boundaries: `greet` for brief presence, `whois` for rich discovery, `capability` kind for transfer
- SHOULD update contract tests so future backend and frontend work have one authoritative payload shape to target
</requirements>

## Subtasks
- [x] 4.1 Update brief and rich capability discovery code paths to use the unified runtime model
- [x] 4.2 Align peer detail and peer-card surfaces so brief and rich capability views remain coherent
- [x] 4.3 Rewrite daemon/API contract types and handlers for the unified capability payloads
- [x] 4.4 Preserve filtering and response-size behavior under the new schema
- [x] 4.5 Add regression coverage for discovery, peer details, and API serialization

## Implementation Details

See TechSpec "System Architecture", "Data Models", "API Endpoints", and "Build Order" item 6. The central goal is to make every discovery-facing surface derive from the same normalized capability model introduced in task_01 and the transfer/runtime rules introduced in tasks_02-03.

### Relevant Files
- `internal/network/capability_brief.go` - brief discovery projection for `greet` and peer presence
- `internal/network/capability_catalog.go` - rich discovery filtering and catalog projection logic
- `internal/network/manager.go` - network join and peer-state orchestration that publishes discovery data
- `internal/network/manager_test.go` - manager-level discovery regressions for join and reconnect flows
- `internal/api/contract/contract.go` - daemon/API-visible network and peer types
- `internal/api/core/network.go` - network API handlers exposing peer lists and discovery summaries
- `internal/api/core/network_details.go` - detailed peer discovery responses including rich capability data

### Dependent Files
- `internal/api/core/network_test.go` - API regression coverage for peer details and unified capability payloads
- `internal/api/contract/contract_test.go` - contract serialization tests for updated capability fields
- `internal/api/udsapi/network_test.go` - UDS-visible network behavior relying on the unified contract
- `web/src/generated/agh-openapi.d.ts` - frontend typed client surface will change after this task lands
- `docs/rfcs/003_agh-network-v0.md` - documentation of discovery and payloads depends on the final backend/API shape

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - requires discovery and API contracts to stop describing two concepts
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - governs which structured fields remain first-class across discovery surfaces
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - constrains discovery to remain distinct from transferred capability artifacts

## Deliverables
- Updated brief and rich discovery projections derived from the unified capability model
- Peer detail and API contract updates with no lingering recipe vocabulary
- Regression coverage for peer-card summaries, rich `whois` payloads, filters, and size guards **(REQUIRED)**
- Contract tests establishing the authoritative backend payload shape for frontend and docs consumers **(REQUIRED)**
- Test coverage >=80% for the touched backend/API packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Brief discovery exposes unified capability summaries with the expected fields and ordering
  - [x] Rich capability catalog filtering by `capability_ids` still works under the new schema
  - [x] Peer detail serialization exposes unified capabilities without recipe-specific fields or labels
  - [x] Oversized rich discovery payloads are still rejected or bounded according to the existing guard behavior
- Integration tests:
  - [x] Initial join and reconnect flows publish brief capability discovery consistently across peer presence paths
  - [x] Explicit `whois` responses return rich unified capabilities derived from the same normalized catalog used in brief discovery
  - [x] HTTP and UDS network endpoints expose the same unified capability contract to clients
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Discovery, peer details, and daemon API contracts all speak the unified capability model
- Backend clients can rely on one coherent capability payload shape across brief and rich discovery
