---
status: resolved
file: internal/daemon/restart.go
line: 541
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kSG,comment:PRRC_kwDOR5y4QM65B603
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Surface the fallback persistence failure too.**

If `store.Transition(...Failed...)` fails here, the caller only sees the original action error even though the durable restart record may still be stuck in a non-terminal state. That makes restart-status polling lie about the final outcome.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 func failRestartOperation(
 	store *restartStore,
 	operation RestartOperation,
 	action string,
 	err error,
 ) (RestartOperation, error) {
 	if err == nil {
 		return operation, nil
 	}
 
 	failed, transitionErr := store.Transition(operation.OperationID, restartTransition{
 		status:        RestartStatusFailed,
 		failureReason: fmt.Sprintf("%s: %v", action, err),
 	})
 	if transitionErr == nil {
 		operation = failed
+		return operation, fmt.Errorf("daemon: %s: %w", action, err)
 	}
-	return operation, fmt.Errorf("daemon: %s: %w", action, err)
+	return operation, errors.Join(
+		fmt.Errorf("daemon: %s: %w", action, err),
+		fmt.Errorf("daemon: persist failed restart operation %q: %w", operation.OperationID, transitionErr),
+	)
 }
```

</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/restart.go` around lines 524 - 541, failRestartOperation
currently hides errors from store.Transition, causing callers to miss
persistence failures; change it to surface transitionErr when store.Transition
fails by wrapping both errors into the returned error. Concretely, in
failRestartOperation (and around the call to store.Transition with
restartTransition{status: RestartStatusFailed, failureReason: fmt.Sprintf("%s:
%v", action, err)}), if transitionErr != nil include it in the final fmt.Errorf
(e.g., "daemon: %s: %w; persistence transition error: %v" or by wrapping
transitionErr) instead of discarding it, and only set operation = failed when
transitionErr == nil so the caller sees both the original action error and the
fallback persistence error.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `failRestartOperation`: if persisting the fallback `Failed` transition also fails, the returned error drops that second failure and only exposes the original action error. I will join both errors so callers can see that the restart operation may still be stuck in a non-terminal persisted state.
