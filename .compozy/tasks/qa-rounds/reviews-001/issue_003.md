---
status: resolved
file: web/src/routes/_app/agents.$name.tsx
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:803039727d80
review_hash: 803039727d80
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 003: This still fetches agent detail data on nested session routes.
## Review Comment

`useAgentDetailPage(name)` runs before the child-route early return, so `/agents/$name/sessions/$id` still pays for the agent + sessions queries even though this component renders only `<Outlet />`. The clean fix is to move the detail shell into an index child route and let this parent route stay as a layout route.

## Triage

- Decision: `VALID`
- Notes:
  - `AgentDetailPage` currently calls `useAgentDetailPage(name)` before checking `useChildMatches()`, so nested session routes still start agent detail and session list queries even though the parent renders only `<Outlet />`.
  - The root cause is that the detail data hook lives in the layout component instead of in the detail-only rendering branch.
  - Fix by splitting the detail shell into a child component that is rendered only when there is no child match, preserving hook rules while preventing nested routes from paying for unused data. Add a route test proving child routes render the outlet without invoking `useAgentDetailPage`.
  - Resolution: split the detail shell into `AgentDetailContent`, leaving nested child routes to render `<Outlet />` without invoking the detail hook.
  - Verification: targeted Vitest passed; `make verify` passed.
