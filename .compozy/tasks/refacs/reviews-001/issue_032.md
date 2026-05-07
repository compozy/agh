---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/cli/daemon_wait_test.go
line: 41
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:7060901a68d9
review_hash: 7060901a68d9
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 032: Wrap modified tests in t.Run("Should ...") subtests per project guidelines.
## Review Comment

`TestWaitForDaemonStartReturnsStatusWhenDaemonBecomesReady` (lines 41–61) and `TestRunDaemonDetachedReturnsReadyStatus` (lines 260–286) both have changed lines in this PR but still lack the required `t.Run("Should ...")` subtest wrapper. As per coding guidelines, "MUST use `t.Run('Should ...')` pattern for ALL test cases."

As per coding guidelines, "Use `agh-test-conventions` skill before writing or editing any `*_test.go` file. Covers: `t.Run('Should ...')` subtests."

Also applies to: 260-286

## Triage

- Decision: `valid`
- Root cause: the touched daemon wait tests at lines 41-61 and 260-286 remain top-level flat cases instead of `Should ...` subtests, which violates the active AGH test convention.
- Fix plan: wrap both scenarios in named subtests and keep the existing assertions intact.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
