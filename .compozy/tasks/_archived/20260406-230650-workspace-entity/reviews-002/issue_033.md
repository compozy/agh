---
status: resolved
file: web/src/systems/session/adapters/session-api.ts
line: 23
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDD,comment:PRRC_kwDOR5y4QM61T6Io
---

# Issue 033: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Treat blank workspace filters as "no filter".**

`fetchSessions("")` currently calls `/api/sessions?workspace=` instead of the unfiltered endpoint. That creates a different request for what is usually the "all workspaces" state and can diverge from backend handling.


<details>
<summary>Suggested fix</summary>

```diff
 export async function fetchSessions(
   workspace?: string,
   signal?: AbortSignal
 ): Promise<SessionPayload[]> {
+  const normalizedWorkspace = workspace?.trim();
   const url =
-    workspace == null
+    normalizedWorkspace == null || normalizedWorkspace === ""
       ? "/api/sessions"
-      : `/api/sessions?workspace=${encodeURIComponent(workspace)}`;
+      : `/api/sessions?workspace=${encodeURIComponent(normalizedWorkspace)}`;
   const res = await fetch(url, { signal });
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/adapters/session-api.ts` around lines 15 - 23, The
fetchSessions function treats an empty string workspace as a valid filter,
causing requests to hit /api/sessions?workspace=; modify fetchSessions so that
blank strings are treated as "no filter" by checking workspace for
null/undefined or empty (e.g., workspace == null || workspace === "") before
building the URL, and only append the ?workspace=... query when workspace is a
non-empty string; update the URL construction logic in fetchSessions
accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `fetchSessions` currently treats empty or whitespace-only workspace strings as
  real filters and generates `/api/sessions?workspace=`. That should normalize
  to the unfiltered endpoint. Plan: trim the input before URL construction and
  treat blank values the same as `undefined`.
