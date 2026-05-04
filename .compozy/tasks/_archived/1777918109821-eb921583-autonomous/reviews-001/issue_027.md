---
status: resolved
file: internal/cli/cli_integration_test.go
line: 1599
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsc,comment:PRRC_kwDOR5y4QM67YHC2
---

# Issue 027: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Avoid fixed sleeps for lease expiry.**

A hard-coded 1.5s sleep makes this integration test timing-sensitive under CI load. Prefer polling for the expired/stale condition or injecting controllable time into the lease path.



As per coding guidelines, `**/*.go`: "No time.Sleep() in orchestration — use proper synchronization primitives."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/cli_integration_test.go` around lines 1598 - 1599, The test uses
a hard-coded time.Sleep(1500 * time.Millisecond) after capturing staleToken
(staleNext.Claim.ClaimToken), which makes the test timing-sensitive; replace
this fixed sleep with a polling loop that repeatedly checks the lease/claim
state (for example by calling the same helper/CLI check used elsewhere in the
test suite to assert a claim is expired or marked stale) until the expected
condition is true or a reasonable timeout elapses; implement the poll using a
ticker and context timeout (avoid blocking the test thread), then proceed once
the lease is observed expired/stale instead of sleeping.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The stale lease path uses `time.Sleep(1500 * time.Millisecond)` after claiming a one-second lease. That fixed wait is timing-sensitive and does not explicitly wait for the observed lease expiry boundary.
- Fix: Replace the fixed sleep with a context/ticker polling helper that waits until the claimed lease's `lease_until` timestamp is actually expired or a bounded timeout elapses.
