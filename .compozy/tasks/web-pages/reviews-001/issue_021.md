---
status: pending
file: web/src/systems/network/components/network-channels-list-panel.tsx
line: 50
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:4830d59f7aba
review_hash: 4830d59f7aba
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 021: Add loading and error states to the panel API.
## Review Comment

This component can only render a filled list or an empty state, so routes have to special-case loading and failure outside the panel. That breaks the “component owns all states” rule for UI components in `web/`.

As per coding guidelines, "Handle all loading, error, and empty states in components — never assume `data` exists".

## Triage

- Decision: `UNREVIEWED`
- Notes:
