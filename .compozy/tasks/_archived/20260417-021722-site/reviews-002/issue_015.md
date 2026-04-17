---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 1265
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:6f5cf1ae828f
review_hash: 6f5cf1ae828f
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 015: Replace the polling sleeps in the new wait helpers.
## Review Comment

These helpers now gate several resource-projection assertions, but fixed `time.Sleep()` polling still makes the suite flaky on slow CI and slower than necessary on fast runs. Expose a reconciliation/projector signal and wait on that instead of sleeping.

As per coding guidelines, `Never use time.Sleep() in orchestration — use proper synchronization primitives`.

## Triage

- Decision: `INVALID`
- Reason: The current `internal/api/udsapi/udsapi_integration_test.go` does not contain the sleep-based wait helpers described in the review comment. There is no actionable `time.Sleep()` polling in this file at the cited location, so the note is stale against earlier test code.

## Resolution

- Analysis complete. No code change was required because the cited `time.Sleep()` orchestration pattern is not present in the current test file.
