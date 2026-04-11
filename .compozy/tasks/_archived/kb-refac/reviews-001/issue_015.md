---
status: resolved
file: internal/session/hooks.go
line: 65
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4090986708,nitpick_hash:07c62188c14c
review_hash: 07c62188c14c
source_review_id: "4090986708"
source_review_submitted_at: "2026-04-10T16:14:03Z"
---

# Issue 015: Consider adding compile-time interface verification.
## Review Comment

The accessor methods correctly return no-op implementations when fields are nil. Per coding guidelines, consider adding compile-time interface verification for the no-op types to catch signature mismatches early.

As per coding guidelines: "Use compile-time interface verification: `var _ Interface = (*Type)(nil)`".

## Triage

- Decision: `VALID`
- Reasoning: The no-op hook implementations satisfy the grouped interfaces by construction, but there is no compile-time verification to catch signature drift if one interface changes later.
- Fix approach: Add interface assertion lines for the no-op hook implementations so mismatches fail at compile time.
