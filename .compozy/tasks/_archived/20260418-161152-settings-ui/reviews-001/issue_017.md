---
status: resolved
file: internal/daemon/restart.go
line: 801
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kSS,comment:PRRC_kwDOR5y4QM65B61G
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Persist `Failed` when status polling itself breaks.**

A read failure here exits the helper without marking the operation terminal, so clients can keep polling a stale `starting` state forever. This path should go through `h.fail(...)` like the earlier restart phases.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 		case <-ticker.C:
 			operation, err := store.Get(operationID)
 			if err != nil {
-				return err
+				return h.fail(
+					store,
+					operationID,
+					fmt.Errorf("daemon: load restart operation %q: %w", operationID, err),
+				)
 			}
 			switch operation.Status {
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
			operation, err := store.Get(operationID)
			if err != nil {
				return h.fail(
					store,
					operationID,
					fmt.Errorf("daemon: load restart operation %q: %w", operationID, err),
				)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/restart.go` around lines 799 - 801, The Get call error
handling near "operation, err := store.Get(operationID)" currently returns err
and exits without marking the operation terminal; change this path to call the
helper's failure handler (h.fail(...)) so the operation is persisted as Failed
(e.g., invoke h.fail(ctx, operationID, fmt.Errorf("status polling read failed:
%w", err)) or the existing h.fail signature used elsewhere), propagate the
result of h.fail back to the caller, and preserve the original error details in
the failure message; replace the bare "return err" at that location with the
h.fail invocation matching other restart phases.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `relaunchHelper.waitForReady`: a `store.Get` failure during polling returns immediately without attempting to persist `RestartStatusFailed`, leaving clients stuck on `starting`. I will route this path through the helper’s failure persistence flow so polling-read failures are durably terminal.
