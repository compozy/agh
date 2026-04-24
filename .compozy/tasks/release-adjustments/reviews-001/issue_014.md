---
status: resolved
file: internal/session/prompt_activity.go
line: 50
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:b43c36abf652
review_hash: b43c36abf652
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 014: Reconsider context.Background() fallback.
## Review Comment

The coding guidelines specify avoiding `context.Background()` outside `main` and focused tests. While this is defensive, consider requiring a non-nil context from callers instead.

As per coding guidelines, "Include `context.Context` as first argument to functions crossing runtime boundaries — avoid `context.Background()` outside `main` and focused tests".

## Triage

- Decision: `VALID`
- Notes:
  - `newPromptActivitySupervisor` defensively replaces a nil context with `context.Background()`, even though the public prompt path already rejects nil contexts before calling it.
  - The fix is to remove the fallback and rely on the validated caller context instead of hiding a bad internal call.
