---
status: resolved
file: web/src/systems/bridges/components/bridge-detail-panel.test.tsx
line: 73
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:02027608129f
review_hash: 02027608129f
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 022: Consider unique defaults in makeRoute to avoid duplicate-key noise in tests.
## Review Comment

If multiple default routes are generated in one test, repeated `session_id`/`routing_key_hash` can create duplicate row keys and flaky assertions.

## Triage

- Decision: `valid`
- Root cause: `makeRoute` returns the same default `session_id` and `routing_key_hash` for every route. `BridgeEventStreamSection` keys rows as `${route.session_id}:${route.routing_key_hash}`, so tests that render multiple default routes can emit duplicate React keys and make row assertions noisy or flaky.
- Fix approach: make the route helper defaults deterministic but unique per call, while preserving explicit overrides for tests that need specific IDs.
- Resolution: `makeRoute` now generates deterministic unique default route IDs and hashes, and the component test asserts multiple default routes have distinct row identities. Targeted bridge tests and full `make verify` passed after the code change.
