---
status: resolved
file: web/src/hooks/routes/use-network-page.ts
line: 498
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQ6,comment:PRRC_kwDOR5y4QM67VX7Z
---

# Issue 021: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Sum `presence_count` here instead of counting grouped greet rows.**

The API now returns collapsed presence episodes, so one `"greet"` item can represent multiple heartbeats. `filter(...).length` reports episode count, not actual presence count, and underreports peers with repeated greets.

<details>
<summary>Suggested fix</summary>

```diff
-    presenceCount: showPresence
-      ? rawMessages.filter(message => message.kind === "greet").length
-      : 0,
+    presenceCount: showPresence
+      ? rawMessages.reduce(
+          (total, message) =>
+            message.kind === "greet" ? total + (message.presence_count ?? 1) : total,
+          0
+        )
+      : 0,
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    presenceCount: showPresence
      ? rawMessages.reduce(
          (total, message) =>
            message.kind === "greet" ? total + (message.presence_count ?? 1) : total,
          0
        )
      : 0,
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-network-page.ts` around lines 496 - 498, The current
presenceCount calculation counts greet rows instead of summing presence
episodes; update the presenceCount logic in the rawMessages handling (the code
that uses rawMessages.filter(message => message.kind === "greet").length) to sum
each greet item's presence_count field (e.g., use a reduce over rawMessages that
adds Number(message.presence_count) when message.kind === "greet", falling back
to 0 if missing) while preserving the showPresence conditional.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `makePeerActiveRoom()` computes `presenceCount` by counting greet rows, but peer timelines now receive collapsed greet episodes where one row can represent multiple heartbeats.
  - Root cause: the hook ignores `message.presence_count` and therefore underreports presence totals whenever repeated greets were coalesced upstream.
  - Fix plan: sum `presence_count ?? 1` across greet rows and add a route-level regression that asserts the rendered Presence toggle shows the aggregated total.
