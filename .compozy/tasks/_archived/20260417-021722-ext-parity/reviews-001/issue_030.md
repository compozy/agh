---
status: resolved
file: internal/config/agent_resource.go
line: 48
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:b5e127ab2a2c
review_hash: b5e127ab2a2c
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 030: Use errors.Join() to preserve the validation error in the unwrap chain.
## Review Comment

Line 49 wraps `resources.ErrValidation` with `%w`, but formats the validation error using `%v`, which removes it from the error chain. This breaks `errors.As()` for the original validation failure. Use `errors.Join()` instead to preserve both errors, consistent with the codebase's error handling pattern.

## Triage

- Decision: `VALID`
- Notes: `validateAgentResourceSpec` still wraps `resources.ErrValidation` with `%w` but formats the concrete validation failure with `%v`, which drops the underlying validation error from the unwrap chain. That breaks `errors.As` and loses detail for callers. The fix is to preserve both errors in the chain with `errors.Join` and extend the codec tests to assert both signals survive.
