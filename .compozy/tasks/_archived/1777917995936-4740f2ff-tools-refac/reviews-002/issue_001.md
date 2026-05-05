---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: resolved
file: internal/api/core/tools.go
line: 288
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4206460462,nitpick_hash:9533cf3daca2
review_hash: 9533cf3daca2
source_review_id: "4206460462"
source_review_submitted_at: "2026-04-30T15:34:55Z"
---

# Issue 001: Add doc comments for the new unexported helpers.
## Review Comment

This introduces a large block of unexported conversion/scope/error helpers without comments, which makes the projection and error-mapping path harder to audit and violates the repo's Go comment policy.

As per coding guidelines, "Comments in Go must explain the 'why' and 'what', not just 'what'. Unexported identifiers must have a comment".

## Triage

- Decision: `VALID`
- Notes:
  The DTO conversion and error-mapping helpers in `internal/api/core/tools.go`
  are part of the public API projection boundary. Their names describe the
  mechanical conversion, but they do not explain the boundary invariant: copy
  registry-owned slices/raw messages, preserve stable transport error payloads,
  and avoid leaking backend details. Add concise comments that document those
  invariants without changing behavior.
