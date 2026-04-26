---
status: resolved
file: internal/automation/dispatch_test.go
line: 563
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:891a23f7f73b
review_hash: 891a23f7f73b
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 003: Use the required subtest shape for this new fire-limit case.
## Review Comment

This new scenario is another standalone top-level test. Please move it under a table-driven parent with `t.Run("Should...")` to stay consistent with the repo’s Go test pattern.

As per coding guidelines, "`**/*_test.go`: Table-driven tests with subtests (t.Run) as default." and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  - `TestDispatchScheduledReservedRunCancelsOnFireLimit` is currently a standalone top-level test rather than a named subtest scenario.
  - Root cause: the new fire-limit regression was added directly to the file instead of using the repo's default subtest pattern.
  - Fix plan: wrap the scenario in a `t.Run("Should...")` block while preserving the fire-limit assertions and any added regression coverage tied to issue 002.
