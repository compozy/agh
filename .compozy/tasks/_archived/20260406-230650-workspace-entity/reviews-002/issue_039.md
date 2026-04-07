---
status: resolved
file: web/src/systems/workspace/hooks/use-workspaces.ts
line: 18
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDX,comment:PRRC_kwDOR5y4QM61T6JI
---

# Issue 039: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify mutation lifecycle callbacks used in hooks under systems.
rg -n --type=ts -C3 'useMutation\(' web/src/systems/*/hooks
rg -n --type=ts -C2 'onMutate|onError|onSuccess|onSettled' web/src/systems/*/hooks
```

Repository: compozy/agh

Length of output: 5247

---



</details>

**Use `onSettled` for cache invalidation and implement optimistic updates with rollback.**

Line 16 uses `onSuccess` to invalidate, which misses error paths. The guidelines require `onSettled` for unconditional invalidation and `onMutate`/`onError` snapshots for optimistic updates with rollback on failure.

<details>
<summary>♻️ Proposed refactor</summary>

```diff
 export function useResolveWorkspace() {
   const queryClient = useQueryClient();

   return useMutation({
     mutationFn: (params: ResolveWorkspaceParams) => resolveWorkspace(params),
-    onSuccess: () => {
-      queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() });
-    },
+    onMutate: async () => {
+      await queryClient.cancelQueries({ queryKey: workspaceKeys.lists() });
+      const previous = queryClient.getQueryData(workspaceKeys.lists());
+      return { previous };
+    },
+    onError: (_error, _variables, context) => {
+      if (context?.previous !== undefined) {
+        queryClient.setQueryData(workspaceKeys.lists(), context.previous);
+      }
+    },
+    onSettled: async () => {
+      await queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() });
+    },
   });
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/workspace/hooks/use-workspaces.ts` around lines 14 - 18,
Replace the current onSuccess-only invalidation with an onSettled handler and
add optimistic-update handlers: in the useMutation call that wraps
resolveWorkspace, implement onMutate to take a snapshot of the current list via
queryClient.getQueryData(workspaceKeys.lists()), apply optimistic changes with
queryClient.setQueryData, and return the snapshot as context; implement onError
to rollback by restoring the snapshot returned from onMutate (using
queryClient.setQueryData) when the mutation fails; finally use onSettled (not
onSuccess) to unconditionally call queryClient.invalidateQueries({ queryKey:
workspaceKeys.lists() }) so cache is refreshed in all cases.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  `useResolveWorkspace` is not consumed by any production component yet, so
  there is no current optimistic-UI gap to repair. The mutation only changes
  server state on success, making success-only invalidation sufficient, and
  adding speculative optimistic list entries would guess server behavior without
  a real caller. No change.
