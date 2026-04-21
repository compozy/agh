## TC-INT-009: Brief capability discovery stays aligned across greet, peer state, and API payloads

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/network`, `internal/api/core`
**Traceability:** Task 03; TechSpec projection rules; RFC 003 brief capability extension.
**Execution Surfaces:** `greet`, peer registry/listing, API payload conversion.
**Durable Regression Anchors:** `TestManagerJoinPublishesProjectedCapabilityBriefInInitialAndReconnectGreets`, `TestProjectCapabilityBriefViewMatchesProjectedIDsAndBriefEntries`, `TestCloneAndNormalizePeerCardPreserveCapabilityBriefExt`, `TestNetworkConversionHelpersPreserveMetadata`

### Objective

Verify brief discovery is derived from one normalized catalog and stays aligned across `peer_card.capabilities`, `peer_card.ext["agh.capabilities_brief"]`, peer listings, peer detail payloads, and reconnect/regreet flows.

### Preconditions

- [ ] A capability-aware local peer can join a network channel.
- [ ] The executor can capture `greet` traffic, peer registry state, and API payloads for the same peer.

### Test Steps

1. Join a local peer that has at least two capabilities.
   - **Expected:** The initial `greet` advertises `peer_card.capabilities` and `agh.capabilities_brief`.
2. Compare the brief projection surfaces for the same peer.
   - **Expected:** Capability IDs and brief entries share the same order and IDs across the wire, in-memory peer state, and API payloads.
3. Trigger a reconnect or regreet path.
   - **Expected:** The same brief metadata is preserved without duplication, mutation, or omission.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Capability IDs | `review-pr`, `draft-spec` | Stable-order comparison |
| Brief fields | `id`, `summary` | Only fields allowed in brief projection |

### Post-conditions

- Any temporary peers can be removed from the channel.
- Evidence includes one `greet` artifact, one peer-list/detail artifact, and one API payload artifact.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| No-catalog peer | Control fixture with no catalog | `peer_card.capabilities = []` and brief ext key omitted |
| Cloning/normalization | Convert through API payload helpers | Metadata is copied, not aliased or mutated |

### Related Test Cases

- `TC-INT-008`
- `TC-INT-010`
- `TC-INT-011`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- Capture one aligned artifact set from a single peer so the evidence compares the exact same source catalog.
