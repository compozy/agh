---
status: resolved
file: internal/automation/dispatch.go
line: 552
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0M,comment:PRRC_kwDOR5y4QM623e7Y
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/automation/dispatch.go | sed -n '520,600p'
```

Repository: compozy/agh

Length of output: 3280

---

🏁 Script executed:

```shell
# Search for the hook interface and function definitions
rg "DispatchAutomationJobPostFire|DispatchAutomationRunCompleted|DispatchAutomationRunFailed" -A 2 -B 2 | head -80
```

Repository: compozy/agh

Length of output: 6632

---

🏁 Script executed:

```shell
# Look for comments about hook error handling near the hook calls
grep -B 5 -A 5 "_, _ = d.hooks" internal/automation/dispatch.go | head -60
```

Repository: compozy/agh

Length of output: 1817

---

🏁 Script executed:

```shell
# Search for hook interface definition
rg "type.*hooks.*interface|interface.*Hooks" -A 10 | head -60
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for any comments explaining hook error handling philosophy
rg "hook.*error|ignore.*hook|optional.*hook" -i internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 1211

---

🏁 Script executed:

```shell
# Check if there are comments explaining the design intent around these hook calls
sed -n '520,595p' internal/automation/dispatch.go | grep -E "^[[:space:]]*//|^[[:space:]]*/*"
```

Repository: compozy/agh

Length of output: 2615

---

🏁 Script executed:

```shell
# Check dispatchPreFireHook to see if it handles errors differently
sed -n '485,525p' internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 1453

---



</details>

**Handle lifecycle hook failures explicitly.**

Four hook dispatch calls use `_, _ =` to ignore errors: `DispatchAutomationJobPostFire` (line 533), `DispatchAutomationTriggerPostFire` (line 542), `DispatchAutomationRunCompleted` (line 559), and `DispatchAutomationRunFailed` (line 579). This causes post-fire and lifecycle hook failures to disappear silently, allowing downstream automation integrations to drift without any signal.

Per coding guidelines: "Never ignore errors with `_` — every error must be handled or have a written justification". Contrast this with `dispatchPreFireHook`, which properly checks hook errors with `if err != nil` before returning. Either log hook dispatch failures or return them from these functions.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 526 - 552, These post-fire hook
calls currently ignore returned errors (in dispatchPostFireHook and the other
post/lifecycle dispatches) causing failures to be silent; capture the error
returned by each hook call (e.g., the calls to DispatchAutomationJobPostFire,
DispatchAutomationTriggerPostFire, DispatchAutomationRunCompleted,
DispatchAutomationRunFailed), then handle it instead of using "_, _ =" — at
minimum call the dispatcher logger (e.g., d.logger.Errorf or similar logger on
the Dispatcher) with a clear message and error details including the relevant
IDs (JobID/TriggerID/RunID), or propagate the error by returning it from the
function if the call site expects error handling. Ensure you replace each "_, _
=" usage with proper err checking (if err != nil { ... }) referencing the
specific function names above.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `dispatchPostFireHook` and `emitRunLifecycleHooks` currently discard hook-dispatch errors with `_, _ =`, which hides downstream hook failures entirely. I will handle those errors explicitly by logging the failure with the relevant automation identifiers and add dispatcher-level tests for the new behavior.
