---
status: resolved
file: web/e2e/automation.spec.ts
line: 38
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:1558f2a4e64b
review_hash: 1558f2a4e64b
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 018: Align the test name with current behavior.
## Review Comment

The scenario no longer edits/saves automation; it validates prefilled form data and closes. Rename the test (or restore edit-save assertions) to keep intent explicit.

Also applies to: 93-98

## Triage

- Decision: `VALID`
- Notes:
  - The Playwright scenario opens the edit form to inspect prefilled values, closes it, and then triggers a run; it no longer edits and saves automation.
  - The fix is to rename the test so the name matches the actual inspect-and-trigger behavior.
