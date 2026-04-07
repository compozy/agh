---
status: resolved
file: web/src/systems/workspace/adapters/workspace-api.ts
line: 1
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDN,comment:PRRC_kwDOR5y4QM61T6I3
---

# Issue 036: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use the `@/` alias for this import.**

This new relative import is out of step with the frontend import convention.

<details>
<summary>♻️ Proposed fix</summary>

```diff
-import { workspaceResponseSchema, workspacesResponseSchema, type WorkspacePayload } from "../types";
+import {
+  workspaceResponseSchema,
+  workspacesResponseSchema,
+  type WorkspacePayload,
+} from "@/systems/workspace/types";
```
</details>

As per coding guidelines, `web/src/**/*.{ts,tsx}`: `Use path alias `@/`* mapping to ./src/* for all imports`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import {
  workspaceResponseSchema,
  workspacesResponseSchema,
  type WorkspacePayload,
} from "@/systems/workspace/types";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/workspace/adapters/workspace-api.ts` at line 1, The import in
workspace-api.ts uses a relative path; update it to use the project alias
mapping (`@/`) instead of a relative path so it follows the frontend convention —
replace the current import that brings in workspaceResponseSchema,
workspacesResponseSchema and WorkspacePayload from "../types" with an equivalent
import via the "@/..." alias (matching the ./src/* mapping) that points to the
same types module; ensure the imported symbol names remain unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  The relative import resolves correctly, is type-safe, and does not affect
  runtime behavior. Rewriting it to use the path alias would be a style-only
  edit with no functional impact or verification gap. No change.
