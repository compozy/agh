---
status: resolved
file: packages/ui/src/components/code-block.tsx
line: 35
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:5b6b5000632c
review_hash: 5b6b5000632c
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 009: Reset copy feedback duration on repeated copy clicks.
## Review Comment

On Line 47, clicking copy again while `copied` is already `true` does not retrigger the timeout in Lines 37-41, so the success state can clear too early relative to the latest click.

## Triage

- Decision: `valid`
- Reasoning: The copy feedback timeout is keyed off `copied`. Clicking copy while `copied` is already `true` does not change state, so the effect does not restart and the feedback can clear too early after the latest click.
- Root cause: Repeated copy actions do not reset the active timer.
- Fix plan: Manage the timer per copy action so every successful copy restarts the feedback window.

## Resolution

- Changed `packages/ui/src/components/code-block.tsx` to restart copy feedback on every successful copy via a dedicated feedback key.
- Added a regression in `packages/ui/src/components/code-block.test.tsx` because the existing companion test file lives outside the initial code-file list, and verified with `make verify`.
