---
status: resolved
file: internal/cli/daemon_wait_test.go
line: 8
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRl,comment:PRRC_kwDOR5y4QM65B60K
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the trimmed value directly.**

This does not currently verify trimming. `strings.TrimSpace(captured.OperationID)` makes the test pass even if the command forwards the env var unchanged. Seed the env var with surrounding whitespace and compare `captured.OperationID` directly so the relaunch path is actually exercised.

<details>
<summary>Proposed fix</summary>

```diff
-	t.Setenv(aghdaemon.RestartOperationEnvKey, "restart-op-123")
+	t.Setenv(aghdaemon.RestartOperationEnvKey, "  restart-op-123 \n")
@@
-	if got, want := strings.TrimSpace(captured.OperationID), "restart-op-123"; got != want {
+	if got, want := captured.OperationID, "restart-op-123"; got != want {
 		t.Fatalf("captured.OperationID = %q, want %q", got, want)
 	}
```
</details>

As per coding guidelines, "MUST test meaningful business logic, not trivial operations".


Also applies to: 287-293

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/daemon_wait_test.go` at line 8, The test currently trims
captured.OperationID before asserting which masks whether the relaunch path
actually forwards whitespace; update the test in
internal/cli/daemon_wait_test.go to seed the environment variable with
surrounding whitespace (e.g. "  opid  "), remove the strings.TrimSpace(...)
usage, and assert captured.OperationID equals the exact string with whitespace
to ensure the command forwards the env var unchanged; apply the same change for
the other similar assertions around lines 287-293 that also use
strings.TrimSpace.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  The suggested assertion contradicts the current production contract. `internal/cli/daemon.go` intentionally passes `strings.TrimSpace(os.Getenv(aghdaemon.RestartOperationEnvKey))` into the relaunch helper, so the existing test is checking the normalized operation id, not masking a bug. Changing the test to require preserved whitespace would assert the wrong behavior.
