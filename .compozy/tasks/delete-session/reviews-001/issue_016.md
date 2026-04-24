---
status: resolved
file: web/src/systems/session/hooks/use-session-actions.ts
line: 86
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6u,comment:PRRC_kwDOR5y4QM66AonH
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In TanStack Query v5, does useMutation `onSettled` execute for both success and error results?`

💡 Result:

Yes. In TanStack Query v5, useMutation’s onSettled callback executes for both successful mutations and mutations that encounter an error. Official docs state that onSettled “will fire when the mutation is either successfully fetched or encounters an error” and it is passed either the data or the error (data may be undefined, error may be null).

Citations:

- 1: https://tanstack.com/query/v5/docs/framework/react/guides/mutations
- 2: https://tanstack.com/query/v5/docs/framework/react/reference/useMutation

---

🏁 Script executed:

```shell
fd -t f "use-session-actions.ts" web/
```

Repository: compozy/agh

Length of output: 107

---

🏁 Script executed:

```shell
# Find and examine the file
find . -name "use-session-actions.ts" -type f -exec head -100 {} +
```

Repository: compozy/agh

Length of output: 3106

---

🏁 Script executed:

```shell
# Check if sessionKeys definition exists and what keys it contains
rg "sessionKeys\s*=" --type ts -A 10 web/src/systems/session/
```

Repository: compozy/agh

Length of output: 972

---



</details>

**Move cache removal to `onSuccess` to avoid clearing session data on failed delete, and also remove the stale `events` cache.**

In TanStack Query v5, `onSettled` executes for both successful and failed mutations. The current implementation removes session caches unconditionally in `onSettled`, which means a failed delete will still wipe local session data. Additionally, `sessionKeys.events(id)` is left behind after successful deletion.

<details>
<summary>Suggested fix</summary>

```diff
 export function useDeleteSession() {
   const queryClient = useQueryClient();

   return useMutation({
     mutationFn: (id: string) => deleteSession(id),
     onSuccess: (_data, id) => {
       useSessionStore.getState().clearDraft(id);
+      queryClient.removeQueries({ queryKey: sessionKeys.detail(id) });
+      queryClient.removeQueries({ queryKey: sessionKeys.history(id) });
+      queryClient.removeQueries({ queryKey: sessionKeys.transcript(id) });
+      queryClient.removeQueries({ queryKey: sessionKeys.events(id) });
     },
-    onSettled: (_data, _error, id) => {
-      queryClient.removeQueries({ queryKey: sessionKeys.detail(id) });
-      queryClient.removeQueries({ queryKey: sessionKeys.history(id) });
-      queryClient.removeQueries({ queryKey: sessionKeys.transcript(id) });
+    onSettled: () => {
       queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
     },
   });
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/hooks/use-session-actions.ts` around lines 82 - 86,
Currently cache removal runs in the mutation's onSettled handler which clears
session data even when delete fails; move those queryClient.removeQueries calls
into the mutation's onSuccess handler instead, and add a call to remove
sessionKeys.events(id) alongside sessionKeys.detail(id),
sessionKeys.history(id), sessionKeys.transcript(id), and still call
queryClient.invalidateQueries({ queryKey: sessionKeys.lists() }) on success so
that only successful deletes purge all related caches.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `useDeleteSession` currently clears drafts and removes session caches from `onSettled`, which runs for both success and failure. That means a failed delete can still wipe local session state, and the `events` cache is not removed on success. I will move delete-side cache removal into `onSuccess`, include `sessionKeys.events(id)`, and keep failure paths from purging local state.
