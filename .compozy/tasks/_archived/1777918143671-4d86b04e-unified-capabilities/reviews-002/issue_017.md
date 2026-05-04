---
status: resolved
file: internal/hooks/executor_subprocess_windows.go
line: 39
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyyR,comment:PRRC_kwDOR5y4QM654NpB
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find definitions and call sites of force-exit helpers.
rg -nP --type=go -C3 '\b(forceSubprocessCommandExit|forceManagedProcessGroupExit)\s*\('

# Spot potential ignored errors at call sites.
rg -nP --type=go -C2 '(_\s*=\s*)?(forceSubprocessCommandExit|forceManagedProcessGroupExit)\s*\('
```

Repository: compozy/agh

Length of output: 8962

---



</details>

**Return explicit error for unsupported platform instead of false success**

The Windows implementation at lines 37–39 returns `nil` unconditionally, which reports false success to callers even though no process termination occurs. Call sites at `internal/hooks/executor_subprocess.go:170` and line 178 use `errors.Join()` to combine the result with other errors — if the force-exit returns `nil` and other errors are also `nil`, callers will see overall success despite Windows not supporting process-group termination.

This violates the guideline requiring explicit error returns with context. Return a sentinel error instead:

<details>
<summary>Proposed change</summary>

```diff
 import (
 	"errors"
+	"fmt"
 	"os"
 	"os/exec"
 	"time"
 )
+
+var errForceSubprocessCommandExitUnsupported = errors.New("force subprocess command exit is not supported on windows")
@@
 func forceSubprocessCommandExit(_ *exec.Cmd, _ time.Duration) error {
-	return nil
+	return fmt.Errorf("force subprocess command exit: %w", errForceSubprocessCommandExitUnsupported)
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/hooks/executor_subprocess_windows.go` around lines 35 - 39, Replace
the unconditional nil return in forceSubprocessCommandExit with an explicit
sentinel error indicating the operation is unsupported on Windows (e.g., a
package-level var ErrUnsupportedPlatform =
errors.New("forceSubprocessCommandExit: unsupported on Windows") or a
descriptive fmt.Errorf result) so callers that errors.Join the result won't
observe false success; update forceSubprocessCommandExit to return that error
and ensure the error text clearly names the function/operation for easy
diagnosis.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Root cause analysis: this helper has the same intentional Windows no-op behavior as the ACP variant. Returning a sentinel error from only this leaf function would turn every canceled/stopped Windows hook subprocess into an error path even when the direct process exited correctly, without adding real process-tree cleanup.
- Why not fix here: a safe fix requires a broader Windows process-tree strategy or caller contract change, not a leaf-level error flip in this batch.
- Resolution: analysis completed; no safe self-contained code change exists for this batch.
