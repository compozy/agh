---
status: resolved
file: internal/observe/tasks.go
line: 1616
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUm,comment:PRRC_kwDOR5y4QM65ChG2
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`loadTaskSnapshot` introduces an N+1 dependency query pattern.**

Line 1606-1616 performs one `CountDependencies` call per task. For large task sets, dashboard/inbox/summary reads can degrade sharply and put avoidable pressure on the store.

Consider a batched dependency-count API (or embedding counts in `ListTasks`) to keep snapshot loading O(1) store round-trips.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/tasks.go` around lines 1606 - 1616, loadTaskSnapshot
currently issues an N+1 query by calling o.registry.CountDependencies for each
task in the tasks slice; replace this with a single batched dependency-count
lookup (either add a new registry method like CountDependenciesForTasks(ctx,
taskIDs []string) that returns a map[id]count or change ListTasks to populate
DependencyCount) and then set tasks[idx].DependencyCount from that map in one
pass; update the call site in loadTaskSnapshot to collect task IDs, call the new
batched API once, and apply results to the tasks slice instead of calling
o.registry.CountDependencies per task.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `loadTaskSnapshot` currently iterates over every task and calls `o.registry.CountDependencies(ctx, taskID)` one task at a time. That is an N+1 read pattern on observer summary/dashboard/inbox paths.
- Root cause analysis: Dependency counts are loaded after `ListTasks` with per-task store calls instead of a single batched lookup.
- Intended fix: Replace the per-task count loop with a batched dependency-count load inside the scoped observer files and add regression coverage that proves snapshot reads no longer rely on repeated `CountDependencies` calls.
- Resolution: Replaced the per-task dependency-count loop with a batched SQLite lookup inside the scoped observer files and added a regression test that fails if `CountDependencies` is invoked per task.
- Verification:
  - `go test ./internal/extension ./internal/observe`
  - `go test -tags integration ./internal/observe -run 'TestObserveTaskDashboard|TestObserveHealthReflectsRecoveryAndForcedStopOutcomes|TestObserveTaskLifecycleSummaryAndMetrics'`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
