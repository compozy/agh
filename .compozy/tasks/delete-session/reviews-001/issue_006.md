---
status: resolved
file: internal/api/udsapi/handlers_test.go
line: 969
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:a108dbe4458f
review_hash: a108dbe4458f
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 006: Consider adding t.Parallel() for test isolation.
## Review Comment

Other tests in this file use `t.Parallel()` for independent execution. Adding it here would be consistent with the codebase patterns.

## Triage

- Decision: `valid`
- Notes:
  The UDS delete-session handler test is independent and can safely run in parallel, but it currently does not use the repo's preferred subtest structure or `t.Parallel()`. I will wrap it in a `Should...` subtest and mark the test parallel.
