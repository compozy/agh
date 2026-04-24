---
status: resolved
file: internal/daemon/daemon_test.go
line: 4350
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58o_sk,comment:PRRC_kwDOR5y4QM66AfBQ
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Add compile-time interface assertions for new test doubles**

These newly added methods/types are interface-contract critical for this test path; explicit compile-time checks will prevent accidental drift.

<details>
<summary>🧩 Suggested addition</summary>

```diff
+var (
+	_ networkBindableSessionManager = (*fakeSessionManager)(nil)
+	_ syntheticPrompter             = (*fakeSessionManager)(nil)
+	_ syntheticPrompter             = nonBindableHarnessSessionManager{}
+)
```
</details>


As per coding guidelines, `**/*.go`: “Use compile-time interface verification: `var _ Interface = (*Type)(nil)`”.


Also applies to: 4476-4491

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 4333 - 4350, Add compile-time
interface assertions for the test double type to prevent future drift: add lines
like `var _ session.Manager = (*fakeSessionManager)(nil)` (or the concrete
session interface type used in production) next to the fakeSessionManager
definition to assert it implements SetNetworkPeerLifecycle, SetTurnEndNotifier,
PromptNetwork, and IsPrompting; also add the same form of assertion for any
other newly added test doubles mentioned (e.g., the types around lines
4476-4491) so the compiler verifies the interface contract.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The new daemon test doubles currently rely on implicit interface satisfaction only; there are no compile-time assertions covering `fakeSessionManager`, `fakeNetworkBindableSessionManager`, or the synthetic prompt wrapper types near the cited lines.
  - Root cause: new test helpers were added for the network-binding path without the repo-standard interface assertions that would catch future drift at compile time.
  - Fix approach: add explicit `var _ ...` assertions adjacent to the test double declarations for the production/test interfaces they are expected to satisfy.
  - Implemented: compile-time assertions now pin the session-manager, network-bindable, and synthetic-prompter test doubles to their intended interfaces.
  - Verified: focused `go test ./internal/api/httpapi ./internal/api/udsapi ./internal/api/core ./internal/config ./internal/daemon ./internal/testutil/e2e` passed, then `make verify` passed.
