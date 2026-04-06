---
status: resolved
file: web/src/systems/session/hooks/use-session-actions.ts
line: 21
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDF,comment:PRRC_kwDOR5y4QM61T6Is
---

# Issue 034: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Move create-session cache invalidation to `onSettled`**

At Line 19, invalidation is still in `onSuccess`. This breaks the hook guideline for mutation cache handling consistency.



<details>
<summary>Proposed callback structure</summary>

```diff
 return useMutation({
   mutationFn: (params: CreateSessionParams) => createSession(params),
   onSuccess: session => {
-    queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
     navigate({ to: "/session/$id", params: { id: session.id } });
   },
+  onSettled: () => {
+    queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
+  },
 });
```
</details>

As per coding guidelines, "Always invalidate TanStack Query cache after mutations via `onSettled` callback".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  return useMutation({
    mutationFn: (params: CreateSessionParams) => createSession(params),
    onSuccess: session => {
      navigate({ to: "/session/$id", params: { id: session.id } });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/hooks/use-session-actions.ts` around lines 16 - 21,
The cache invalidation currently in the useMutation onSuccess block should be
moved to an onSettled handler: remove queryClient.invalidateQueries from
onSuccess (keep navigate({ to: "/session/$id", params: { id: session.id } }) in
onSuccess to preserve redirect), add an onSettled callback on the same
useMutation (signature like onSettled: (data, error, variables, context) => {
... }) and call queryClient.invalidateQueries({ queryKey: sessionKeys.lists() })
there so TanStack Query cache is always invalidated after createSession
regardless of outcome.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  In the current call flow, only a successful `createSession` mutation changes
  the sessions list. Failed creates do not require rollback or forced cache
  reconciliation, and the list query already polls on an interval. Moving this
  invalidation to `onSettled` would be a consistency-only refactor without a
  demonstrated correctness bug in this batch. No change.
