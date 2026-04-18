---
status: resolved
file: internal/api/core/tasks.go
line: 1152
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUY,comment:PRRC_kwDOR5y4QM65ChGn
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Post-fetch draft filtering may return fewer items than the requested limit.**

When `IncludeDrafts=false` and no explicit status filter is set, drafts are removed after the domain query already applied its limit. If the database returns `limit` items and some are drafts, the final result will be smaller than requested.

Consider either:
1. Passing the draft-exclusion filter down to the domain query so the DB can compensate, or
2. Over-fetching and then truncating to the requested limit.



<details>
<summary>🔧 Example scenario</summary>

```
Request: limit=10, include_drafts=false
DB returns: 10 tasks (6 drafts + 4 non-drafts)
After filter: 4 tasks returned to client
Expected: up to 10 non-draft tasks
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks.go` around lines 1136 - 1152, filterTaskListDrafts
removes drafts after the DB applied its limit, which can yield fewer items than
requested; fix by pushing draft exclusion into the domain query or by
over-fetching until you have query.Limit non-draft items. Specifically, when
query.IncludeDrafts is false and query.Status.Normalize() == "" (i.e., no
explicit status filter), update the code that builds/executes the DB/domain Task
list query to add a draft=false predicate so the DB returns only non-drafts, or
implement an over-fetch loop in the listing function that requests more rows
(e.g., increase page size or fetch additional pages) and re-runs
filterTaskListDrafts on the combined results until either you have query.Limit
non-draft taskpkg.Summary items or the DB has no more rows; adjust callers that
rely on filterTaskListDrafts accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: In `internal/api/core/tasks.go`, `ListTasks` applies `manager.ListTasks(..., query)` with the requested limit and only removes drafts afterward via `filterTaskListDrafts`. When drafts are excluded by default and the limited result window contains drafts, the client can receive fewer non-draft rows than requested.
- Root cause analysis: Draft exclusion happens after the domain/store limit is already applied, and the core HTTP path has no compensating overfetch strategy.
- Intended fix: Introduce a bounded draft-aware list helper in the core task surface that increases the effective fetch limit safely, re-filters drafts, and truncates back to the requested limit without performing an unbounded full fetch.
- Resolution: Added a bounded draft-compensation helper to the core task list handler so default draft exclusion can refill under-filled windows without unbounded fetches.
- Verification:
  - `go test ./internal/api/contract ./internal/api/core ./internal/daemon`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
