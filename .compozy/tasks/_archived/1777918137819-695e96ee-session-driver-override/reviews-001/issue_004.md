---
status: resolved
file: internal/config/provider_test.go
line: 313
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:b6c960fd80cc
review_hash: b6c960fd80cc
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 004: Use t.Run("Should...") subtests for newly added cases.
## Review Comment

The new tests are valid functionally, but they don’t follow the repository’s required subtest naming/pattern convention.

As per coding guidelines, "MUST use t.Run(\"Should...\") pattern for ALL test cases".

Also applies to: 354-452, 454-479

## Triage

- Decision: `valid`
- Root cause: the newly added `ResolveSessionAgent` tests are separate top-level cases instead of the repo-standard `t.Run("Should ...")` subtests.
- Fix plan: consolidate the new session-agent coverage into a parent test with scenario subtests that preserve the same assertions.
