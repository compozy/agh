---
status: resolved
file: internal/api/core/automation_test.go
line: 645
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM567YIh,comment:PRRC_kwDOR5y4QM63tHj6
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert `Owner.Kind` in the new task mapping check.**

Line 644 validates `Owner.Ref` but skips `Owner.Kind`, so a regression in ownership-kind mapping could pass unnoticed.

<details>
<summary>Proposed test assertion update</summary>

```diff
-	if createdJob.Scope != automationpkg.AutomationScopeWorkspace || createdJob.Name != "build review" || createdJob.AgentName != "coder" || createdJob.WorkspaceID != "ws-alpha" || createdJob.Prompt != "inspect repo" || createdJob.Schedule == nil || createdJob.Schedule.Interval != "2h" || createdJob.Task == nil || createdJob.Task.Title != "Review repo" || createdJob.Task.NetworkChannel != "ops-automation" || createdJob.Task.Owner == nil || createdJob.Task.Owner.Ref != "rule:build-review" {
+	if createdJob.Scope != automationpkg.AutomationScopeWorkspace || createdJob.Name != "build review" || createdJob.AgentName != "coder" || createdJob.WorkspaceID != "ws-alpha" || createdJob.Prompt != "inspect repo" || createdJob.Schedule == nil || createdJob.Schedule.Interval != "2h" || createdJob.Task == nil || createdJob.Task.Title != "Review repo" || createdJob.Task.NetworkChannel != "ops-automation" || createdJob.Task.Owner == nil || createdJob.Task.Owner.Kind != taskpkg.OwnerKindAutomation || createdJob.Task.Owner.Ref != "rule:build-review" {
 		t.Fatalf("jobFromCreateRequest() = %#v", createdJob)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		Task: &automationpkg.JobTaskConfig{
			Title:          " Review repo ",
			NetworkChannel: " ops-automation ",
			Owner: &taskpkg.Ownership{
				Kind: taskpkg.OwnerKindAutomation,
				Ref:  " rule:build-review ",
			},
		},
	})
	if createdJob.Scope != automationpkg.AutomationScopeWorkspace || createdJob.Name != "build review" || createdJob.AgentName != "coder" || createdJob.WorkspaceID != "ws-alpha" || createdJob.Prompt != "inspect repo" || createdJob.Schedule == nil || createdJob.Schedule.Interval != "2h" || createdJob.Task == nil || createdJob.Task.Title != "Review repo" || createdJob.Task.NetworkChannel != "ops-automation" || createdJob.Task.Owner == nil || createdJob.Task.Owner.Kind != taskpkg.OwnerKindAutomation || createdJob.Task.Owner.Ref != "rule:build-review" {
		t.Fatalf("jobFromCreateRequest() = %#v", createdJob)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/automation_test.go` around lines 635 - 645, The test
assertion after creating the job (checking createdJob from jobFromCreateRequest)
currently verifies Task.Owner.Ref but omits verifying Task.Owner.Kind; update
the assertion to also check createdJob.Task.Owner.Kind ==
taskpkg.OwnerKindAutomation so ownership-kind mapping is validated alongside
Owner.Ref and other fields (adjust the combined conditional that compares
createdJob.Scope, Name, AgentName, WorkspaceID, Prompt, Schedule, Task.Title,
Task.NetworkChannel, Task.Owner.Ref to include Task.Owner.Kind).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The `jobFromCreateRequest` assertion checks `Task.Owner.Ref` but not `Task.Owner.Kind`, so an ownership-kind mapping regression would escape. I will extend the assertion to verify `taskpkg.OwnerKindAutomation`.
  Resolution: Extended the `jobFromCreateRequest` assertion to verify both `Task.Owner.Kind` and `Task.Owner.Ref`.
