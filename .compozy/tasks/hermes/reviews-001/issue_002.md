---
status: resolved
file: internal/acp/types.go
line: 516
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:57bc6d62fbcf
review_hash: 57bc6d62fbcf
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 002: Wrap checkpoint error with operation context per coding guidelines.
## Review Comment

Line 520 returns the underlying error directly without wrapped context. Use `fmt.Errorf()` to add diagnostic information consistent with other error returns in this package.

## Triage

- Decision: `VALID`
- Notes: `checkpointProcessOwner` returns the raw `toolruntime.Handle.Checkpoint` error, unlike the rest of the ACP package. Wrapping the error preserves `errors.Is`/`errors.As` behavior while adding the missing operation context for logs and diagnostics.
