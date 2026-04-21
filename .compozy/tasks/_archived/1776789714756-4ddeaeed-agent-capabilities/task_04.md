---
status: completed
title: Explicit Rich Capability Discovery via Whois
type: backend
complexity: high
dependencies:
  - task_03
---

# Task 04: Explicit Rich Capability Discovery via Whois

## Overview

Implement the explicit rich capability discovery flow over `whois` using AGH-specific envelope extensions instead of overloading `query` or bloating periodic `greet` traffic. This task adds full-catalog and filtered rich discovery, keeps the legacy `whois` path cheap when not requested, and enforces the response-shape rules approved in the RFC update.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, RFC 003, and tasks 01-03 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC "Projection rules" plus the RFC 003 `whois` enrichment section
- KEEP RICH DISCOVERY EXPLICIT - ordinary `whois` must remain lean unless `agh.include` requests `capability_catalog`
- DO NOT move rich catalog data into `PeerCard.Ext`; the rich catalog belongs in envelope `ext`
- TESTS REQUIRED - cover no-request, full-catalog, filtered-catalog, no-catalog, unknown-id, and oversized-response cases
- GREENFIELD: do not smuggle new semantics into `WhoisBody.Query`; use structured envelope extensions only
</critical>

<requirements>
- MUST support `ext["agh.include"]` containing `capability_catalog` as the only trigger for rich capability discovery
- MUST support optional `ext["agh.capability_ids"]` filtering and return only matching capabilities in normalized catalog order
- MUST return `ext["agh.capability_catalog"].capabilities = []` when rich discovery is explicitly requested but the peer has no catalog or none of the requested IDs match
- MUST keep baseline `whois` request/response behavior unchanged when rich discovery is not requested
- MUST keep `peer_card` brief and continue returning it alongside any rich discovery payload
- MUST guard against emitting oversized rich catalog responses that would violate the network envelope size limit
- SHOULD ignore unknown AGH extension keys rather than rejecting otherwise valid `whois` requests
</requirements>

## Subtasks
- [x] 4.1 Add the AGH `whois` extension parsing needed to detect rich capability discovery requests
- [x] 4.2 Generate rich capability catalog responses for full-catalog and filtered `capability_ids` requests
- [x] 4.3 Keep ordinary `whois` behavior unchanged when no rich discovery is requested
- [x] 4.4 Enforce empty-catalog behavior for no-catalog and unknown-ID cases plus a safe oversized-response guard
- [x] 4.5 Add unit and integration coverage for request parsing, response generation, filtering, and response-size protection

## Implementation Details

See TechSpec "Projection rules" and the RFC 003 sections for `whois`, `agh.include`, `agh.capability_ids`, and `agh.capability_catalog`. The rich catalog lives in envelope `ext`, not inside the `WhoisBody` or `PeerCard`. The code should continue to treat `PeerCard` as the identity/brief-discovery object and envelope `ext` as the optional rich discovery carrier.

### Relevant Files
- `internal/network/router.go` - `handleWhois` currently generates minimal responses and is the primary rich-discovery integration point
- `internal/network/envelope.go` - envelope and `WhoisBody` wire types that will carry the AGH-specific `ext` enrichment
- `internal/network/validate.go` - envelope normalization and validation must continue to accept opaque ext payloads while preserving `whois` body rules
- `internal/network/router_test.go` - existing `whois` request/response tests to extend with rich discovery cases
- `internal/network/router_integration_test.go` - integration-style router coverage for real envelope round trips and peer presence refresh behavior

### Dependent Files
- `internal/network/manager.go` - manager-level inbound routing and outbound reply publishing will surface the richer `whois` responses generated here
- `internal/network/manager_test.go` - higher-level manager tests need coverage for rich discovery through real manager message handling
- `internal/network/helpers_test.go` - useful place for ext/validation regression cases if request or response normalization changes
- `docs/rfcs/003_agh-network-v0.md` - runtime behavior should stay aligned with the already-updated RFC contract

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - rich discovery must be projected from the explicit runtime catalog
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - rich discovery exposes the structured delegation offer, not a rigid RPC schema

## Deliverables
- Explicit `whois` rich-discovery support driven by `agh.include` and optional `agh.capability_ids`
- Rich responses that include `agh.capability_catalog` while keeping `peer_card` brief
- Deterministic empty-catalog behavior for no-catalog and unknown-ID requests
- Response-size guard that prevents invalid oversized rich `whois` responses **(REQUIRED)**
- Updated unit and integration tests for `whois` rich discovery **(REQUIRED)**
- Test coverage >=80% for touched network packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] A `whois` request with no `agh.include` continues to generate the current minimal response only
  - [x] A `whois` request with `agh.include=["capability_catalog"]` returns `agh.capability_catalog` with required `id`, `summary`, and `outcome` fields for each entry
  - [x] A request with `agh.capability_ids=["create-landing-page"]` returns only that capability and preserves normalized catalog order
  - [x] Unknown `agh.capability_ids` return `agh.capability_catalog.capabilities = []` instead of omitting the catalog key
  - [x] A peer with no catalog returns `agh.capability_catalog.capabilities = []` when rich discovery is explicitly requested
  - [x] Unknown AGH ext keys on an otherwise valid `whois` request are ignored rather than causing rejection
  - [x] A rich response that would exceed the allowed envelope size is blocked or reduced by the chosen guard and is never emitted as an invalid envelope
- Integration tests:
  - [x] Directed `whois` rich discovery returns the responder `peer_card` plus `agh.capability_catalog` to the requester in one valid response envelope
  - [x] Filtered rich discovery returns only the requested capability entries while still refreshing remote presence from the response `peer_card`
  - [x] Envelope normalization and router publish/receive cycles preserve `agh.capability_catalog` ext payloads intact for valid rich responses
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- AGH rich capability discovery is available only when explicitly requested and remains separate from the brief `PeerCard`
- No-catalog, unknown-ID, and oversized-response cases have deterministic behavior instead of implicit fallbacks
