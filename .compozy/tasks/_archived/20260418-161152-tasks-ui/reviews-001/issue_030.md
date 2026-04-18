---
status: resolved
file: internal/task/manager.go
line: 1188
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb-,comment:PRRC_kwDOR5y4QM65B8fd
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**This summary path is N+1 on tasks and then N+1 again on dependencies.**

`listTaskSummaries` enriches every task with separate child-count, dependency, run, and event reads, then `buildDependencyReferences` resolves each dependency through another task/runs/dependencies chain. The new dashboard/inbox surfaces call this over large task sets, so latency will grow with total tasks and dependency edges instead of the page size. This should be batched in the store/read model before the new UI lands.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 1105 - 1188,
listTaskSummaries/enrichTaskSummary/buildDependencyReferences cause N+1 queries:
each summary calls CountDirectChildren, ListDependencies, ListTaskRuns,
ListTaskEvents and taskReference for each dependency; instead add batched store
APIs (e.g. ListTasksWithMeta or separate methods like
CountDirectChildrenForTasks(ctx, []taskID), ListDependenciesForTasks(ctx,
[]taskID), ListTaskRunsForTasks(ctx, []taskID), ListTaskEventsForTasks(ctx,
[]taskID), and ResolveTaskReferences(ctx, []taskID)) and use them in
listTaskSummaries to fetch all counts/dependencies/runs/events/references in
bulk, then change
enrichTaskSummary/enrichTaskSummaryFromState/buildDependencyReferences to accept
pre-fetched maps/slices and build summaries from those maps (lookup by task ID)
rather than issuing per-item store calls, preserving existing return types and
status logic.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: the review comment’s key premise is incorrect for this batch. The new dashboard and inbox surfaces are implemented in `internal/observe` and do not call `task.Service.listTaskSummaries`.
- Reasoning: `listTaskSummaries` does have an N+1-shaped enrichment pattern, but that is an existing optimization opportunity rather than a correctness regression introduced by these scoped review changes. A real fix would require broader batched store APIs and interface/test updates well beyond the targeted review remediation here.

## Resolution

- Closed as `invalid`.
- No code change was made because the reported N+1 path is not exercised by the new observer surfaces in this batch and would require broader out-of-scope read-model work.
