---
status: pending
file: internal/registry/github/client.go
line: 688
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107316563,nitpick_hash:ecb6eb082800
review_hash: ecb6eb082800
source_review_id: "4107316563"
source_review_submitted_at: "2026-04-14T15:47:27Z"
---

# Issue 010: Good pattern: closeResponseBody properly wraps close errors.
## Review Comment

This helper correctly handles and wraps close errors. Consider using this helper consistently throughout the file instead of `_ = response.Body.Close()` to maintain uniform error handling.

The codebase already has `closeResponseBody` that properly handles close errors. Lines 272, 465, and 501 should use this helper or similar pattern for consistency.

## Triage

- Decision: `UNREVIEWED`
- Notes:
