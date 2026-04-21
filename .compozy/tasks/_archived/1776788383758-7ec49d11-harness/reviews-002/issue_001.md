---
status: resolved
file: internal/daemon/daemon_integration_test.go
line: 907
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uT8,comment:PRRC_kwDOR5y4QM65IlO9
---

# Issue 001: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify new detached-harness tests are mostly top-level and not using the required subtest naming pattern.
rg -nP '^func Test(BootWiresDetachedHarnessTaskRuntimeAcrossScopes|DetachedHarnessCompletion|BootRecoveryDetachedHarness|BootRecoversDetachedHarness)' internal/daemon/daemon_integration_test.go
rg -n 't\.Run\("Should' internal/daemon/daemon_integration_test.go
```

Repository: compozy/agh

Length of output: 567

---

🏁 Script executed:

```shell
sed -n '3257,3390p' internal/daemon/daemon_integration_test.go
```

Repository: compozy/agh

Length of output: 3690

---



</details>

**Refactor detached harness tests into table-driven subtests with `t.Run("Should...")` pattern.**

The six new test functions (lines 221–907) repeat the same scaffold pattern—setup homePaths/cfg/sessions/daemon, configure infos, submit work, wait for completion, verify assertions—making them ideal for parameterization. Per test review requirements, use a single parent test with `t.Run("Should...")` subtests for each scenario instead of six separate top-level functions. This reduces duplication and aligns with repository test standards.

Additionally, wrap errors in `ensureDetachedHarnessWorkspaceIndex` using `fmt.Errorf("context: %w", err)` at lines 3334, 3339, and 3341 for consistency with Go error-handling conventions.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_integration_test.go` around lines 221 - 907, The six
top-level tests TestBootWiresDetachedHarnessTaskRuntimeAcrossScopes,
TestDetachedHarnessCompletionWakeEmitsSyntheticReentryEndToEnd,
TestDetachedHarnessCompletionSilentPolicyRecordsDropEndToEnd,
TestDetachedHarnessCompletionWakePreservesFIFOAcrossRuns,
TestBootRecoveryDetachedHarnessWakeUsesPersistedSyntheticEventForDedupe, and
TestBootRecoversDetachedHarnessRunThroughTaskRuntimeRules should be consolidated
into a single parent test (e.g., TestDetachedHarnessIntegration) that uses
table-driven subtests with t.Run("Should ...", func(t *testing.T) { ... })
entries describing each scenario, moving the common setup (integrationHomePaths,
testConfig, fakeSessionManager, bootDetachedHarnessIntegrationDaemon, workspace
resolution, seeding) into shared code reused by each subtest and keeping
scenario-specific inputs/assertions in the table; additionally, update
ensureDetachedHarnessWorkspaceIndex to wrap returned errors with
fmt.Errorf("...: %w", err) (use ensureDetachedHarnessWorkspaceIndex identifier
to locate the function) for the error paths currently returned raw so they
follow Go error-wrapping conventions.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The repository rules for Go tests require `t.Run("Should...")` subtests, and these six detached-harness integration cases were standalone top-level tests with repeated boot/setup scaffolding. I consolidated them under `TestDetachedHarnessIntegration` with `Should...` subtests while preserving each scenario's assertions. The overlapping helper error-wrapping note was implemented as part of issue `002`. Verified with `go test -tags integration ./internal/daemon -run TestDetachedHarnessIntegration -count=1` and `make verify`.
