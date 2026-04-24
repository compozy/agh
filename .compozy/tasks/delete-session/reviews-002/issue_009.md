---
status: resolved
file: web/src/systems/session/hooks/use-session-actions.test.tsx
line: 170
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167261241,nitpick_hash:f40554fb62a7
review_hash: f40554fb62a7
source_review_id: "4167261241"
source_review_submitted_at: "2026-04-24T01:30:33Z"
---

# Issue 009: Also assert list-query invalidation on delete success.
## Review Comment

This test validates entity cache removal and draft clearing, but it doesn’t assert the `sessionKeys.lists()` invalidation contract from `useDeleteSession`, which is an important regression guard.

## Triage

- Decision: `valid`
- Notes:
  - `useDeleteSession` explicitly invalidates `sessionKeys.lists()` on success, but `web/src/systems/session/hooks/use-session-actions.test.tsx` only asserts cache removal and draft clearing.
  - Missing the list-invalidation assertion leaves an important contract unguarded.
  - Planned fix: spy on `queryClient.invalidateQueries` in the delete-success test and assert the list-key invalidation call.

## Resolution

- Extended the delete-success test in `web/src/systems/session/hooks/use-session-actions.test.tsx` to assert the `sessionKeys.lists()` invalidation contract.
- The test now guards cache removal, draft clearing, and list refresh behavior together, which closes the missing regression check.
- Verified with `make verify`, `make web-lint`, and `make web-typecheck` (all exit `0`).
