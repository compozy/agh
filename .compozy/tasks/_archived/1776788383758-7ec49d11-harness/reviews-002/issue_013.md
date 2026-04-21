---
status: resolved
file: internal/session/manager_prompt.go
line: 265
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135285786,nitpick_hash:8b8f8f21f276
review_hash: 8b8f8f21f276
source_review_id: "4135285786"
source_review_submitted_at: "2026-04-18T23:45:16Z"
---

# Issue 013: Consider extracting clonePromptSyntheticMeta to a shared location.
## Review Comment

This helper function is duplicated in both `manager_prompt.go` (lines 265-275) and `transcript.go` (lines 824-834). Consider moving it to `internal/acp/types.go` alongside the `PromptSyntheticMeta` type definition to avoid duplication.

Also applies to: 824-834

## Triage

- Decision: `invalid`
- Notes:
  - This is a small duplication cleanup suggestion, not a correctness defect in `internal/session/manager_prompt.go`.
  - The requested extraction would require widening shared ACP surface area and editing additional out-of-scope files (`internal/transcript/transcript.go` and likely `internal/acp/types.go`) for no behavior change.
  - The current helper is intentionally local, tiny, and package-private in both call sites; keeping it local avoids coupling unrelated packages to an extra shared utility.
