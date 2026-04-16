---
status: resolved
file: packages/site/components/docs/mermaid.tsx
line: 20
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAU,comment:PRRC_kwDOR5y4QM64gE5W
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reset the cached loader promise on import failure.**

At Line 9, a failed dynamic import is cached permanently. One transient failure can break all future Mermaid renders until a full page reload.



<details>
<summary>💡 Proposed fix</summary>

```diff
 function loadMermaid() {
   if (!mermaidLoader) {
-    mermaidLoader = import("mermaid").then(({ default: mermaid }) => {
-      mermaid.initialize({
-        startOnLoad: false,
-        securityLevel: "strict",
-        theme: "dark",
-      });
-      return mermaid;
-    });
+    mermaidLoader = import("mermaid")
+      .then(({ default: mermaid }) => {
+        mermaid.initialize({
+          startOnLoad: false,
+          securityLevel: "strict",
+          theme: "dark",
+        });
+        return mermaid;
+      })
+      .catch(error => {
+        mermaidLoader = null;
+        throw error;
+      });
   }
 
   return mermaidLoader;
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
function loadMermaid() {
  if (!mermaidLoader) {
    mermaidLoader = import("mermaid")
      .then(({ default: mermaid }) => {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "strict",
          theme: "dark",
        });
        return mermaid;
      })
      .catch(error => {
        mermaidLoader = null;
        throw error;
      });
  }

  return mermaidLoader;
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/docs/mermaid.tsx` around lines 7 - 20, The dynamic
import in loadMermaid currently assigns a failing promise to the module-level
mermaidLoader and never clears it, so any transient import error blocks future
attempts; update loadMermaid to attach a .catch handler to the import promise
(the promise assigned to mermaidLoader in the import("mermaid").then(...) chain)
that sets mermaidLoader = undefined before rethrowing or returning the error so
subsequent calls will retry the import; reference the loadMermaid function and
the mermaidLoader variable when making this change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `loadMermaid()` memoizes the promise stored in `mermaidLoader`, so an import failure leaves a rejected promise cached for the rest of the browser session.
  - Root cause: the loader promise is never reset on rejection, which prevents later retries after transient module-load failures.
  - Fix plan: attach a rejection handler that clears `mermaidLoader` before rethrowing, then add a component test that fails once and succeeds on a retry.
  - Resolution: the Mermaid loader now clears `mermaidLoader` on rejection, and `packages/site/components/docs/mermaid.test.tsx` covers the retry-after-failure path.
  - Verification: `packages/site` `bun run test`, `bun run typecheck`, and `bun run build` passed.
