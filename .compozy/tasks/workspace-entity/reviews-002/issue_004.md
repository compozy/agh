---
status: resolved
file: internal/acp/client_test.go
line: 365
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoBt,comment:PRRC_kwDOR5y4QM61T6G7
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**This “empty AdditionalDirs” test only covers `nil`.**

The setup never passes `AdditionalDirs: []string{}`, so it doesn't exercise the explicit-empty case the test name describes. Add a second case with an empty slice (or rename this one to `nil`) so nil and empty serialization can't regress independently.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client_test.go` around lines 351 - 365, The test
TestStartWithEmptyAdditionalDirsKeepsBaselinePayload only covers a nil
AdditionalDirs scenario; update it to also exercise the explicit-empty slice
case by invoking startHelperProcess with StartOpts{AdditionalDirs: []string{},
Env: helperEnvWithCapture(...)} (or add a separate subtest) so both nil and
[]string{} serialization paths are validated; reference
TestStartWithEmptyAdditionalDirsKeepsBaselinePayload, StartOpts and
startHelperProcess to locate where to add the empty-slice invocation and assert
that params does not contain "additional_dirs".
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The current test only exercises the zero-value `StartOpts` path, which means `AdditionalDirs` is `nil`.
  - An explicit `AdditionalDirs: []string{}` travels through a separate caller-side code path before normalization, and the omission contract for `additional_dirs` should be covered directly.
  - I will convert this into subtests that verify both nil and explicit-empty inputs keep the baseline payload.
