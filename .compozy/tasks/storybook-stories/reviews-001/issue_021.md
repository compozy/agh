---
status: resolved
file: web/src/systems/bridges/mocks/handlers.ts
line: 73
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:132066857702
review_hash: "132066857702"
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 021: Allow the PATCH mock to clear nullable fields.
## Review Comment

`display_name`, `provider_config`, and `routing_policy` are nullable in the request shape, but `??` turns an explicit `null` into “keep the fixture value”. That makes the mock diverge from the API contract and hides bugs in stories that try to clear those fields.

## Triage

- Decision: `invalid`
- Notes: I traced this through the actual transport contract before changing the mock. `UpdateBridgeRequest` in `internal/api/contract/bridges.go` uses pointer fields, so JSON `null` and omission are both decoded as `nil` and are intentionally indistinguishable on the server. The generated client types reflect that: the request permits `null`, but the bridge response keeps `display_name` and `routing_policy` non-nullable. The existing Storybook mock behavior therefore matches the current backend semantics instead of diverging from them, so no code change is warranted for this issue.

## Resolution

- No source change. The review comment was invalid against the current backend contract.

## Verification

- Verified by source inspection against `internal/api/contract/bridges.go`, generated client types, and the final clean runs of `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
