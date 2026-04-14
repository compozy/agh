---
status: pending
file: web/src/systems/network/components/network-peers-list-panel.tsx
line: 66
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:062d07c23a7d
review_hash: 062d07c23a7d
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 025: Let the panel render loading and error states too.
## Review Comment

Right now this component only knows how to render a list or an empty state, so routes have to branch around it for loading/failure. That pushes state ownership out of the component and conflicts with the UI-state rule for `web/` components.

As per coding guidelines, "Handle all loading, error, and empty states in components — never assume `data` exists".

## Triage

- Decision: `UNREVIEWED`
- Notes:
