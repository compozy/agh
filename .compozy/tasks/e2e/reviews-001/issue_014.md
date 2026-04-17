---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 292
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEb6,comment:PRRC_kwDOR5y4QM640q0V
---

# Issue 014: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Same error string comparison issue.**

Apply the same `errors.Is()` pattern here for consistency with guidelines.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 286 - 292, The test
currently compares the error string from syncer.Sync() to the literal "provider
failure"; change this to use errors.Is for robust comparison (e.g., replace the
string equality check with errors.Is(err, providerFailureErr)) and import the
standard errors package if needed; reference the syncer.Sync call and the
sentinel error (providerFailureErr or whatever sentinel is used elsewhere for
the provider failure) so the test asserts errors.Is(err, providerFailureErr)
instead of comparing err.Error() to a literal.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: this provider-failure test uses the same brittle string-equality pattern as issue 013.
- Fix plan: reuse a sentinel provider error and assert the wrapped result with `errors.Is`.
- Resolution: introduced a real sentinel provider error and switched the assertion to `errors.Is`.
- Verification: `go test ./internal/daemon` passed. `make verify` was rerun after the fix set and still fails in unrelated pre-existing `internal/testutil/acpmock` and `internal/testutil/e2e` packages because this branch does not contain `internal/testutil/acpmock/driver/dist/index.js`.
