---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/udsapi/bridges_test.go
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:d94a37bebc9a
review_hash: d94a37bebc9a
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 018: Wrap TestCreateBridgeHandlerReturnsPersistedPayload in a t.Run("Should ...") subtest.
## Review Comment

As per coding guidelines, "Use `t.Run('Should ...')` subtests with `t.Parallel` as default" and "MUST use `t.Run('Should ...')` pattern for ALL test cases." Since line 56 was modified, this is a good opportunity to align with the mandate.

## Triage

- Decision: `VALID`
- Notes:
  The touched UDS bridge test is still a top-level body without a `t.Run("Should ...")` wrapper. Wrapping it aligns the file with the repo’s required subtest structure without changing behavior.
