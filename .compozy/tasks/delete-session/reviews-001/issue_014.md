---
status: resolved
file: web/src/hooks/routes/use-session-page-controls.ts
line: 63
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:9d369e2886c9
review_hash: 9d369e2886c9
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 014: Harden handleDelete against concurrent control mutations.
## Review Comment

`handleDelete` currently blocks only duplicate deletes. Consider guarding against other in-flight control mutations too (`isStopping`, `isResuming`, `isClearing`, `isCancellingPrompt`) to avoid overlapping operations through non-UI call paths.

Also applies to: 92-97

## Triage

- Decision: `valid`
- Notes:
  `handleDelete` and `handleClear` only guard against their own mutation being pending. The UI disables these paths via derived state, but the callback functions themselves still allow overlapping operations if they are invoked programmatically while another control mutation is active. I will harden both callbacks against all in-flight control mutations and add a focused hook test in `web/src/hooks/routes/use-session-page-controls.test.tsx` because the current scoped files do not include route-hook coverage.
