---
status: resolved
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

- Decision: `valid`
- Root cause: `NetworkPeersListPanel` only knows how to render loaded peers or an empty state, so `web/src/routes/_app/network.tsx` handles peers-list loading/failure with page-level early returns. That duplicates UI-state ownership outside the component and conflicts with the `web/` component-state rule.
- Fix approach: add loading/error support to the peers panel and update the network route/tests so the peers list renders those states in-panel instead of replacing the whole page. This also requires a minimal route/test touch outside the batch code-file list because the current caller wiring is part of the defect.
- Resolution: added in-panel loading/error states to `NetworkPeersListPanel` and updated the route so peers loading/failure is rendered inside the shell instead of short-circuiting the page.
- Verification: added peers-panel route coverage and passed the focused Vitest run, `make web-lint`, `make web-typecheck`, and `make verify`.
