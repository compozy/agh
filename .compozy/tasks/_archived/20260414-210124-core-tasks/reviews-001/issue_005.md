---
status: resolved
file: internal/api/core/tasks.go
line: 1122
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:5dddb7599a61
review_hash: 5dddb7599a61
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 005: Consider nil slice handling for cloned raw message.
## Review Comment

The `cloneRawMessage` function (called on line 1126) should handle the case where the source `json.RawMessage` is an empty slice vs nil. Currently, if `*source` is `[]byte{}`, it will be passed to `cloneRawMessage` which returns `nil` for empty slices. Verify this is the intended behavior for JSON semantics (empty vs absent).

---

## Triage

- Decision: `invalid`
- Root cause check: `cloneRawMessage` intentionally normalizes zero-length `json.RawMessage` values to `nil`.
- Why invalid: a zero-length raw message is not valid JSON, and for these `omitempty` payload fields the API does not rely on distinguishing `[]byte{}` from `nil`. Preserving an empty byte slice would only risk emitting invalid/ambiguous payload state.

## Resolution

- No code change was required because the current normalization is intentional and matches the API's JSON semantics.
- The batch still passed the final `make verify` run unchanged for this issue.
