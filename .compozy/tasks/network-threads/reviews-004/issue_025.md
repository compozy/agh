---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/audit_test.go
line: 213
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:0e7a986fa2d0
review_hash: 0e7a986fa2d0
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 025: Parallelize this new top-level test.
## Review Comment

Everything here is per-test state, so leaving the top-level case serialized just slows the package and breaks the default parallel-test convention already used elsewhere in this file.

As per coding guidelines, `**/*_test.go`: "Default to `t.Parallel` in Go tests unless there is a specific reason to disable it (opt-out with `t.Setenv`)."

## Triage

- Decision: `VALID`
- Root cause: `TestAuditWriterRecordsRepeatedGreetHeartbeatsOnlyAsAuditRows` does not use `t.Parallel()` even though the test owns all of its state and does not call `t.Setenv`, so it diverges from the repo’s default parallel test convention.
- Fix approach: mark the top-level test parallel and keep the existing subtest behavior intact.
- Verification: fixed in scoped code and validated with fresh `make verify`.
