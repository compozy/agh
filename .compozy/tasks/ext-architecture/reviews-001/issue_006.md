---
status: resolved
file: internal/acp/client.go
line: 356
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaD,comment:PRRC_kwDOR5y4QM62zlsO
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Inspect subprocess shutdown/wait semantics.
rg -n -C12 'type Process struct|func Launch|func \(.*\*Process\) Shutdown|func \(.*\*Process\) Wait' internal/subprocess

# Inspect the ACP managed shutdown path that depends on those semantics.
rg -n -C8 'proc\.managed\.Shutdown|proc\.Wait\(|func \(p \*AgentProcess\) waitForExit' internal/acp/client.go
```

Repository: compozy/agh

Length of output: 7840

---

🏁 Script executed:

```shell
sed -n '293,330p' internal/subprocess/process.go
```

Repository: compozy/agh

Length of output: 1102

---

🏁 Script executed:

```shell
rg -n 'func.*waitWithContext' internal/subprocess/process.go -A 20
```

Repository: compozy/agh

Length of output: 627

---

🏁 Script executed:

```shell
sed -n '330,365p' internal/subprocess/process.go
```

Repository: compozy/agh

Length of output: 1115

---



</details>

**Keep the managed stop path context-bounded.**

If `proc.managed.Shutdown(ctx)` returns an error due to context timeout or cancellation, `Stop` unconditionally calls `proc.Wait()` which blocks indefinitely if the process is still running. The non-managed path below correctly uses `select` with `ctx.Done()` to respect the context boundary, but the managed path does not. This violates the coding guideline requiring context boundaries for functions crossing runtime boundaries.

```diff
if proc.managed != nil {
    if err := proc.managed.Shutdown(ctx); err != nil {
        errs = append(errs, err)
    }
+   select {
+   case <-proc.Done():
+   case <-ctx.Done():
+       return errors.Join(append(errs, ctx.Err())...)
+   }
    return errors.Join(append(errs, proc.Wait())...)
}
```

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client.go` around lines 352 - 356, The managed-stop branch in
Stop should respect the provided context: after calling
proc.managed.Shutdown(ctx) (in the managed path) do not call proc.Wait()
unconditionally; instead start proc.Wait() in a goroutine and use a select that
waits for either the wait result or ctx.Done(), appending the wait error only if
it completes before context cancellation; ensure you still collect and join errs
with any shutdown error (from proc.managed.Shutdown) and the proc.Wait() error
when available so the method remains context-bounded.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: In the managed-process stop path, `proc.managed.Shutdown(ctx)` can return because the caller's context expired while the subprocess is still alive, and the subsequent unconditional `proc.Wait()` blocks forever. I will make the managed wait path respect `ctx.Done()` and cover it with a regression test.
