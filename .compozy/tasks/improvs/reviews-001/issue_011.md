---
status: resolved
file: internal/observe/tasks.go
line: 861
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:56bdd7ad9ba4
review_hash: 56bdd7ad9ba4
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 011: Honor OriginKind when filtering audit-backed metrics.
## Review Comment

`TaskMetricsQuery.OriginKind` is applied to runs and events, but not to ingress audits. For any non-network origin view, `DuplicateIngressTotal` and `ChannelMismatchTotal` can still come back non-zero because this function keeps all network audit rows.

## Triage

- Decision: `VALID`
- Notes:
  `filterTaskIngressAudits` filters by time and channel but ignores
  `TaskMetricsQuery.OriginKind`. Since ingress audits are network-origin rows,
  non-network metric queries currently overcount duplicate ingress and channel
  mismatch totals. Plan: honor origin filtering for audits and add metrics
  coverage for a non-network query.
