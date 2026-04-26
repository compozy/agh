---
status: resolved
file: internal/task/hooks.go
line: 127
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vT,comment:PRRC_kwDOR5y4QM67Z0NO
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail
rg -n --type go 'context\.Background\(\)' internal -g '!**/*_test.go'
```

Repository: compozy/agh

Length of output: 11227

---



</details>

**Replace `context.Background()` with `context.TODO()` to comply with internal package guidelines.**

This function violates the internal package rule against `context.Background()` outside `main` and focused tests. Since this is a fallback for a nil context in a non-main internal file, `context.TODO()` is the correct placeholder.

<details>
<summary>Proposed fix</summary>

```diff
 func taskRunObservationHookContext(ctx context.Context) context.Context {
     if ctx == nil {
-        return context.Background()
+        return context.TODO()
     }
     return context.WithoutCancel(ctx)
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func taskRunObservationHookContext(ctx context.Context) context.Context {
    if ctx == nil {
        return context.TODO()
    }
    return context.WithoutCancel(ctx)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/hooks.go` around lines 123 - 127, Replace the use of
context.Background() with context.TODO() in the taskRunObservationHookContext
function: when ctx is nil, return context.TODO() instead of context.Background()
so the function uses the internal package's preferred placeholder; keep the rest
of the function (including the call to context.WithoutCancel(ctx)) unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `taskRunObservationHookContext` uses `context.Background()` in an internal production package as a nil-context fallback. Fix by returning `context.TODO()` for the nil fallback and keeping `context.WithoutCancel(ctx)` for valid caller contexts.
