---
status: resolved
file: internal/memory/recall.go
line: 46
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745az,comment:PRRC_kwDOR5y4QM65BAQB
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep recall augmentation best-effort.**

Returning the search error here aborts prompt delivery for an auxiliary feature. Any transient catalog issue—or a query-parse failure from the search layer—will fail the whole turn instead of just skipping recall.


<details>
<summary>Suggested fix</summary>

```diff
 		results, err := target.Search(ctx, query, SearchOptions{
 			Workspace: workspaceRoot,
 			Limit:     maxRecallResults,
 		})
 		if err != nil {
-			return message, err
+			return message, nil
 		}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/recall.go` around lines 41 - 46, The recall augmentation
should be best-effort: when calling target.Search(ctx, query,
SearchOptions{Workspace: workspaceRoot, Limit: maxRecallResults}) and receiving
a non-nil err from the results, do not return the error from this function;
instead log the error (including context like query and workspaceRoot) and
continue execution with an empty/zero results set so the function returns the
original message; update the error-handling around the results, err assignment
to handle failures by logging and skipping recall rather than returning err.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The current prompt path already keeps augmentation best-effort for ordinary failures. `Manager.Prompt` logs augmenter errors and still dispatches the original message, so a recall search error does not abort prompt delivery today.
  - Swallowing all errors inside `NewRecallAugmenter` would also hide cancellation/deadline signals once cancellation handling is corrected in `manager_prompt.go`.
  - The real bug is in manager-level cancellation propagation, which is addressed under issue 011. No change is needed in `internal/memory/recall.go`.
