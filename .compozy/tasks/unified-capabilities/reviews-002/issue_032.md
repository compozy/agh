---
status: resolved
file: internal/subprocess/signals_windows.go
line: 40
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyy9,comment:PRRC_kwDOR5y4QM654Np9
---

# Issue 032: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Force-exit helper should not silently succeed when unsupported**

Line 38-Line 40 unconditionally return `nil`. For a force-termination API, this can mask failure and leave child processes running while upper layers assume cleanup succeeded.

Please align this function with an explicit unsupported error (or a real best-effort implementation) instead of returning success.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/subprocess/signals_windows.go` around lines 35 - 40, The function
forceManagedProcessGroupExit currently returns nil which hides unsupported
behavior; change it to return a clear explicit error (e.g. define and return a
package-level sentinel like ErrForceExitUnsupported or return
errors.New("forceManagedProcessGroupExit: not supported on Windows")) so callers
cannot assume success, update signals_windows.go to return that error from
forceManagedProcessGroupExit, and ensure higher-level callers (functions/methods
that invoke forceManagedProcessGroupExit) handle the non-nil error path
accordingly and/or document the unsupported status.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  On Windows, `forceManagedProcessGroupExit()` currently returns `nil` even though the helper explicitly lacks process-group parity. In `Process.waitForExit()`, a stop-requested process treats that return value as success, so upper layers can believe the whole tree exited cleanly when that guarantee was never enforced.
  I will return an explicit unsupported error from the Windows helper; `waitForExit()` already propagates non-nil results for stop-requested shutdowns, so callers will see the limitation instead of a false success.
  Fixed and verified with targeted package tests plus `make verify`.
