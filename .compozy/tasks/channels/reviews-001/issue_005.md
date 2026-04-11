---
status: resolved
file: internal/api/core/channels_test.go
line: 128
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:32e7f65412b0
review_hash: 32e7f65412b0
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 005: Wrap table cases in t.Run for better isolation and debugging.
## Review Comment

The table-driven test iterates directly without using `t.Run`. This makes it harder to identify which case failed and prevents running subtests independently.

As per coding guidelines: "Use table-driven tests with subtests (t.Run) as default" and "MUST use t.Run("Should...") pattern for ALL test cases".

---

## Triage

- Decision: `valid`
- Notes:
  - The lifecycle transition test currently loops inline without `t.Run`, which weakens failure isolation and does not follow the repository’s test convention.
  - I will convert the table to named `t.Run("Should ...")` subtests and keep the assertions per case.
  - Resolution: Converted the lifecycle table to `Should ...` subtests in [internal/api/core/channels_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/core/channels_test.go:113); verified with `make verify`.
