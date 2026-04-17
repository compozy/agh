---
status: resolved
file: internal/testutil/acpmock/fixture_test.go
line: 369
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziM-,comment:PRRC_kwDOR5y4QM645av6
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid mutating process env alongside parallel package tests.**

This subtest sets `AGH_TEST_ACPMOCK_DRIVER_BIN`, while `TestValidationAndDriverHelpers` runs in parallel and also exercises `resolveDriverPath("")`. That makes the package order-dependent: the “default path” assertion can non-deterministically observe the env override.


As per coding guidelines, "Use t.Parallel() for independent subtests in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/fixture_test.go` around lines 360 - 369, The
subtest TestRegistrationHelperOverridesAndDiagnosticsErrors mutates the process
env (driverBinaryEnvVar) which races with TestValidationAndDriverHelpers when
tests run in parallel; fix by isolating the env mutation: extract the
env-dependent checks into their own top-level test (e.g.,
TestResolveDriverPathHonorsOverrides) that calls t.Setenv(driverBinaryEnvVar,
...) and does NOT call t.Parallel(), or alternatively ensure both tests use
t.Setenv in their own top-level tests and avoid t.Parallel() for any test that
relies on resolveDriverPath("") reading the default/override; reference
resolveDriverPath and driverBinaryEnvVar when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestRegistrationHelperOverridesAndDiagnosticsErrors` mutates `AGH_TEST_ACPMOCK_DRIVER_BIN`, while `TestValidationAndDriverHelpers` runs in parallel and also calls `resolveDriverPath("")`.
  - That creates a real process-global race where the “default path” helper assertion can observe the env override from another test.
  - Implemented: moved the override/env assertions into their own non-parallel top-level test so the parallel helper tests no longer race on process-global environment state.
  - Verification: `go test ./internal/testutil/acpmock -count=1`; `make verify`.
