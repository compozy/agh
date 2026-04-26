---
status: resolved
file: web/src/hooks/routes/use-app-layout.ts
line: 65
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQ3,comment:PRRC_kwDOR5y4QM67VX7W
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Align `agentsLoading`/`agentsError` with the active agent source.**

When workspace-scoped agents exist, the hook still propagates global `useAgents` loading/error. That can show loading/error states even though `agents` is already resolved from workspace detail.


<details>
<summary>💡 Proposed fix</summary>

```diff
   const activeWorkspaceDetail = useWorkspace(activeWorkspaceId ?? "", {
     enabled: activeWorkspaceId !== null,
   });
-  const workspaceAgents = activeWorkspaceDetail.data?.agents ?? agents;
+  const hasWorkspaceScopedAgents =
+    activeWorkspaceId !== null && activeWorkspaceDetail.data?.agents !== undefined;
+  const workspaceAgents = hasWorkspaceScopedAgents
+    ? activeWorkspaceDetail.data?.agents
+    : agents;
@@
-    agentsLoading: agentsLoading || (activeWorkspaceId !== null && activeWorkspaceDetail.isLoading),
-    agentsError:
-      agentsError ||
-      (activeWorkspaceId !== null &&
-        activeWorkspaceDetail.isError &&
-        workspaceAgents === undefined),
+    agentsLoading: hasWorkspaceScopedAgents
+      ? activeWorkspaceDetail.isLoading
+      : agentsLoading || (activeWorkspaceId !== null && activeWorkspaceDetail.isLoading),
+    agentsError: hasWorkspaceScopedAgents
+      ? activeWorkspaceDetail.isError
+      : agentsError || (activeWorkspaceId !== null && activeWorkspaceDetail.isError),
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
   const activeWorkspaceDetail = useWorkspace(activeWorkspaceId ?? "", {
     enabled: activeWorkspaceId !== null,
   });
   const hasWorkspaceScopedAgents =
     activeWorkspaceId !== null && activeWorkspaceDetail.data?.agents !== undefined;
   const workspaceAgents = hasWorkspaceScopedAgents
     ? activeWorkspaceDetail.data?.agents
     : agents;

    agents: workspaceAgents,
    agentsLoading: hasWorkspaceScopedAgents
      ? activeWorkspaceDetail.isLoading
      : agentsLoading || (activeWorkspaceId !== null && activeWorkspaceDetail.isLoading),
    agentsError: hasWorkspaceScopedAgents
      ? activeWorkspaceDetail.isError
      : agentsError || (activeWorkspaceId !== null && activeWorkspaceDetail.isError),
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-app-layout.ts` around lines 59 - 65, The hook
currently mixes global useAgents state with workspace-scoped agents causing
false loading/error states; update the agentsLoading and agentsError expressions
to prefer workspaceAgents when present: set agentsLoading to true only if
workspaceAgents === undefined and (agentsLoading || (activeWorkspaceId !== null
&& activeWorkspaceDetail.isLoading)), and set agentsError only if
workspaceAgents === undefined and (agentsError || (activeWorkspaceId !== null &&
activeWorkspaceDetail.isError)); this ensures workspaceAgents (from
activeWorkspaceDetail) short-circuits global useAgents loading/error
propagation.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `useAppLayout()` still forwards global `useAgents()` loading/error state even when workspace-scoped agents are already available from `useWorkspace()`.
  - Root cause: `workspaceAgents` is selected from workspace detail data, but `agentsLoading`/`agentsError` still derive from a mixed global-plus-workspace expression.
  - Fix plan: short-circuit loading/error to the active source of truth and add focused coverage in `web/src/hooks/routes/use-app-layout.test.tsx` because that hook already owns the surrounding behavior tests.
