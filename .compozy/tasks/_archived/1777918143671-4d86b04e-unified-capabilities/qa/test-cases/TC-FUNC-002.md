# TC-FUNC-002: API Peer List Filtering And Enrichment

**Priority:** P1
**Type:** Functional
**Module:** API Core
**Requirement:** Peer summaries should be accurate and cheap to enrich.

## Objective

Verify peer listing filters by channel, enriches local peers from session records, preserves remote peer-card identity, and strips internal capability-discovery metadata.

## Preconditions

- Network service fixture returns local and remote peers.
- Session manager fixture returns matching local sessions and may return failures.
- Peer cards include capability brief extension metadata.

## Test Steps

1. Request `GET /api/network/peers?channel=builders`.
   **Expected:** Service receives `builders`; response excludes peers outside that channel.
2. Include a local peer whose `session_id` maps to a session name.
   **Expected:** Display name prefers session name, then agent name, then peer card, then peer ID.
3. Include a remote peer without `session_id`.
   **Expected:** Display name comes from peer card or peer ID and no session lookup is required.
4. Include `agh.capabilities_brief` and `agh.capability_catalog` in peer-card `ext`.
   **Expected:** Brief capabilities are projected; internal discovery keys are not returned in public `ext`.
5. Force session enrichment failure.
   **Expected:** Peer list still succeeds with peer-card fallback and logs a best-effort warning.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Blank display name | `"   "` | falls back to peer ID or session |
| Unknown session ID | no matching session | no 500 |
| Unknown catalog state | rich catalog present but not known | stale rich catalog ignored |

## Related

- SMOKE-001
- TC-FUNC-003
