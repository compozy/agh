---
status: resolved
file: internal/store/globaldb/global_db_task_aux.go
line: 735
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQt,comment:PRRC_kwDOR5y4QM67VX7M
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid scanning full run history while holding the SQLite write lock.**

This path now loads all runs for the task inside `BEGIN IMMEDIATE` just to answer “is there any open run?”. On tasks with long run history, every enqueue becomes a full read/allocation before insert, which lengthens the writer lock and can stall unrelated writes. A targeted `EXISTS`/`LIMIT 1` query on non-terminal statuses keeps the guard cheap and preserves the run ID for the error message.  


<details>
<summary>♻️ Suggested shape</summary>

```diff
-	existingRuns, err := g.listTaskRunsWithExecutor(ctx, exec, taskpkg.RunQuery{TaskID: taskRecord.ID})
-	if err != nil {
-		return taskpkg.Task{}, taskpkg.Run{}, false, err
-	}
-	if err := validateNoOpenRunForQueuedRunReservation(taskRecord, existingRuns); err != nil {
+	openRunID, err := g.findOpenRunIDForQueuedRunReservation(ctx, exec, taskRecord.ID)
+	if err != nil {
+		return taskpkg.Task{}, taskpkg.Run{}, false, err
+	}
+	if openRunID != "" {
+		return taskpkg.Task{}, taskpkg.Run{}, false, fmt.Errorf(
+			"%w: task %q has open run %q; finish or cancel it before enqueueing another run",
+			taskpkg.ErrInvalidStatusTransition,
+			taskRecord.ID,
+			openRunID,
+		)
+	}
-		return taskpkg.Task{}, taskpkg.Run{}, false, err
-	}
```

```go
func (g *GlobalDB) findOpenRunIDForQueuedRunReservation(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (string, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT id
		   FROM task_runs
		  WHERE task_id = ?
		    AND status NOT IN (?, ?, ?)
		  ORDER BY queued_at DESC, id DESC
		  LIMIT 1`,
		taskID,
		string(taskpkg.TaskRunStatusCompleted),
		string(taskpkg.TaskRunStatusFailed),
		string(taskpkg.TaskRunStatusCanceled),
	)

	var runID string
	if err := row.Scan(&runID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("store: lookup open task run for %q: %w", taskID, err)
	}
	return runID, nil
}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_task_aux.go` around lines 729 - 735, The
current code calls listTaskRunsWithExecutor and
validateNoOpenRunForQueuedRunReservation while holding the SQLite write lock,
which loads the full run history; instead add a new helper
findOpenRunIDForQueuedRunReservation(ctx, exec, taskID) that runs a targeted
SELECT id ... WHERE task_id = ? AND status NOT IN (...) ORDER BY queued_at DESC,
id DESC LIMIT 1 (using exec.QueryRowContext) and returns the run ID or empty
string, and then replace the existing call to
listTaskRunsWithExecutor/validateNoOpenRunForQueuedRunReservation with a call to
findOpenRunIDForQueuedRunReservation; if it returns a non-empty run ID, pass
that ID into a lightweight validation branch (or adapt
validateNoOpenRunForQueuedRunReservation to accept a run ID instead of a full
slice) so you preserve the run ID for the error message while avoiding scanning
all runs inside BEGIN IMMEDIATE.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `reserveQueuedRunWithExecutor()` currently calls `listTaskRunsWithExecutor()` inside the `BEGIN IMMEDIATE` path just to decide whether any non-terminal run exists.
  - Root cause: open-run validation is implemented as a full run-history scan instead of a targeted existence lookup, which unnecessarily lengthens the SQLite writer lock.
  - Fix plan: replace the full scan with a `LIMIT 1` helper that returns the newest non-terminal run id, preserve the current error message, and reuse the expanded open-run regression coverage in `global_db_task_test.go`.
