---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/httpapi/middleware_refac_test.go
line: 32
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:b2e5c0d2f399
review_hash: b2e5c0d2f399
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 009: Wrap each table entry in its own subtest.
## Review Comment

This loop flattens five scenarios into one failure domain, so the first mismatch aborts coverage for the rest and misses the repo’s required per-case `t.Run("Should ...")` structure.

As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures".

## Triage

- Decision: `VALID`
- Notes:
  `TestCanonicalHostNormalizesBoundHostPorts` collapses multiple cases into one failure domain inside a single subtest. Splitting the table rows into per-case `t.Run` subtests matches repo test shape and keeps failures isolated.
