---
status: resolved
file: internal/extension/host_api_test.go
line: 209
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:5690e7191663
review_hash: 5690e7191663
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 003: Use a t.Run("Should...") subtest wrapper for this new scenario.
## Review Comment

The scenario is valid, but the repository test convention expects explicit `Should...` subtests.

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: The new bridge-session scenario is a top-level test body, while this repository expects `t.Run("Should...")` wrappers for individual scenarios. I will wrap the case in a `Should...` subtest without changing its assertions.
