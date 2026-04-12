---
status: resolved
file: internal/session/manager_helpers.go
line: 128
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857291,nitpick_hash:d06e1c78889a
review_hash: d06e1c78889a
source_review_id: "4093857291"
source_review_submitted_at: "2026-04-11T14:15:44Z"
---

# Issue 021: Wrap errors with session context for consistency.
## Review Comment

Per coding guidelines, errors should be wrapped with context. While `joinNetworkPeer` error is wrapped at the call site (line 91), wrapping in the helpers ensures all callers receive consistent context.

Also applies to: 149-149

## Triage

- Decision: `invalid`
- Notes: Both current call sites already wrap join/leave failures with session-specific context (`activateAndWatch` and `stopLocked`), so adding more wrapping inside the thin helpers would only duplicate error text for existing behavior. There is no uncovered caller today that loses the session context.
