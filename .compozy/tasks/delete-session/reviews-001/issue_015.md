---
status: resolved
file: web/src/systems/session/hooks/use-session-actions.test.tsx
line: 170
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:cc1a31cf1c40
review_hash: cc1a31cf1c40
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 015: Add a delete-failure test case to lock cache behavior.
## Review Comment

This test is good for the success path. Please add a failure-path case (`deleteSession` rejects) to assert caches/drafts are not incorrectly cleared when deletion fails.

## Triage

- Decision: `valid`
- Notes:
  The success-path delete hook test does not cover the regression described in issue 016: failed deletes should not clear drafts or remove cached session data. I will add a failure-path test that keeps the detail/history/transcript/events caches and draft intact when `deleteSession` rejects.
