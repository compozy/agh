---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/source_id.go
line: 12
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:6b6d27992a38
review_hash: 6b6d27992a38
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 022: Reject whitespace-padded source IDs.
## Review Comment

`ValidateSourceID` trims before validating, so values like `" builtin "` or `"extension:demo "` pass even though callers still retain the padded string. Since source IDs are used as exact identities, that lets a non-canonical value validate and then miss equality/map lookups later. Reject surrounding whitespace here, or expose a normalize-and-validate helper instead. Based on learnings "User-visible runtime capabilities in Go backend must expose stable machine-readable control surfaces."

## Triage

- Decision: `valid`
- Notes:
  - `ValidateSourceID` trims before validation, and `ValidateSourceIdentity` also trims before delegating, so whitespace-padded IDs still validate.
  - Source IDs are exact identities used in equality and map lookups, so accepting non-canonical padded input is a real correctness bug.
  - Fix plan: reject surrounding whitespace instead of silently normalizing it in the validation helpers.
