---
status: resolved
file: web/src/hooks/routes/use-session-page-controls.test.tsx
line: 137
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167261241,nitpick_hash:095ae13675d7
review_hash: 095ae13675d7
source_review_id: "4167261241"
source_review_submitted_at: "2026-04-24T01:30:33Z"
---

# Issue 007: Strengthen the idle delete test by executing callback side effects.
## Review Comment

Right now it only verifies callback presence. Executing captured `onSuccess`/`onError` would directly validate reset + toast + optional `onDeleteSuccess` behavior.

## Triage

- Decision: `valid`
- Notes:
  - The idle-delete test in `web/src/hooks/routes/use-session-page-controls.test.tsx` only checks that `mutate` receives callbacks, but it never executes `onSuccess` or `onError`.
  - That leaves the reset, toast, and optional `onDeleteSuccess` behavior unverified even though the hook owns those effects.
  - Planned fix: capture the mutation options, invoke both callbacks, and assert the expected side effects.

## Resolution

- Strengthened the idle-delete hook test to execute the captured `onSuccess` and `onError` callbacks instead of only asserting their presence.
- The test now directly verifies reset behavior, success and error toasts, and the optional `onDeleteSuccess` callback contract owned by the hook.
- Verified with `make verify`, `make web-lint`, and `make web-typecheck` (all exit `0`).
