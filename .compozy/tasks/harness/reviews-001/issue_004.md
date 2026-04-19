---
status: resolved
file: internal/api/httpapi/stream_helpers_test.go
line: 187
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:ca254fccc3d3
review_hash: ca254fccc3d3
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 004: Assert that session_id is forwarded into the observer query.
## Review Comment

This test is meant to cover `/api/observe/events/stream?session_id=sess-harness`, but the fake drops the `store.EventSummaryQuery` argument entirely. If the handler stopped applying that filter, this test would still pass because the stub always returns the same event. Capture the query and assert its session filter before returning the summary.

## Triage

- Decision: `valid`
- Notes:
  - The test exercises `/api/observe/events/stream?session_id=sess-harness` but the stubbed observer currently ignores the `store.EventSummaryQuery` argument.
  - That means the test would still pass even if the handler stopped forwarding the session filter into the observer query.
  - I will capture the query in the fake and assert the expected `SessionID` value before returning the summary payload.
