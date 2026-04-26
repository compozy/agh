---
status: resolved
file: internal/automation/schedule_test.go
line: 183
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:bee851f74c9c
review_hash: bee851f74c9c
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 004: Use the required subtest pattern for this new scheduler case.
## Review Comment

This coverage was added as a standalone top-level test. Please fold it under a table-driven parent with `t.Run("Should...")` so new fire-limit scheduler scenarios can extend the same harness cleanly.

As per coding guidelines, "`**/*_test.go`: Table-driven tests with subtests (t.Run) as default." and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  - `TestSchedulerDefersNextRunAfterFireLimit` is still a standalone top-level test case.
  - Root cause: the new scheduler scenario was added outside the file's expected `t.Run("Should...")` structure.
  - Fix plan: fold the existing assertions into a named subtest and keep the scheduler harness unchanged.
