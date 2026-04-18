---
status: resolved
file: internal/extension/host_api_tasks.go
line: 1422
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb2,comment:PRRC_kwDOR5y4QM65B8fU
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Filtering drafts after `ListTasks` breaks `limit` semantics.**

`manager.ListTasks` has already applied `query.Limit`, so removing drafts afterward can return fewer items even when more non-draft tasks exist beyond that first page. The draft predicate needs to be part of the task query itself, or this path needs to over-fetch until it fills the visible page.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_tasks.go` around lines 1406 - 1422, filtering out
drafts in filterTaskListDrafts after manager.ListTasks has already honored
query.Limit breaks pagination by possibly returning fewer non-draft items; fix
by moving the draft predicate into the original task query (so manager.ListTasks
receives an explicit exclude-drafts flag) or by changing this path to over-fetch
until you accumulate query.Limit non-draft items: either 1) add/propagate a flag
on apicontract.TaskListQuery (or set the existing IncludeDrafts appropriately)
so manager.ListTasks returns only non-drafts, or 2) implement an over-fetch loop
in filterTaskListDrafts that requests additional pages from ListTasks until
len(filtered)==query.Limit or no more tasks remain (use filterTaskListDrafts,
apicontract.TaskListQuery, taskpkg.Summary and the query.Limit semantics to
locate where to change).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed. `handleTasks` asks `manager.ListTasks` to apply the caller’s limit first, then `filterTaskListDrafts` removes drafts afterward. When drafts are hidden, the returned page can contain fewer non-draft items than requested even if more visible tasks exist later in the full result set. I’ll over-fetch for the hidden-draft case and only apply the visible limit after filtering.
