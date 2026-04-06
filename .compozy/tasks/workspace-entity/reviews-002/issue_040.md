---
status: resolved
file: web/src/systems/workspace/lib/query-keys.ts
line: 4
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDa,comment:PRRC_kwDOR5y4QM61T6JN
---

# Issue 040: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**`lists()` and `list()` currently generate identical keys**

At Line 3 and Line 4, both functions resolve to the same tuple, so there’s no real hierarchy for finer invalidation targeting.



<details>
<summary>Proposed key hierarchy fix</summary>

```diff
 export const workspaceKeys = {
   all: ["workspaces"] as const,
   lists: () => [...workspaceKeys.all, "list"] as const,
-  list: () => [...workspaceKeys.lists()] as const,
+  list: (scope: "all" | "active" = "all") => [...workspaceKeys.lists(), scope] as const,
 };
```
</details>

As per coding guidelines, "Define hierarchical query keys in `src/systems/<domain>/lib/query-keys.ts` for granular cache invalidation".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
export const workspaceKeys = {
  all: ["workspaces"] as const,
  lists: () => [...workspaceKeys.all, "list"] as const,
  list: (scope: "all" | "active" = "all") => [...workspaceKeys.lists(), scope] as const,
};
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/workspace/lib/query-keys.ts` around lines 3 - 4, The two keys
are identical; change workspaceKeys.lists to produce a "lists" level (e.g.
[...workspaceKeys.all, "lists"]) and make workspaceKeys.list accept a list id
(e.g. list: (id) => [...workspaceKeys.lists(), id]) so that list() yields a more
specific key for a single list; keep the "as const" typing and update any
callers to pass the id to workspaceKeys.list.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  The workspace registry currently exposes only one list shape, so
  `workspaceKeys.lists()` and `workspaceKeys.list()` sharing the same tuple does
  not block any real invalidation use case. Introducing extra hierarchy would be
  speculative API churn without a caller that needs finer granularity. No
  change.
