---
status: resolved
file: internal/store/globaldb/global_db_task.go
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:a79cdb1d5641
review_hash: a79cdb1d5641
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 023: Potential performance concern with correlated subquery ordering.
## Review Comment

The `taskListOrderByActivitySQL` uses a correlated subquery that scans `task_runs` and `task_events` for each task row. For large datasets, this could become slow. Consider whether a denormalized `last_activity_at` column on `tasks` (updated via triggers or application logic) would be warranted if performance becomes an issue.

For now, this is acceptable for the expected scale, but monitor query times as the task table grows.

---

## Triage

- Decision: `invalid`
- Reasoning: this is an advisory performance note, not a concrete defect in the current change set. The correlated activity-ordering query is intentional for the current alpha-scale task volume, and the review comment itself says the current implementation is acceptable.
- Reasoning: introducing denormalized activity columns or trigger/application-maintained shadow state would be architectural scope creep for this batch and is not required to preserve correctness.

## Resolution

- Closed as `invalid`.
- No code change was made because the comment identifies a future optimization tradeoff rather than a correctness or regression issue in the scoped batch.
