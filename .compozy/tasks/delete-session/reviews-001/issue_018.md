---
status: resolved
file: web/src/systems/tasks/hooks/use-task-actions.ts
line: 155
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:59fea0c87854
review_hash: 59fea0c87854
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 018: Minor redundancy in cache operations.
## Review Comment

`removeQueries` correctly purges the deleted task's detail from cache. However, `invalidateTaskQueries(queryClient, id)` on line 164 will also call `invalidateQueries({ queryKey: tasksKeys.detail(id) })` (see line 110), which is redundant after the query has already been removed.

This is harmless but slightly wasteful. Consider either:
1. Not passing `id` to `invalidateTaskQueries` since the detail query is already removed
2. Or keeping as-is for consistency with other mutation hooks

## Triage

- Decision: `invalid`
- Notes:
  This is an optional micro-optimization, not a correctness issue. `removeQueries(tasksKeys.detail(id))` followed by the existing shared invalidation helper is harmless and keeps the delete hook behavior consistent with the rest of the task mutation hooks. I will leave the code as-is and resolve this issue as analysis-only.
