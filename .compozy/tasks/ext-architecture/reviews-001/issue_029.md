---
status: resolved
file: internal/extension/reference_support_unit_test.go
line: 52
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAap,comment:PRRC_kwDOR5y4QM62zls1
---

# Issue 029: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Refactor these new tests to required subtest style and stronger error assertions.**

Please convert these cases to table-driven `t.Run("Should...")` subtests and make the invalid-JSON assertion specific (message/content), not only non-nil.

As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases`, `Use table-driven tests with subtests (t.Run) as default in Go tests`, and `MUST have specific error assertions (ErrorContains, ErrorAs)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/reference_support_unit_test.go` around lines 5 - 52,
Refactor the three tests (TestNonEmptyLines, TestContainsFragmentsInOrder,
TestDecodeJSONLines) into table-driven subtests using t.Run with descriptive
"Should..." names for each case; for nonEmptyLines and containsFragmentsInOrder
create a table of inputs/expected outputs and iterate with t.Run per row, and
for decodeJSONLines likewise use subtests for valid cases and a specific
invalid-JSON case where you assert the error content (use ErrorContains or check
err.Error() includes the expected substring) instead of only checking non-nil;
locate usages of nonEmptyLines, containsFragmentsInOrder and decodeJSONLines in
those tests to update them.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The referenced tests are currently single-path assertions and the invalid-JSON case only checks for non-nil error. Strengthening them makes failures more diagnostic and aligns them with the repo’s test conventions.
  Fix approach: refactor the helper tests into table-driven `t.Run` subtests and assert specific invalid-JSON error content.
