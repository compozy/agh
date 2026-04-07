---
status: resolved
file: web/src/systems/session/lib/query-keys.ts
line: 4
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDL,comment:PRRC_kwDOR5y4QM61T6Iz
---

# Issue 035: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid `"all"` as the unfiltered cache-key sentinel.**

A real workspace named `"all"` now collides with the unfiltered sessions list, so switching between that workspace and the global view can reuse the wrong cache entry. Keep the empty state as `null` instead of a user-controlled string.


<details>
<summary>Suggested fix</summary>

```diff
 export const sessionKeys = {
   all: ["sessions"] as const,
   lists: () => [...sessionKeys.all, "list"] as const,
-  list: (workspace: string | null = null) => [...sessionKeys.lists(), workspace ?? "all"] as const,
+  list: (workspace: string | null = null) =>
+    [...sessionKeys.lists(), workspace?.trim() || null] as const,
   detail: (id: string) => [...sessionKeys.all, "detail", id] as const,
   events: (id: string) => [...sessionKeys.all, "detail", id, "events"] as const,
```
</details>
As per coding guidelines, "Maintain hierarchical query keys in `lib/query-keys.ts` for granular cache invalidation".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/lib/query-keys.ts` around lines 3 - 4, The cache key
sentinel currently uses the string "all" which can collide with a real workspace
named "all"; update the sessionKeys functions to use null as the unfiltered
sentinel instead: modify sessionKeys.list to return [...sessionKeys.lists(),
workspace ?? null] (keeping sessionKeys.lists and sessionKeys.all hierarchical
shape) and ensure any consumers expect null for the global/unfiltered key so
hierarchical query keys in lib/query-keys.ts remain consistent for granular
cache invalidation.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  The current frontend passes workspace IDs into `sessionKeys.list(...)` via
  `useSessions(...)`, and AGH workspace IDs are generated `ws_*` identifiers,
  not user-controlled names like `"all"`. That means the reported cache-key
  collision does not occur in the current call graph. No change in this batch.
