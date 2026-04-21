## TC-INT-013: Oversized rich `whois` responses are rejected before publish

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/network`
**Traceability:** Task 04; TechSpec envelope-size guard; RFC 003 requirement that responses stay within protocol limits.
**Execution Surfaces:** Router receive path, envelope-size guard, transport publish count.
**Durable Regression Anchors:** `TestRouterWhoisRichCapabilityDiscoveryRejectsOversizedResponse`

### Objective

Verify AGH blocks rich `whois` responses that would exceed the protocol envelope limit and never emits an invalid oversized envelope.

### Preconditions

- [ ] A responding peer exists with an intentionally large capability catalog or payload.
- [ ] The executor can observe both the router error and whether the transport published anything.

### Test Steps

1. Construct or load a responder whose rich capability payload would exceed the allowed envelope size.
   - **Expected:** The responder is otherwise valid and only fails because of envelope size.
2. Send a directed rich `whois` request for that peer.
   - **Expected:** The router rejects the response path with `ErrEnvelopeTooLarge` or the equivalent guard error.
3. Inspect the transport activity.
   - **Expected:** No invalid response envelope is published.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Trigger | Explicit `agh.include=["capability_catalog"]` request | Required to hit rich discovery path |
| Payload | Oversized rich catalog | Must exceed protocol envelope limit |

### Post-conditions

- Oversized test fixtures can be removed.
- Evidence includes the guard error and zero-publish confirmation.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Filtered request | Request a smaller valid subset | Response succeeds when it no longer exceeds the limit |
| Non-rich `whois` | Ordinary `whois` request | No envelope-size failure if the peer card remains within normal limits |

### Related Test Cases

- `TC-INT-010`
- `TC-INT-012`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This is a release-blocking safety case because invalid protocol envelopes must never be emitted.
