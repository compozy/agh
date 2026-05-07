---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/api/core/session_workspace_internal_test.go
line: 43
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:b2961d7ef62c
review_hash: b2961d7ef62c
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 006: Rename subtest to the required Should ... form.
## Review Comment

Line 43 currently uses `"validate create session runtime overrides"`; please rename to `Should ...` to match enforced test naming conventions.

As per coding guidelines `**/*_test.go`: Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures.

## Triage

- Decision: `invalid`
- Notes:
  - The relevant subtest in `internal/api/core/session_workspace_internal_test.go` is already named `Should validate create session runtime overrides`.
  - The non-`Should ...` name referenced by the review is no longer present.
  - No code change is needed.
  - Resolved as invalid after branch inspection and full verification.
