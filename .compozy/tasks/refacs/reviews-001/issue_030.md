---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/cli/bridge_test.go
line: 152
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:231a73904090
review_hash: 231a73904090
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 030: Use a t.Run("Should ...") subtest for the new status-flag rejection case.
## Review Comment

Line 152 adds a flat test case; this repo’s test convention prefers `Should ...` subtests for individual scenarios.

As per coding guidelines, `**/*_test.go`: Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures.

## Triage

- Decision: `valid`
- Root cause: `TestBridgeCreateRejectsOperationalStatusFlag` is still a flat test case rather than a `Should ...` subtest, which does not match the repository test shape used for new scenarios.
- Fix plan: wrap the scenario in a named subtest without changing the validation logic.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
