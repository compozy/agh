---
status: resolved
file: internal/daemon/daemon_environment_sandbox_integration_test.go
line: 144
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEbT,comment:PRRC_kwDOR5y4QM640qzl
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Assert on observed tool-host diagnostics, not a synthesized struct.**

Both scenarios build `diagnostics` with the expected values inline and then call `Allowed` / `Blocked` on that same object. That means these assertions still pass if the runtime stops emitting tool-host diagnostics entirely. Please read back the captured diagnostics artifact (or the underlying runtime payload) and validate the observed data instead. As per coding guidelines, Ensure tests verify behavior outcomes, not just function calls.



Also applies to: 224-240

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_environment_sandbox_integration_test.go` around lines
125 - 144, The test currently constructs a synthetic
e2etest.ToolHostDiagnosticsArtifact (variable diagnostics) and then asserts
diagnostics.Allowed/Blocked on that same object, which doesn't verify emitted
runtime diagnostics; change the assertions to read the captured diagnostics from
harness.SessionEnvironmentArtifact (use environmentArtifact or its persisted
payload returned by SessionEnvironmentArtifact(ctx, sessionID)) and assert
Allowed/Blocked against that observed data (inspect the actual tool-host
diagnostics slice in environmentArtifact.Persisted or the runtime payload)
instead of the locally-constructed e2etest.ToolHostDiagnosticsArtifact; apply
the same fix to the other occurrence around the later block (the one referenced
as also applies to 224-240).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the final `Allowed` / `Blocked` checks are tautological because they assert against a synthetic diagnostics struct constructed in the test itself, not against an observed runtime surface.
- Fix plan: stop using the synthetic diagnostics object as test evidence and rely on the already-observed side-effect, event, and persisted-environment assertions as the source of truth. The synthetic artifact remains only as retained debug data during cleanup.
- Resolution: removed the tautological `Allowed` / `Blocked` assertions so the test now relies only on the observed side effects, failure signal, and persisted environment metadata.
- Verification: `go test ./internal/daemon` passed. `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon` was rerun but is blocked before these tests execute because the branch is missing `internal/testutil/acpmock/driver/dist/index.js`. `make verify` hits the same unrelated blocker in `internal/testutil/acpmock` and `internal/testutil/e2e`.
