---
status: resolved
file: internal/extension/manifest_test.go
line: 467
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:2a9c65fbf177
review_hash: 2a9c65fbf177
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 010: Please refactor this new validation test into t.Run("Should...") subtests.
## Review Comment

The assertions are useful, but this new case should follow the suite’s required table-driven/subtest pattern.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Root cause: the new bridge-manifest validation case is not following the repo-default table-driven / subtest structure.
- Fix approach: refactor it into `t.Run("Should...")` subtests while preserving the same assertions.
