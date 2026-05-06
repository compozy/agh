---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/api/httpapi/network_test.go
line: 144
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232600854,nitpick_hash:a9c584d5d889
review_hash: a9c584d5d889
source_review_id: "4232600854"
source_review_submitted_at: "2026-05-06T01:29:19Z"
---

# Issue 003: Consider using newTestRouter for consistency.
## Review Comment

This test manually creates a gin.Engine and registers only one route, while other tests in this file use `newTestRouter(t, handlers)`. If the goal is test isolation, this is fine, but using the shared router helper would ensure the handler is wired the same way as production.

## Triage

- Decision: `invalid`
- Notes:
  - This test already exercises the real handler wiring through `newTestHandlers(...)`; the only custom piece is the intentionally minimal `gin.Engine` with one route registration.
  - Using `newTestRouter(...)` would widen the test surface to unrelated network routes and middleware without increasing coverage for the behavior under test, which is preserving routing metadata from `NetworkPeerMessages`.
  - The current isolated router keeps failure locality tighter and is not a correctness or maintainability bug, so no code change is warranted.

## Resolution

- Left the existing isolated route-registration pattern in place.
- Confirmed the analysis during fresh full `make verify`; no production or test bug was present here.
