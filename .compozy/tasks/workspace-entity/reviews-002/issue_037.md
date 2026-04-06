---
status: resolved
file: web/src/systems/workspace/adapters/workspace-api.ts
line: 35
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDR,comment:PRRC_kwDOR5y4QM61T6I_
---

# Issue 037: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Give this adapter a typed error surface.**

Both non-OK branches throw raw `Error`, and schema parsing can still bubble a raw `ZodError`. That leaves hooks/UI without a stable way to distinguish HTTP failures from invalid response shapes or inspect the status code.

<details>
<summary>🛠️ Proposed fix</summary>

```diff
+export class WorkspaceApiError extends Error {
+  constructor(
+    message: string,
+    public readonly status: number,
+    public readonly cause?: unknown
+  ) {
+    super(message);
+    this.name = "WorkspaceApiError";
+  }
+}
+
 export async function fetchWorkspaces(signal?: AbortSignal): Promise<WorkspacePayload[]> {
   const res = await fetch("/api/workspaces", { signal });
   if (!res.ok) {
-    throw new Error(`Failed to fetch workspaces: ${res.status}`);
+    const message = (await res.text()).trim() || "Failed to fetch workspaces";
+    throw new WorkspaceApiError(message, res.status);
   }
 
   const json = await res.json();
-  const parsed = workspacesResponseSchema.parse(json);
-  return parsed.workspaces;
+  try {
+    return workspacesResponseSchema.parse(json).workspaces;
+  } catch (error) {
+    throw new WorkspaceApiError("Invalid workspaces response", res.status, error);
+  }
 }
@@
 export async function resolveWorkspace(
   params: ResolveWorkspaceParams,
   signal?: AbortSignal
 ): Promise<WorkspacePayload> {
@@
   });
   if (!res.ok) {
-    throw new Error(`Failed to resolve workspace: ${res.status}`);
+    const message = (await res.text()).trim() || "Failed to resolve workspace";
+    throw new WorkspaceApiError(message, res.status);
   }
 
   const json = await res.json();
-  const parsed = workspaceResponseSchema.parse(json);
-  return parsed.workspace;
+  try {
+    return workspaceResponseSchema.parse(json).workspace;
+  } catch (error) {
+    throw new WorkspaceApiError("Invalid workspace response", res.status, error);
+  }
 }
```
</details>

As per coding guidelines, `web/src/systems/*/adapters/*.ts`: `Define typed error classes in adapters — never throw raw errors in API service layer`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/workspace/adapters/workspace-api.ts` around lines 3 - 35,
Create typed adapter errors and throw them instead of raw Error/ZodError: add
two exported classes (e.g., WorkspaceApiHttpError and WorkspaceApiParseError)
and update fetchWorkspaces and resolveWorkspace to catch non-OK responses and
parsing failures, wrapping HTTP failures with WorkspaceApiHttpError (include
status and response text/body) and Zod parse failures with
WorkspaceApiParseError (include original ZodError). Ensure the functions still
return WorkspacePayload(s), preserve existing behavior for successful responses,
and update error throws in workspaceResponseSchema.parse and
workspacesResponseSchema.parse paths to rethrow the wrapped errors so callers
can distinguish HTTP vs. schema issues.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  The frontend does not currently define a shared typed adapter-error taxonomy,
  and current workspace consumers only surface message strings. Adding bespoke
  HTTP/parse error classes in this single adapter would create an isolated
  pattern without caller integration or broader consistency. No concrete bug to
  fix in this batch.
