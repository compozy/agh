---
status: resolved
file: internal/store/globaldb/global_db_test.go
line: 1277
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4132976935,nitpick_hash:4c1017b72a7e
review_hash: 4c1017b72a7e
source_review_id: "4132976935"
source_review_submitted_at: "2026-04-18T00:19:15Z"
---

# Issue 012: Please cover the Limit > 0 branch too.
## Review Comment

`ListEventSummaries` now has separate SQL for limited vs. unlimited merged results, but this test only exercises the unlimited path. Add a mixed-stream case with `EventSummaryQuery{Limit: ...}` so the new ordering logic across `event_summaries` and `memory_operation_log` is locked down as well.

As per coding guidelines, Focus on critical paths: workflow execution, state management, error handling.

## Triage

- Decision: `valid`
- Notes:
  - The existing test only exercises the unlimited merged-query branch in `ListEventSummaries`.
  - The limited path uses different SQL shape and ordering, so I will add a mixed event/memory-operation case with `Limit > 0` to lock that behavior down explicitly.

## Resolution

- Extended the global DB summary test with a mixed session-event and memory-operation stream using `EventSummaryQuery{Limit: 2}`.
- Verified the limited query returns the most recent merged rows while preserving ascending output order for the selected window.
