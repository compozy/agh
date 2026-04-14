---
status: resolved
file: internal/automation/dispatch.go
line: 456
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562an8,comment:PRRC_kwDOR5y4QM63mgRN
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Task-backed dispatch drops the pre-fire hook’s prompt rewrite.**

`dispatchPreFireHook` returns the patched prompt, but this path discards it and `directTaskSpec` still falls back to `job.Prompt`. Any hook-based prompt rewrite/sanitization never reaches the delegated task payload.

<details>
<summary>💡 Proposed fix</summary>

```diff
-	_, cancelled, hookErr := d.dispatchPreFireHook(ctx, req, preFirePrompt, attempt)
+	prompt, cancelled, hookErr := d.dispatchPreFireHook(ctx, req, preFirePrompt, attempt)
 	if hookErr != nil {
 		return d.finishRun(ctx, scheduledRun, RunFailed, hookErr)
 	}
 	if cancelled {
 		return d.finishRun(ctx, scheduledRun, RunCancelled, nil)
 	}
@@
-	taskRecord, err := d.tasks.CreateTask(ctx, directTaskSpec(req.Job), actor)
+	taskRecord, err := d.tasks.CreateTask(ctx, directTaskSpec(req.Job, prompt), actor)
```

```diff
-func directTaskSpec(job *Job) taskpkg.CreateTask {
+func directTaskSpec(job *Job, prompt string) taskpkg.CreateTask {
 	if job == nil || job.Task == nil {
 		return taskpkg.CreateTask{}
 	}
@@
 	description := strings.TrimSpace(job.Task.Description)
 	if description == "" {
-		description = strings.TrimSpace(job.Prompt)
+		description = strings.TrimSpace(prompt)
+	}
+	if description == "" {
+		description = strings.TrimSpace(job.Prompt)
 	}
```
</details>


Also applies to: 469-472, 920-923

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 452 - 456, The pre-fire hook's
rewritten prompt returned by dispatchPreFireHook is being ignored; update the
call sites (where preFirePrompt is set and where directTaskSpec is constructed)
to assign the returned patched prompt (the first return value) back into
preFirePrompt and use that variable when building directTaskSpec instead of
falling back to req.Job.Prompt or req.Prompt; specifically, capture the
patchedPrompt from dispatchPreFireHook and pass that into the delegated task
payload so hook-based rewrites/sanitization are preserved (adjust the
occurrences around dispatchPreFireHook and where directTaskSpec is populated).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: task-backed dispatch ignores the prompt returned by `dispatchPreFireHook` and still builds the durable task from the original job prompt.
- Fix approach: propagate the rewritten prompt through the task-backed path and use it when constructing the delegated task payload.

## Resolution

- Propagated the rewritten pre-fire prompt through task-backed dispatch and added a regression test that asserts the durable delegated task description uses the hook-rewritten prompt.
- Verified in the final `make verify` run.
