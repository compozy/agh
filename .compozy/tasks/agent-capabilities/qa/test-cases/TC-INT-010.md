## TC-INT-010: Explicit rich `whois` discovery returns full and filtered capability catalogs

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/network`
**Traceability:** Task 04; RFC 003 rich capability discovery extension; TechSpec projection rules for explicit `whois`.
**Execution Surfaces:** Router request/response envelopes, remote presence refresh.
**Durable Regression Anchors:** `TestRouterWhoisRichCapabilityDiscoveryReturnsCapabilityCatalog`, `TestRouterWhoisRichCapabilityDiscoveryFiltersRequestedIDsInCatalogOrder`, `TestDirectedWhoisRichDiscoveryDeliversPeerCardAndCapabilityCatalog`, `TestDirectedWhoisRichDiscoveryFilteringRefreshesRemotePresence`

### Objective

Verify rich capability discovery is only returned when explicitly requested and that full-catalog and filtered-catalog responses preserve normalized order while still returning the normal `peer_card`.

### Preconditions

- [ ] Two routers or an equivalent request/response harness are available.
- [ ] The responding peer has a valid multi-entry capability catalog.
- [ ] The executor can inspect both the request `ext` and response `ext`.

### Test Steps

1. Send a directed `whois` request with `ext["agh.include"] = ["capability_catalog"]`.
   - **Expected:** The response includes the normal `peer_card` plus `ext["agh.capability_catalog"]`.
2. Send a second request with `ext["agh.capability_ids"]` targeting one known capability.
   - **Expected:** The response returns only the requested capability entry and preserves catalog order.
3. Compare the rich catalog to the peer card.
   - **Expected:** The peer card remains brief while the full details live only in the envelope `ext`.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Include key | `agh.include=["capability_catalog"]` | Required trigger |
| Filter key | `agh.capability_ids=["review-pr"]` | Filtered response |
| Expected response key | `agh.capability_catalog` | Envelope `ext`, not `peer_card.ext` |

### Post-conditions

- Test routers can be shut down or cleaned up.
- Evidence includes both a full-catalog response and a filtered response.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| No include key | Ordinary `whois` request | No rich catalog is returned |
| Unknown AGH ext key | Add unrelated `agh.*` extension | Request remains valid and unknown key is ignored |

### Related Test Cases

- `TC-INT-009`
- `TC-INT-011`
- `TC-INT-012`
- `TC-INT-013`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- Keep request and response artifacts together so the include/filter semantics are auditable.
