---
status: resolved
file: internal/automation/manager.go
line: 387
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0S,comment:PRRC_kwDOR5y4QM623e7e
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's check the file exists and examine the problematic lines
wc -l internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 90

---

🏁 Script executed:

```shell
# Check lines 365-387
sed -n '365,387p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 828

---

🏁 Script executed:

```shell
# Check lines 495-505
sed -n '495,505p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 341

---

🏁 Script executed:

```shell
# Check lines 541-545 (expanding slightly from 541-541)
sed -n '541,545p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 270

---

🏁 Script executed:

```shell
# Check lines 665-681
sed -n '665,681p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 679

---

🏁 Script executed:

```shell
# Check broader context around 365-387 - look at the full function
sed -n '355,395p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 1179

---

🏁 Script executed:

```shell
# Check broader context around 495-505
sed -n '485,515p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 805

---

🏁 Script executed:

```shell
# Check broader context around 665-681
sed -n '655,690p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 1080

---

🏁 Script executed:

```shell
# Also check what these shutdown/delete methods return
rg -A 3 "func.*Shutdown\(" internal/automation/manager.go | head -20
```

Repository: compozy/agh

Length of output: 202

---

🏁 Script executed:

```shell
# Check if there's an error handling pattern or issue tracker comment
rg -B 2 "_ = scheduler.Shutdown|_ = triggerEngine.Shutdown" internal/automation/manager.go | head -30
```

Repository: compozy/agh

Length of output: 913

---



</details>

**Handle cleanup failures explicitly or document the rationale.**

Errors from `Shutdown()`, `DeleteJob()`, `Unregister()`, and `DeleteTrigger()` are being discarded with `_` at multiple locations (lines 365–387, 495–505, 541–545, 665–681) without justification, violating the coding guideline: "Never ignore errors with `_` — every error must be handled or have a written justification." When cleanup operations fail silently, persisted state and runtime registrations can diverge, leaving the system in an inconsistent state.

Additionally, the `Shutdown()` calls at lines 365–387 use `context.Background()` in non-main functions, which also violates the guideline: "avoid `context.Background()` outside `main` and focused tests."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/manager.go` around lines 365 - 387, The cleanup calls
currently swallow errors and use context.Background(); update the error handling
in the startup/cleanup sequence around m.loadSchedulerRegistrations,
m.loadTriggerRegistrations, triggerEngine.Start and scheduler.Start so that
Shutdown(...) returns are checked and handled (log the error and/or
aggregate/return a wrapped error) instead of using `_`, and replace
context.Background() with the provided ctx or a derived context with
timeout/cancellation; ensure runtimeCancel() is still called, then call
scheduler.Shutdown(ctx) and triggerEngine.Shutdown(ctx) (or ctxWithTimeout) and
propagate or log any shutdown error rather than discarding it.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Several manager startup/rollback paths still discard cleanup failures and use `context.Background()` during shutdown/rollback work, which can hide divergence between persisted state and runtime registrations. I will replace those ignored calls with explicit cleanup helpers that use the active context lineage, return or join cleanup failures where appropriate, and log non-fatal teardown errors instead of dropping them.
