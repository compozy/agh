---
status: resolved
file: internal/store/globaldb/global_db_extra_test.go
line: 740
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPQ,comment:PRRC_kwDOR5y4QM6628D5
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Adopt required `t.Run("Should...")` subtest pattern for these new test cases.**

Line 679 and Line 742 add direct top-level test flows, but this repo requires table-driven subtests with `t.Run("Should...")` as the default for test cases.



As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".


Also applies to: 742-825

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_extra_test.go` around lines 679 - 740, Wrap
the existing top-level test body of
TestMigrateSessionColumnsAddsProviderIdempotently in a t.Run subtest using the
required pattern (e.g. t.Run("Should add provider column idempotently", func(t
*testing.T) { ... })), move t.Parallel() into that subtest, and keep all setup,
DB operations and assertions unchanged inside the subtest; do the same for the
other similar test(s) referenced (the test starting at line ~742) so each
original test flow becomes a t.Run("Should...") subtest while preserving
function names (TestMigrateSessionColumnsAddsProviderIdempotently and the other
test) and behavior. Ensure any per-test cleanup (t.Cleanup) remains inside the
subtest so resources are scoped correctly.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: The two new migration tests are top-level flows and should follow the repository’s `t.Run("Should...")` convention. I will wrap each test body in a single descriptive subtest and keep their setup/cleanup local to the subtest.
