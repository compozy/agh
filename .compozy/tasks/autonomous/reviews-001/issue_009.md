---
status: resolved
file: internal/api/core/conversions_parsers_test.go
line: 188
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsJ,comment:PRRC_kwDOR5y4QM67YHCi
---

# Issue 009: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use `t.Run("Should...")` subtests for these new cases.**

`TestRuntimeActivityPayloadFromSessionMeta` already bundles multiple scenarios, and the added conversion tests below are new standalone cases. Converting them to `Should...` subtests/table cases will align this file with the repo’s required test pattern.


As per coding guidelines, `**/*_test.go`: "MUST use `t.Run("Should...")` pattern for ALL test cases" and "Table-driven tests with subtests (`t.Run`) as default pattern".


Also applies to: 217-292

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/conversions_parsers_test.go` around lines 137 - 188, The
test TestRuntimeActivityPayloadFromSessionMeta bundles multiple scenarios;
refactor it into t.Run subtests following the repo pattern (use descriptive
"Should..." names) and convert the similar cases in the later block (lines
217-292) into table-driven subtests as well; specifically, create subtests for
the nil input, empty SessionLivenessMeta, and the populated SessionActivityMeta
case, using t.Run("Should ...") wrappers and a table for any repeating
assertions so each scenario is its own subtest while keeping the existing
assertions and references to RuntimeActivityPayloadFromSessionMeta,
store.SessionLivenessMeta and store.SessionActivityMeta.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestRuntimeActivityPayloadFromSessionMeta`, `TestSessionEventPayloadFromEventIncludesStopDiagnostics`, and `TestJobPayloadFromJobCopiesNestedOptionalFields` each cover distinct conversion scenarios without named `Should...` subtests. Fix by wrapping the new scenarios in subtests so failures report the exact conversion contract that regressed.
