---
status: resolved
file: internal/observe/observer_test.go
line: 617
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLih,comment:PRRC_kwDOR5y4QM67SmDo
---

# Issue 017: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**This test can pass even if `Failures.Recent` is unsorted.**

The newer failure (`sess-crash`) is also registered first, so `Recent[0] == "sess-crash"` still succeeds when the implementation preserves registry order instead of sorting by `UpdatedAt`. Register the older failure first, or assert the full descending order from deliberately scrambled insert order.

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/observer_test.go` around lines 556 - 617, The test relies on
Failures.Recent being sorted by UpdatedAt but currently registers the newer
"sess-crash" first so the test can pass even if no sorting occurs; update the
test setup in observer_test.go to register the older session before the newer
one (i.e., call h.registry.RegisterSession for "sess-protocol" before
"sess-crash") or, better, scramble insertion order deliberately and then assert
that observer.Health() returns Failures.Recent sorted descending by UpdatedAt
(check h.observer.Health(), health.Failures.Recent, and the
SessionID/FailureKind ordering) so the test actually verifies the sorting
behavior rather than registry insertion order.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the failure-health test registers the newer session first, so an implementation that merely preserves registry order can still pass the `Recent[0]` assertion.
- Fix approach: deliberately register failures in non-recency order and assert the full descending `UpdatedAt` order in `Failures.Recent`.
