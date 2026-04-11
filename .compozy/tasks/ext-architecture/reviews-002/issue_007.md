---
status: resolved
file: internal/api/spec/spec_test.go
line: 17
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092845293,nitpick_hash:6b96c72a44fc
review_hash: 6b96c72a44fc
source_review_id: "4092845293"
source_review_submitted_at: "2026-04-10T23:04:43Z"
---

# Issue 007: Add t.Parallel() inside each independent subtest.
## Review Comment

The subtests under `TestDocumentTracksRequiredFieldsAndEnums` are read-only checks and can run concurrently. Adding `t.Parallel()` inside each `t.Run` keeps this file aligned with repo test conventions and speeds execution.

As per coding guidelines, "Use `t.Parallel()` for independent subtests in Go tests."

---

## Triage

- Decision: `valid`
- Notes:
  - The finding is accurate. The subtests under `TestDocumentTracksRequiredFieldsAndEnums` are read-only schema assertions and can run independently.
  - Root cause: the file sets `t.Parallel()` at the parent level but leaves the individual subtests serialized.
  - Fix approach: add `t.Parallel()` inside each independent subtest while keeping the shared document setup at the parent level.
  - Resolution: implemented in `internal/api/spec/spec_test.go` and verified with focused package tests plus `make verify`.
