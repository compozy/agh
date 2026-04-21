---
status: resolved
file: internal/extension/host_api_test.go
line: 3517
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUL,comment:PRRC_kwDOR5y4QM65IlPQ
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Test metadata round-tripping through the Host API, not by patching the store.**

This still passes if `tasks/runs/enqueue` drops `metadata`, because the test rewrites `task_runs.metadata_json` directly right before listing. Seed the metadata via `enqueueRun(..., metadata)` and assert the list response reflects that value so the test fails on a real persistence regression.

<details>
<summary>💡 Suggested change</summary>

```diff
-	completedQueued := enqueueRun(completedTask.ID, "enqueue-complete", nil)
+	completedQueued := enqueueRun(completedTask.ID, "enqueue-complete", map[string]any{
+		"phase": "extension",
+	})
...
-	storedRun, err := env.registry.GetTaskRun(testutil.Context(t), completedQueued.ID)
-	if err != nil {
-		t.Fatalf("registry.GetTaskRun(%q) error = %v", completedQueued.ID, err)
-	}
-	storedRun.Metadata = json.RawMessage(`{"phase":"extension"}`)
-	if err := env.registry.UpdateTaskRun(testutil.Context(t), storedRun); err != nil {
-		t.Fatalf("registry.UpdateTaskRun(%q) error = %v", completedQueued.ID, err)
-	}
-
 	runsWithMetadataResult, err := env.call(t, "ext-runs", "tasks/runs", map[string]any{
```
</details>


As per coding guidelines, "MUST test meaningful business logic, not trivial operations" and "Ensure tests verify behavior outcomes, not just function calls."


Also applies to: 3669-3695

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 3507 - 3517, The test
currently rewrites task_runs.metadata_json directly before listing, so it
doesn't catch if the Host API endpoint drops metadata; change the test to seed
the metadata through the enqueueRun helper by passing a non-nil metadata map
into enqueueRun(taskID, idempotencyKey, metadata) (which sends it via env.call
to "ext-runs" "tasks/runs/enqueue"), then call the listing endpoint and assert
the returned task run's metadata equals the seeded map; remove or stop relying
on any direct writes to task_runs.metadata_json so the assertion fails if
tasks/runs/enqueue does not persist metadata.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The current test enqueues the run without metadata, then mutates `task_runs.metadata_json` directly through the registry before listing.
  - That means the assertion does not prove that the Host API `tasks/runs/enqueue` route actually persists metadata end to end.
  - Root cause: the regression test bypasses the behavior it is supposed to validate.
  - Fix approach: seed metadata through `enqueueRun(..., metadata)` and assert the subsequent `tasks/runs` response preserves that value without any direct store patching.
  - During implementation, the stronger test exposed a real production bug: `apicontract.EnqueueTaskRunRequest` and `enqueueTaskRunFromRequest` both dropped `metadata`.
  - Minimal out-of-scope fixes were required in `internal/api/contract/tasks.go` and `internal/extension/host_api_tasks.go` so the metadata provided by `tasks/runs/enqueue` now reaches `taskpkg.EnqueueRun` and round-trips back through the Host API.
