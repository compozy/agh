---
status: resolved
file: web/src/components/app-sidebar.test.tsx
line: 263
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:ad1c523077c7
review_hash: ad1c523077c7
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 001: Make this route mock param-aware.
## Review Comment

The current stub keys only on `to`, so setting `matchedRouteFuzzy["/agents/$name"] = true` would mark every agent row active. That makes this assertion non-discriminating for the `name` param handling you want to cover.

## Triage

- Decision: `VALID`
- Notes:
  - The `useMatchRoute` test stub only keys route activity by `to`, while `AppSidebar` calls `matchRoute({ to: "/agents/$name", params: { name: agent.name }, fuzzy: true })`.
  - Setting a fuzzy match for `"/agents/$name"` would therefore mark every agent row active and fail to prove that the `name` param is respected.
  - Fix by making the test mock key on `to + params` and updating the active-row assertion to render multiple agents, with only the matching param active.
  - Resolution: implemented param-aware route match keys and asserted that only the matching agent row is active.
  - Verification: targeted Vitest passed; `make verify` passed.
