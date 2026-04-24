---
status: resolved
file: web/src/hooks/routes/use-session-page-controls.ts
line: 50
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167261241,nitpick_hash:8484cc96e3e3
review_hash: 8484cc96e3e3
source_review_id: "4167261241"
source_review_submitted_at: "2026-04-24T01:30:33Z"
---

# Issue 008: Consider applying controlsBusy guard to stop/resume for full control serialization.
## Review Comment

`handleDelete`/`handleClear` are protected, but `handleStop`/`handleResume` can still fire while another control action is pending. Guarding them too keeps mutation concurrency policy consistent.

## Triage

- Decision: `valid`
- Notes:
  - `web/src/hooks/routes/use-session-page-controls.ts` serializes delete and clear behind `controlsBusy`, but `handleStop` and `handleResume` do not apply the same guard.
  - This is a real orchestration gap because the hook can still dispatch stop/resume while other control mutations are pending, and the header component does not know about `clearMutation.isPending`.
  - Planned fix: guard stop/resume in the hook with the same busy-state policy and add regression coverage in the hook test file.

## Resolution

- Added the same `controlsBusy` guard used by delete and clear to both `handleStop` and `handleResume` in `web/src/hooks/routes/use-session-page-controls.ts`.
- Added regression coverage proving stop and resume are suppressed while another control mutation is pending.
- Verified with `make verify`, `make web-lint`, and `make web-typecheck` (all exit `0`).
