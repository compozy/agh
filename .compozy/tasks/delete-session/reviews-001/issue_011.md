---
status: resolved
file: internal/task/manager.go
line: 300
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6o,comment:PRRC_kwDOR5y4QM66AonA
---

# Issue 011: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap delete-path failures with operation context.**

Several returns (for example Line 281, Line 289, Line 294, Line 299, Line 2201, and Line 2214) bubble raw errors. Add contextual wrapping so delete failures are diagnosable in logs and callers.

<details>
<summary>♻️ Suggested patch</summary>

```diff
 record, err := m.store.GetTask(ctx, trimmedID)
 if err != nil {
-	return err
+	return fmt.Errorf("task: load task %q for delete: %w", trimmedID, err)
 }
 if err := m.ensureTaskDeleteAllowed(ctx, record); err != nil {
 	return err
 }

 dependents, err := m.store.ListDependents(ctx, trimmedID)
 if err != nil {
-	return err
+	return fmt.Errorf("task: list dependents for task %q delete: %w", trimmedID, err)
 }

 if err := m.store.DeleteTask(ctx, trimmedID); err != nil {
-	return err
+	return fmt.Errorf("task: delete task %q: %w", trimmedID, err)
 }

 for _, dependentID := range dependentIDs {
 	if _, err := m.reconcileTaskCascade(ctx, dependentID); err != nil {
-		return err
+		return fmt.Errorf("task: reconcile dependent task %q after deleting %q: %w", dependentID, trimmedID, err)
 	}
 }
```
</details>



As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".


Also applies to: 2199-2214

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 279 - 300, The delete path returns raw
errors from calls like m.store.GetTask, m.ensureTaskDeleteAllowed,
m.store.ListDependents, m.store.DeleteTask and m.reconcileTaskCascade; update
each error return to wrap the underlying error with contextual text using
fmt.Errorf("...: %w", err) (e.g., include the operation and trimmedID or
dependentID), so replace bare returns like `return err` with wrapped errors that
mention the failing operation
(GetTask/DeleteTask/ListDependents/reconcileTaskCascade) and the affected task
ID; apply the same wrapping for the other occurrences around lines 2199–2214.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The task manager delete path currently returns raw errors from `GetTask`, `ListDependents`, `DeleteTask`, `reconcileTaskCascade`, `CountDirectChildren`, and `ListTaskRuns`, which makes delete failures harder to diagnose. I will wrap those errors with delete-specific context and add delete-path tests in `internal/task/manager_test.go` to verify the wrapped errors still preserve their sentinels.
