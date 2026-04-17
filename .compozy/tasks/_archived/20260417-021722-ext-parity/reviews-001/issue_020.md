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

# Issue 020: Replace the polling sleeps in the new wait helpers.
## Review Comment

These helpers now gate several resource-projection assertions, but fixed `time.Sleep()` polling still makes the suite flaky on slow CI and slower than necessary on fast runs. Expose a reconciliation/projector signal and wait on that instead of sleeping.

As per coding guidelines, `Never use time.Sleep() in orchestration — use proper synchronization primitives`.

## Triage

- Decision: `INVALID`
- Notes: The helpers are polling externally observable HTTP state in an integration test, and there is no existing reconciliation/projector signal exposed through the scoped runtime to replace that polling. Fixing this as suggested would require introducing new production synchronization surfaces solely for tests. That is out of proportion for this batch and not backed by a concrete failing behavior in the current suite.
