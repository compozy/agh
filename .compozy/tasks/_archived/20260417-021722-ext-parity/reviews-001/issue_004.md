---
status: resolved
file: extensions/bridges/teams/provider_test.go
line: 1255
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:bcca49a3ba22
review_hash: bcca49a3ba22
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 004: Wait for ready state in the polling predicate to reduce flakiness.
## Review Comment

This currently waits for “any state” and asserts readiness afterward. If an intermediate non-ready state is written first, the test can fail intermittently.

## Triage

- Decision: `VALID`
- Notes: The current predicate returns as soon as any state marker exists, but the assertion checks readiness afterward. If the file first contains a non-ready marker and later appends a ready marker, this test can observe the intermediate state and fail. The fix is to wait for a ready marker in the predicate.
