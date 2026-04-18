---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1214
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:3fc0844511aa
review_hash: 3fc0844511aa
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 012: Prefer Should... subtests for these new route flows.
## Review Comment

Both additions are meaningful end-to-end scenarios, but they’re still broad single-path tests. Breaking the route/assertion sets into `t.Run("Should...")` cases would improve failure isolation and match the repo’s expected test layout.

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `invalid`
- Notes: The added integration coverage already verifies a concrete publish/run/live round-trip against the real transport. Breaking it into additional `Should...` subtests would change structure only, not fix a missing assertion or reliability issue in this batch.
