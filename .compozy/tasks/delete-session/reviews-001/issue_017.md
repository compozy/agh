---
status: resolved
file: web/src/systems/tasks/adapters/tasks-api.test.ts
line: 261
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:52be5e2667c2
review_hash: 52be5e2667c2
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 017: Strengthen the 404 test to assert the typed error contract.
## Review Comment

Right now it only checks the message. Also assert `TasksApiError` so a regression to raw `Error` is caught.

## Triage

- Decision: `valid`
- Notes:
  The delete-task 404 adapter test only checks the message string, so it would not catch a regression where the code starts throwing a generic `Error` instead of `TasksApiError`. I will strengthen the assertion to check both the typed error contract and the message.
