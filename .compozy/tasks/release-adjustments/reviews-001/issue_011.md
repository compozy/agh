---
status: resolved
file: internal/network/delivery_test.go
line: 525
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1k,comment:PRRC_kwDOR5y4QM67HMWx
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Run the retry-delay cases as named subtests.**

This block is already table-driven, so wrapping each case in `t.Run("Should...")` with `t.Parallel()` will align it with the repo’s test contract and make the failing attempt obvious when it regresses.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests", "Add `t.Parallel()` to independent subtests in Go tests", and "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/delivery_test.go` around lines 502 - 525,
TestDeliveryCoordinatorRetryDelayUsesExponentialCap should run each table-driven
case as a named subtest: replace the direct loop over cases with per-case t.Run
calls (use the "Should ..." naming pattern) and call t.Parallel() inside each
subtest to run them concurrently; keep using the same
coordinator.retryDelayFor(...) and the same expected tc.want assertions but move
the comparison and t.Fatalf into the subtest body so failures show the specific
attempt name. Ensure you capture tc in the closure to avoid loop variable
capture and preserve the existing test logic and assertions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `TestDeliveryCoordinatorRetryDelayUsesExponentialCap` is table-driven but evaluates every case in a plain loop.
  - The fix is to run each case as an independent `Should...` subtest with `t.Parallel()` and retain the same retry delay assertions.
