---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/store/globaldb/global_db_task_projection.go
line: 101
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Lrn,comment:PRRC_kwDOR5y4QM6-UJdp
---

# Issue 029: _⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_

**Make current-run projection writes atomic.**

`setTaskCurrentRunProjection` does a read of `current_run_id` and then an unconditional `UPDATE`, so two concurrent claim flows can both observe an empty slot and the later write silently overwrites the first. `clearTaskCurrentRunProjection` has the same split read/write pattern and can turn that race into a misleading not-found error. Fold the state check into the `UPDATE` predicate and only do a follow-up read when `RowsAffected()==0` to distinguish conflict from missing task.

 


Also applies to: 119-150

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_task_projection.go` around lines 74 - 101,
The current setTaskCurrentRunProjection (and likewise
clearTaskCurrentRunProjection) is racy because it reads current_run_id then
unconditionally UPDATEs; change each to perform the state check in the UPDATE
WHERE clause (e.g. UPDATE tasks SET current_run_id = ? WHERE id = ? AND
(current_run_id = '' OR current_run_id IS NULL) or for clearing: WHERE id = ?
AND current_run_id = ?), then examine result.RowsAffected(): if >0 return
success, if 0 then perform a follow-up SELECT (reuse currentRunProjection or a
direct SELECT id,current_run_id) to distinguish between task-not-found vs
conflicting current_run_id and return taskpkg.ErrTaskNotFound or
taskpkg.ErrInvalidStatusTransition accordingly; keep error wrapping consistent
via requireRowsAffected helper or similar.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `setTaskCurrentRunProjection` and `clearTaskCurrentRunProjection` currently do a separate `SELECT current_run_id` before an unconditional `UPDATE`. That split read/write window allows a concurrent writer to change the projection between the read and write, leading to silent overwrite on set and a misleading `ErrTaskNotFound` on clear when `RowsAffected()==0`. Fix by folding the expected state into the `UPDATE` predicate and using a follow-up projection read only when the update matched no rows, so conflicts become `ErrInvalidStatusTransition` and missing tasks remain `ErrTaskNotFound`.
- Resolution: Both projection helpers now use guarded updates plus conflict rereads, and new race-simulation tests prove concurrent overwrite / clear conflicts report `ErrInvalidStatusTransition` instead of silently winning or surfacing `ErrTaskNotFound`.
