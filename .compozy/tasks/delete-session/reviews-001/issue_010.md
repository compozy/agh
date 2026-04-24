---
status: resolved
file: internal/store/globaldb/global_db_task.go
line: 120
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6i,comment:PRRC_kwDOR5y4QM66Aom5
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Map parent-child FK delete failures to a domain error.**

`DeleteTask` only maps “0 rows affected” to `ErrTaskNotFound`. If the task has direct children (`tasks.parent_task_id` FK without cascade), this path returns a raw DB constraint error instead of a stable task-domain error, which can surface as an internal server error upstream.

Please normalize this failure (e.g., pre-check child count or translate FK constraint failure to a dedicated task error).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_task.go` around lines 104 - 120, DeleteTask
currently only maps 0-rows to taskpkg.ErrTaskNotFound and lets DB FK constraint
errors bubble up; update DeleteTask to normalize parent-child FK failures by
either (a) pre-checking for children with a SELECT COUNT(*) FROM tasks WHERE
parent_task_id = ? using the same trimmedID and return a domain error (e.g.,
taskpkg.ErrTaskHasChildren) if count>0, or (b) catch the DB constraint error
from g.db.ExecContext and translate it to taskpkg.ErrTaskHasChildren before
returning; modify the logic around ExecContext/requireRowsAffected (referencing
DeleteTask, requireTaskValue, requireRowsAffected, and taskpkg.ErrTaskNotFound)
so callers receive a stable task-domain error instead of a raw DB constraint
error.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `GlobalDB.DeleteTask` only normalizes "0 rows affected" and otherwise returns the raw SQLite error, so a foreign-key delete failure can leak as a transport `500` if it occurs at the storage boundary. The manager already treats child-task deletes as validation failures, so the store should translate the SQLite foreign-key failure to the same stable task-domain validation error. I will add that mapping in `global_db_task.go` and add a minimal storage test in `internal/store/globaldb/global_db_task_test.go` because the scoped files do not currently contain direct coverage for this constraint path.
