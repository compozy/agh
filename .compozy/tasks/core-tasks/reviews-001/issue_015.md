---
status: resolved
file: internal/automation/model/validate.go
line: 340
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aoF,comment:PRRC_kwDOR5y4QM63mgRW
---

# Issue 015: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`Run.Validate` still accepts one invalid delegated state.**

This only checks the `task_id set / task_run_id missing` half of the invariant. `Run{Status: RunDelegated, TaskRunID: "tr-1"}` still passes model validation with an empty `TaskID`, even though the store later rejects it. Keep the model-layer rule symmetric so invalid delegated runs fail early.



<details>
<summary>✅ Suggested validation alignment</summary>

```diff
-	if strings.TrimSpace(r.TaskID) != "" && strings.TrimSpace(r.TaskRunID) == "" && r.Status == RunDelegated {
-		return fmt.Errorf("%s is required when %s is %q and %s is set", nestedPath(path, "task_run_id"), nestedPath(path, "status"), RunDelegated, nestedPath(path, "task_id"))
-	}
+	if r.Status == RunDelegated {
+		if strings.TrimSpace(r.TaskID) == "" {
+			return fmt.Errorf("%s is required when %s is %q", nestedPath(path, "task_id"), nestedPath(path, "status"), RunDelegated)
+		}
+		if strings.TrimSpace(r.TaskRunID) == "" {
+			return fmt.Errorf("%s is required when %s is %q", nestedPath(path, "task_run_id"), nestedPath(path, "status"), RunDelegated)
+		}
+	}
 	return nil
 }
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if r.Status == RunDelegated {
		if strings.TrimSpace(r.TaskID) == "" {
			return fmt.Errorf("%s is required when %s is %q", nestedPath(path, "task_id"), nestedPath(path, "status"), RunDelegated)
		}
		if strings.TrimSpace(r.TaskRunID) == "" {
			return fmt.Errorf("%s is required when %s is %q", nestedPath(path, "task_run_id"), nestedPath(path, "status"), RunDelegated)
		}
	}
	return nil
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/model/validate.go` around lines 337 - 340, The current
Run.Validate only enforces that when TaskID is set and Status == RunDelegated,
TaskRunID must be present; add the symmetric check so that when TaskRunID is set
and Status == RunDelegated, TaskID must not be empty. In the Run.Validate
function (referencing r.TaskID, r.TaskRunID, r.Status and the RunDelegated
constant), add a second conditional mirroring the existing one that returns
fmt.Errorf(...) using nestedPath(path, "task_id") and nestedPath(path,
"task_run_id") to produce the appropriate error message when TaskRunID is
provided but TaskID is missing.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Run.Validate` only enforces `TaskRunID` when `TaskID` is present for delegated runs, but it does not enforce the symmetric `TaskID` requirement when `TaskRunID` is set.
- Fix approach: make delegated-run validation require both fields and add regression coverage for the missing `TaskID` case.

## Resolution

- Made delegated automation runs require both `task_id` and `task_run_id`, and added regression coverage for the missing-`task_id` branch.
- Verified in the final `make verify` run.
