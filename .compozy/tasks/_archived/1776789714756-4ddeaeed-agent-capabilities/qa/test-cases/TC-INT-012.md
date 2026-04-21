## TC-INT-012: Rich discovery returns an empty catalog for unknown capability IDs

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/network`
**Traceability:** Task 04; RFC 003 rule that unknown `agh.capability_ids` filters return `[]`.
**Execution Surfaces:** Router request/response envelopes.
**Durable Regression Anchors:** `TestRouterWhoisRichCapabilityDiscoveryReturnsEmptyCatalogForUnknownIDsOrMissingCatalog`

### Objective

Verify a rich `whois` request that filters by unknown capability IDs still returns the responder `peer_card` and an explicit empty rich catalog rather than an omitted key or error.

### Preconditions

- [ ] A responding peer exists with a valid non-empty capability catalog.
- [ ] The executor can send a directed `whois` request with an unknown `agh.capability_ids` value.

### Test Steps

1. Send a directed `whois` request with `agh.include=["capability_catalog"]` and `agh.capability_ids=["missing-capability"]`.
   - **Expected:** The request is accepted as valid.
2. Inspect the response.
   - **Expected:** The normal `peer_card` is present and `agh.capability_catalog.capabilities = []`.
3. Confirm no fallback behavior occurs.
   - **Expected:** AGH does not return the full catalog and does not reject the request.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Filter ID | `missing-capability` | Intentionally absent from responder catalog |

### Post-conditions

- Routers can be cleaned up.
- Evidence includes the exact response payload.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Multiple unknown IDs | `["missing-a", "missing-b"]` | Still returns explicit empty catalog |
| Mixed known and unknown IDs | One valid, one invalid | Returns only the known capability in normalized order |

### Related Test Cases

- `TC-INT-010`
- `TC-INT-011`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- Keep the unknown-ID response artifact separate from the no-catalog response artifact in `TC-INT-011`.
