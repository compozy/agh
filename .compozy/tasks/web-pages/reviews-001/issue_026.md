---
status: pending
file: web/src/systems/network/hooks/use-network-actions.ts
line: 22
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4e,comment:PRRC_kwDOR5y4QM63ZMIV
---

# Issue 026: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
python - <<'PY'
from pathlib import Path
p = Path("web/src/systems/network/hooks/use-network-actions.ts")
txt = p.read_text()
for key in ("onMutate", "onError", "onSettled"):
    print(f"{key}: {'present' if key in txt else 'missing'}")
PY
```

Repository: compozy/agh

Length of output: 108

---

🏁 Script executed:

```shell
cat -n web/src/systems/network/hooks/use-network-actions.ts | head -40
```

Repository: compozy/agh

Length of output: 1021

---



</details>

**Add optimistic update handlers to this mutation.**

The hook invalidates on settle but is missing the required optimistic update pattern (`onMutate` + `onError`) as specified in the coding guidelines. Implement query cancellation and rollback snapshots:

<details>
<summary>Suggested implementation</summary>

```diff
 export function useCreateNetworkChannel() {
   const queryClient = useQueryClient();

   return useMutation({
     mutationFn: (data: CreateNetworkChannelRequest) => createNetworkChannel(data),
+    onMutate: async () => {
+      await queryClient.cancelQueries({ queryKey: networkKeys.all });
+      const previousQueries = queryClient.getQueriesData({ queryKey: networkKeys.all });
+      return { previousQueries };
+    },
+    onError: (_error, _variables, context) => {
+      if (!context) return;
+      for (const [queryKey, data] of context.previousQueries) {
+        queryClient.setQueryData(queryKey, data);
+      }
+    },
     onSettled: () => invalidateNetworkQueries(queryClient),
   });
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  return useMutation({
    mutationFn: (data: CreateNetworkChannelRequest) => createNetworkChannel(data),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: networkKeys.all });
      const previousQueries = queryClient.getQueriesData({ queryKey: networkKeys.all });
      return { previousQueries };
    },
    onError: (_error, _variables, context) => {
      if (!context) return;
      for (const [queryKey, data] of context.previousQueries) {
        queryClient.setQueryData(queryKey, data);
      }
    },
    onSettled: () => invalidateNetworkQueries(queryClient),
  });
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/hooks/use-network-actions.ts` around lines 19 - 22,
The mutation in useNetworkActions (mutationFn: createNetworkChannel) needs
optimistic update handlers: implement onMutate to cancel relevant queries via
queryClient.cancelQueries, take a snapshot with queryClient.getQueryData (store
as context), apply an optimistic update using queryClient.setQueryData to insert
the new channel into the network list, and return the snapshot in context;
implement onError to rollback by restoring the snapshot from context with
queryClient.setQueryData; keep the existing onSettled to call
invalidateNetworkQueries(queryClient) after either success or failure. Ensure
you reference the existing symbols: useNetworkActions, createNetworkChannel,
queryClient, invalidateNetworkQueries, and add onMutate/onError handlers
accordingly.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
