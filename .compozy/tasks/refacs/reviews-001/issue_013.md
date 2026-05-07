---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/testutil/bridge_stub.go
line: 139
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:896fb193132f
review_hash: 896fb193132f
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 013: ResolveOrCreateRoute and UpsertRoute defaults return ErrBridgeRouteNotFound — semantically wrong for write/create ops.
## Review Comment

Both operations guarantee the caller a route object (by creating it if necessary); returning "not found" as their default zero-stub behaviour contradicts that contract and is inconsistent with every other write-op default in the same file (`CreateInstance` → `nil, nil`, `PutSecretBinding` → `nil`, `PutBridgeTaskSubscription` → `nil`).

Tests that rely on unconfigured stubs will silently get a "not found" failure from code paths that should succeed, producing confusing test output.

## Triage

- Decision: `INVALID`
- Notes:
  The unconfigured route stubs intentionally fail fast instead of fabricating success objects. Returning `nil, nil` here would hide missing test setup and weaken failure signals. For these stateful route operations, `ErrBridgeRouteNotFound` is a deliberate default.
