---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/spec/spec.go
line: 1275
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:e2da4de872ad
review_hash: e2da4de872ad
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 007: Add missing OpenAPI specification for GET /api/network/peers/{peer_id}/messages endpoint.
## Review Comment

The `NetworkPeerMessages` handler is fully implemented, tested, and registered as a public route, but missing from the OpenAPI spec. Similar message endpoints for threads and directs are documented (lines 1313 and 1396 respectively), so this should follow the same pattern in `internal/api/spec/spec.go`.

## Triage

- Decision: `invalid`
- Notes: `NetworkPeerMessages` exists at the shared core-handler layer for focused handler tests, but it is not registered on either public HTTP or UDS route surface. `internal/api/httpapi/routes.go` and `internal/api/udsapi/routes.go` expose `/api/network/peers` and `/api/network/peers/{peer_id}`, not `/messages`. Since the endpoint is not currently public, the OpenAPI omission is correct for this batch.
