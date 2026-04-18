---
status: resolved
file: internal/extension/host_api_tasks.go
line: 34
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUf,comment:PRRC_kwDOR5y4QM65ChGu
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid unbounded overfetch in `handleTasks`.**

Setting `query.Limit = 0` can turn a paged request into a full dataset fetch, which is risky under high cardinality and can regress latency/memory.

Use a bounded overfetch strategy instead (or add a server-side “exclude drafts” filter in the manager query path).

<details>
<summary>🔧 Bounded overfetch example</summary>

```diff
 	if shouldOverfetchTaskDrafts(params) {
-		query.Limit = 0
+		// Keep overfetch bounded to avoid full scans.
+		query.Limit = min(params.Limit*4, 500)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_tasks.go` around lines 32 - 34, handleTasks
currently sets query.Limit = 0 when shouldOverfetchTaskDrafts(params) is true,
which causes unbounded fetches; change this to a bounded overfetch strategy by
replacing the zero with a capped value (e.g., compute newLimit = min(query.Limit
+ overfetchAmount, maxOverfetchLimit) and assign query.Limit = newLimit) or,
alternatively, add a server-side filter in the manager query path to exclude
drafts instead of expanding the page size. Update references to
shouldOverfetchTaskDrafts and query.Limit in handleTasks and introduce clear
constants like overfetchAmount/maxOverfetchLimit or a server-side "exclude
drafts" flag so the behavior is safe under high cardinality.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `internal/extension/host_api_tasks.go` currently sets `query.Limit = 0` when compensating for post-fetch draft filtering. That turns a bounded request into an unbounded fetch.
- Root cause analysis: The host API path tried to address the core draft-filtering shortfall by removing the limit entirely, which fixes under-filled pages at the cost of unbounded reads.
- Intended fix: Replace the unbounded fetch with the same bounded draft-aware overfetch strategy used in the core HTTP surface so the host API remains safe under high cardinality.
- Resolution: Replaced the `query.Limit = 0` escape hatch with bounded draft-aware compensation in the host API task list path.
- Verification:
  - `go test ./internal/extension ./internal/observe`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
