---
status: resolved
file: web/src/systems/network/mocks/network-mocks.test.ts
line: 48
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:e0bed60b6232
review_hash: e0bed60b6232
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 033: Consider adding a failure-path contract test for required fields.
## Review Comment

You already cover the happy path; adding one 400-case assertion would lock down the new validation behavior.

## Triage

- Decision: `valid`
- Notes: The send mock currently has only happy-path coverage. After hardening the handler's required-field parsing, add a 400-path contract test that sends malformed required fields and asserts the mock returns the intended validation response.
