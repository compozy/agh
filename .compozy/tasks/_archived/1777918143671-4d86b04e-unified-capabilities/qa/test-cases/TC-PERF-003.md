# TC-PERF-003: Peer Session Enrichment Efficiency

**Priority:** P2
**Type:** Performance
**Module:** API Core
**Requirement:** Peer listing should enrich sessions in one authoritative lookup.

## Objective

Verify `NetworkPeers` does not issue per-peer session lookups and handles large peer lists efficiently.

## Preconditions

- Network service fixture can return many peers with mixed session IDs.
- Session manager fixture can count `ListAll` and `Status` calls.

## Test Steps

1. Return 200 peers, 100 with local session IDs.
   **Expected:** Handler calls `Sessions.ListAll` once and never calls `Status`.
2. Return peers with repeated session IDs.
   **Expected:** Enrichment map deduplicates wanted session IDs.
3. Return no peers.
   **Expected:** Session manager is not called.
4. Return session-list error.
   **Expected:** Peer endpoint still succeeds with fallback names.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Nil session manager | no manager | no panic, no enrichment |
| Blank session ID pointer | `" "` | ignored |
| Unknown sessions | peer session missing | fallback display name |

## Related

- TC-FUNC-002
- TC-FUNC-003
