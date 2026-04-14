---
status: resolved
file: internal/api/core/tasks.go
line: 1077
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:c7253e5eaa27
review_hash: c7253e5eaa27
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 004: In-memory filtering may not scale well for large task run sets.
## Review Comment

The `filterTaskRuns` function loads all runs from the view and filters in-memory. For tasks with many runs, this could impact performance. Consider whether the `ListTaskRuns` endpoint should delegate filtering to the store layer with proper SQL clauses.

## Triage

- Decision: `invalid`
- Root cause check: this is a performance suggestion, not a correctness defect in the current implementation.
- Why invalid: `ListTaskRuns` filters the already-loaded task view in memory and respects `status`, `session_id`, and `limit` correctly. Pushing those filters into the store would require a broader API redesign without a failing behavior to fix in this batch.

## Resolution

- No localized correctness fix was warranted for this batch because the comment describes a broader performance redesign, not a failing behavior.
- The batch still passed the final `make verify` run unchanged for this issue.
