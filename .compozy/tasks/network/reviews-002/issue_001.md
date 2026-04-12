---
status: resolved
file: internal/acp/client.go
line: 279
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4094005443,nitpick_hash:2c9de93a8fe4
review_hash: 2c9de93a8fe4
source_review_id: "4094005443"
source_review_submitted_at: "2026-04-11T17:12:00Z"
---

# Issue 001: Consider adding debug logging for observability.
## Review Comment

`applySessionMode` silently returns `nil` when no preferred mode is found or when preconditions fail. While not incorrect, adding debug-level logging would aid troubleshooting when session mode negotiation doesn't behave as expected.

---

## Triage

- Decision: `invalid`
- Notes: `applySessionMode` already treats missing ACP mode support as a benign no-op, and the nil precondition guard is an internal safety check rather than a user-facing failure path. Adding debug logs here would increase noise on normal agent startup without addressing a correctness bug or broken workflow.
