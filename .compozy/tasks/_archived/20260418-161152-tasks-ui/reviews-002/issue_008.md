---
status: resolved
file: internal/observe/tasks.go
line: 88
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUk,comment:PRRC_kwDOR5y4QM65ChGz
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`TaskDashboardQuery.Validate` misses scope/workspace binding validation.**

`TaskDashboardQuery` can currently pass validation even with an invalid `Scope`/`WorkspaceID` combination because it only delegates to `summaryQuery().Validate()`.

<details>
<summary>🔧 Proposed fix</summary>

```diff
 func (q TaskDashboardQuery) Validate() error {
-	return q.summaryQuery().Validate()
+	if err := taskpkg.ValidateScopeBinding(q.Scope, q.WorkspaceID, "task_dashboard_query", "workspace_id"); err != nil {
+		return err
+	}
+	return q.summaryQuery().Validate()
 }
```
</details>




Also applies to: 90-99

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/tasks.go` around lines 86 - 88, TaskDashboardQuery.Validate
currently only calls summaryQuery().Validate() and therefore skips verifying the
Scope/WorkspaceID relationship; update TaskDashboardQuery.Validate to first
validate that the Scope and WorkspaceID are present and consistent (e.g.,
non-empty and that the Scope type allows the provided WorkspaceID or that the
WorkspaceID belongs to the Scope) before delegating to
summaryQuery().Validate(), and apply the same explicit Scope/Workspace binding
checks to the similar validators referenced around lines 90-99 so all related
query types enforce the scope-workspace constraint.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `internal/observe/tasks.go` lets `TaskDashboardQuery.Validate()` delegate to `summaryQuery().Validate()`, but `TaskSummaryQuery.Validate()` currently checks enum validity only and does not enforce the `Scope`/`WorkspaceID` binding relationship.
- Root cause analysis: Scope-binding validation exists in `TaskInboxQuery.Validate()` but was not centralized in `TaskSummaryQuery.Validate()`, so dashboard queries can bypass it.
- Intended fix: Add scope/workspace binding validation to `TaskSummaryQuery.Validate()` so dashboard and other summary-backed paths inherit the check consistently.
- Resolution: Added the scope/workspace binding check directly to `TaskDashboardQuery.Validate()`, which closes the dashboard gap without breaking pre-resolution parser validation for workspace references.
- Verification:
  - `go test ./internal/extension ./internal/observe`
  - `go test -tags integration ./internal/observe -run 'TestObserveTaskDashboard|TestObserveHealthReflectsRecoveryAndForcedStopOutcomes|TestObserveTaskLifecycleSummaryAndMetrics'`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
