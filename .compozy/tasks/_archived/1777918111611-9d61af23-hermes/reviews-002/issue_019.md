---
status: resolved
file: internal/retry/retry_test.go
line: 274
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLij,comment:PRRC_kwDOR5y4QM67SmDq
---

# Issue 019: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Mark test helper functions with `t.Helper()`.**

Line 248-274 defines helper utilities used by tests; wiring `*testing.T` into these helpers and calling `t.Helper()` will improve failure locations.


As per coding guidelines, "Add `t.Helper()` on test helper functions in Go".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/retry/retry_test.go` around lines 248 - 274, The three test helper
functions sequenceRand, equalDurations, and nilRetryContext should be converted
to accept a *testing.T parameter and must call t.Helper() at the top so test
failures point to the caller; update their signatures to sequenceRand(t
*testing.T, values ...float64), equalDurations(t *testing.T, left, right
[]time.Duration), and nilRetryContext(t *testing.T) with t.Helper() as the first
statement in each, and then update all test callers to pass the *testing.T
value.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `sequenceRand`, `equalDurations`, and `nilRetryContext` are test helpers but do not accept `*testing.T` or call `t.Helper()`, so failures inside or near helpers point at helper internals.
- Fix approach: thread `*testing.T` into those helpers, call `t.Helper()`, and update callers.
