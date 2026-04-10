---
status: resolved
file: internal/hooks/dispatch_async.go
line: 55
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrXm,comment:PRRC_kwDOR5y4QM62twdA
---

# Issue 012: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# 1) Locate pool Submit declarations/signatures.
rg -n -C3 --type=go 'func\s*\([^)]*\)\s*Submit\s*\('

# 2) Locate async pool Submit call sites to verify handling.
rg -n -C3 --type=go '\.Submit\s*\(' internal/hooks

# 3) If submit methods are interface-based, show interface declarations too.
rg -n -C3 --type=go 'type\s+\w+\s+interface\s*{[^}]*Submit\s*\('
```

Repository: compozy/agh

Length of output: 6177

---

🏁 Script executed:

```shell
#!/bin/bash
cat -n internal/hooks/pool.go | head -140 | tail -50
```

Repository: compozy/agh

Length of output: 1302

---



</details>

**Check and handle `Submit` return value failure path**

`h.pool.Submit(...)` returns `bool` and can fail (when pool is nil, closed, not started, or queue is full). The current code ignores the return value entirely, causing async hook submissions to fail silently without emitting an outcome event. Per coding guidelines, every error/failure must be handled explicitly.

When `Submit` returns `false`, the task is never queued and never executes. The pool logs "hook.dispatch.async_dropped" internally, but the caller has no visibility. Emit a hook outcome event (e.g., `HookRunOutcomeSkipped`) when submission fails, matching the pattern used for other failure paths in this function.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/hooks/dispatch_async.go` around lines 21 - 55, The call to
h.pool.Submit(asyncTask{...}) currently ignores its boolean return; if Submit
returns false the asyncTask is dropped silently — update the code after calling
h.pool.Submit to check the returned bool and when it is false call
h.emitHookRun(...) with the same parameters used for other skipped paths (use
asyncPayload, asyncHook.RegisteredHook, HookRunOutcomeSkipped, duration 0, nil
patch, appropriate err value or nil, and parentDepth) and also log/emit the
"hook.dispatch.async_dropped" outcome so callers get visibility; reference
h.pool.Submit, asyncTask, h.emitHookRun, HookRunOutcomeSkipped, asyncPayload,
asyncHook.RegisteredHook and parentDepth to locate where to add this conditional
handling.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `submitAsyncHooks` ignores the `bool` returned by `h.pool.Submit`. When the queue is full or the pool is unavailable, the hook is dropped and the caller receives no hook-run outcome record even though the system defines a `dropped` outcome for this exact case.
- Fix approach: Check the submit result, emit an explicit hook-run record for the drop path, and add a test that forces queue overflow and asserts the recorded outcome.
