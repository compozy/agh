---
status: completed
title: Brief Capability Projection in Peer Cards
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 03: Brief Capability Projection in Peer Cards

## Overview

Project the normalized capability catalog into the brief discovery surfaces that peers see first: `peer_card.capabilities` and `peer_card.ext["agh.capabilities_brief"]`. This task makes local peer cards, greets, peer listings, and API payloads advertise capabilities consistently without bloating the core `PeerCard` shape.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and tasks 01-02 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC "Projection rules" and RFC 003 capability discovery updates
- KEEP `PeerCard` BRIEF - only IDs plus short summaries belong in the brief projection
- DO NOT let `peer_card.capabilities` and `agh.capabilities_brief` drift; they must come from the same normalized catalog
- TESTS REQUIRED - cover router, peer registry, manager, and API payload conversion where brief metadata crosses boundaries
- GREENFIELD: do not hide brief projection behind ad hoc ext builders scattered across packages
</critical>

<requirements>
- MUST derive `PeerCard.Capabilities` and `peer_card.ext["agh.capabilities_brief"]` from the same normalized capability catalog and keep exact ID alignment
- MUST emit `PeerCard.Capabilities = []` and omit `agh.capabilities_brief` entirely when no capability catalog exists
- MUST keep brief entries limited to `id` and `summary`, with summaries suitable for periodic `greet` traffic
- MUST preserve `PeerCard.Ext` cloning and normalization semantics through router, manager, and API payload conversion
- MUST keep the existing `PeerCard` validation contract intact, including non-nil array requirements in `normalizeAndValidatePeerCard`
- SHOULD centralize brief projection helpers so greet publishing, peer listing, and detail views all reuse the same logic
</requirements>

## Subtasks
- [x] 3.1 Implement the brief projection helper that turns a normalized catalog into capability IDs plus `agh.capabilities_brief`
- [x] 3.2 Build local peer cards from the projected brief capability view during join and regreet flows
- [x] 3.3 Ensure `PeerCard.Ext` continues to round-trip through router, manager, and API payload conversion without mutation or aliasing
- [x] 3.4 Make the no-catalog path explicit and deterministic across peer registration and serialization
- [x] 3.5 Add precise unit and integration coverage for projection agreement, greet publication, and API visibility

## Implementation Details

See TechSpec "Projection rules" and the RFC 003 `greet` update. The projection should stay small and reusable: `peer_card.capabilities` remains the minimal index, while `peer_card.ext["agh.capabilities_brief"]` carries only `id` plus `summary`.

### Relevant Files
- `internal/network/peer.go` - canonical `PeerCard` creation, cloning, and local peer registration behavior
- `internal/network/peer_test.go` - peer-card normalization and cloning regressions should cover brief projection state
- `internal/network/validate.go` - `PeerCard` validation requires non-nil arrays and should tolerate the new brief ext keys
- `internal/api/core/network.go` - API payload conversion already copies `PeerCard.Ext` and must preserve `agh.capabilities_brief`
- `internal/api/core/network_test.go` - existing network payload conversion tests to extend with capability brief assertions

### Dependent Files
- `internal/network/manager.go` - local joins and regreets will consume the projected brief peer card
- `internal/network/router.go` - greet publishing must serialize the enriched local peer card consistently
- `internal/api/core/network_details.go` - peer detail payloads should surface the same brief ext metadata seen in list views
- `internal/network/manager_test.go` - manager-level presence and audit tests need updates once local greets advertise capability brief metadata

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - brief discovery must be projected from the explicit runtime catalog
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - brief discovery exposes the minimal outcome-oriented summary, not internal skills

## Deliverables
- Reusable projection helper for capability IDs and `agh.capabilities_brief`
- Local peer cards that advertise brief capabilities correctly during joins and greets
- API/network payload conversion that preserves brief capability metadata without aliasing source maps
- Updated unit tests in peer/router/API surfaces for projection correctness **(REQUIRED)**
- Integration coverage proving brief discovery appears in real join/greet/list flows **(REQUIRED)**
- Test coverage >=80% for touched network/API packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Brief projection returns the capability ID list and `agh.capabilities_brief` entries in the same stable order as the normalized catalog
  - [x] Each brief entry `id` exactly matches one `PeerCard.Capabilities` element from the same projection output
  - [x] No-catalog projection yields `PeerCard.Capabilities = []` and omits the `agh.capabilities_brief` ext key
  - [x] `clonePeerCard()` and peer normalization preserve brief capability ext payloads without mutating the caller-owned ext map
  - [x] `NetworkPeerPayloadFromInfo()` clones `agh.capabilities_brief` into the contract payload rather than sharing the source map by reference
- Integration tests:
  - [x] Joining a local peer with capabilities causes the initial `greet` to advertise both `peer_card.capabilities` and `agh.capabilities_brief`
  - [x] `ListPeers()` and peer detail payloads expose `agh.capabilities_brief` for the same peer card seen on the wire
  - [x] Reconnect/regreet flows preserve brief capability IDs and summaries instead of dropping or duplicating them
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Brief capability discovery is visible through peer cards, greets, and API payloads with no drift between IDs and summaries
- No-catalog peers still satisfy the `PeerCard` validation contract while remaining discovery-empty
