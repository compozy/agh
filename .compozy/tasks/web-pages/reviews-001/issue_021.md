---
status: resolved
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

- Decision: `valid`
- Root cause: `NetworkChannelsListPanel` only accepts a loaded channel array, so `web/src/routes/_app/network.tsx` has to short-circuit the entire page for channel-list loading and failure. That violates the `web/` rule that components should own loading/error/empty rendering for the data they present.
- Fix approach: extend the panel API to accept loading and error state, render those states inside the panel under the search control, and update the network route/tests to stop handling the channel-list query with page-level early returns. This requires a minimal route/test touch outside the batch code-file list because the current caller behavior is the actual source of the problem.
- Resolution: added in-panel loading/error states to `NetworkChannelsListPanel` and rewired `web/src/routes/_app/network.tsx` so the page shell stays mounted while the panel/detail panes render truthful loading or error UI.
- Verification: `bun x vitest run web/src/routes/_app/-network.test.tsx ...`, `make web-lint`, `make web-typecheck`, and `make verify` all passed.
