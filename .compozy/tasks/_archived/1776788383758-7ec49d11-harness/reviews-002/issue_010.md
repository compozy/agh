---
status: resolved
file: internal/daemon/task_runtime_test.go
line: 1746
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUJ,comment:PRRC_kwDOR5y4QM65IlPM
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restructure new tests to required `t.Run("Should...")` pattern.**

Most newly added cases here are standalone tests instead of the required subtest naming/structure convention.



As per coding guidelines `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/task_runtime_test.go` around lines 456 - 1746, Several newly
added tests are written as standalone cases but must follow the project testing
convention to be table-driven subtests using t.Run("Should..."); refactor each
affected test function (e.g.,
TestHarnessReentryBridgeEmitsSyntheticWakeAndObservability,
TestHarnessReentryBridgeSilentPolicyRecordsDropSummary,
TestHarnessReentryBridgeMissingAndStoppedTargetsDropWithoutWake,
TestHarnessReentryBridgeDuplicateTerminalNotificationsStayIdempotent,
TestHarnessReentryBridgePreservesSyntheticWakeFIFO,
TestHarnessReentryBridgeHelperCoverage,
TestHarnessReentryBridgeDropsWhenSyntheticDispatchFails,
TestHarnessReentryBridgeDropsWhenSyntheticPromptChannelHasNoEvent,
TestHarnessReentryBridgeDropsWhenSyntheticPromptReturnsErrorEvent,
TestHarnessReentryBridgeDispatchWakeUsesRecordedSyntheticEvent) to wrap each
logical case in t.Run("Should ...", func(t *testing.T) { t.Parallel(); ... }),
convert any inline test cases into table-driven subtests where appropriate,
preserve existing helper calls (submitDetachedHarnessWorkForTest,
completeDetachedHarnessRunForTest, waitForDetachedHarnessReentryState,
sessions.syntheticPromptHook, etc.), and ensure all new subtest names start with
"Should" and include t.Parallel() to match the required pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The listed reentry bridge tests were mostly standalone top-level cases even though this repository requires `t.Run("Should...")` subtests. I regrouped them under `TestHarnessReentryBridgeScenarios` with `Should...` subtests, preserving the existing helper usage and scenario coverage while aligning the file with the test-structure convention. Verified with `go test ./internal/daemon -count=1` and `make verify`.
