---
status: pending
file: web/src/systems/workspace/adapters/workspace-api.ts
line: 30
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:a3b38dcd1b6f
review_hash: a3b38dcd1b6f
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 029: Consider using a typed error class for consistency with other adapters.
## Review Comment

Per coding guidelines, API adapters should use typed error classes (e.g., `WorkspaceApiError`) rather than raw `Error`. While this follows the existing pattern in the file, newer adapters like `bridges-api.ts` use `BridgesApiError` which allows consumers to inspect `status` for error handling.

This is a pre-existing pattern, so deferring to a follow-up refactor is reasonable.

As per coding guidelines: "Use typed error classes in API adapters — never throw raw errors"

## Triage

- Decision: `UNREVIEWED`
- Notes:
