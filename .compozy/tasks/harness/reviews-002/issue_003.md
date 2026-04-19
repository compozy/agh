---
status: resolved
file: internal/daemon/daemon_test.go
line: 3649
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUA,comment:PRRC_kwDOR5y4QM65IlPB
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Align new test cases with required `t.Run("Should...")` structure.**

Most new cases in this block are standalone tests instead of subtests following the required `Should...` convention and table-driven default.



As per coding guidelines `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3358 - 3649, The new tests
(TestTaskRuntimeDetachedHarnessSubmissionAllowsProcessedReentryMetadata,
TestHarnessReentryBridgeOnTaskEventSchedulesRescanWhenQueueIsFull,
TestHarnessReentryBridgeRecoverOrdersEqualTimestampsByTerminalSequence,
TestHarnessReentryBridgeShutdownFinalizesHungSyntheticWake,
TestSectionSelectorFallbackStillFiltersProvidersAndDuplicates) must be converted
to follow the repository test convention by turning each standalone case into a
t.Run("Should...") subtest (or table-driven subtests) — wrap the existing test
logic inside a t.Run with a descriptive "Should ..." name, keep calls to
t.Parallel() inside the subtest if appropriate, and if multiple scenarios exist
convert them to a table-driven slice and iterate with t.Run for each entry;
update references to functions like newDetachedHarnessTaskRuntimeForTest,
submitDetachedHarnessWorkForTest, newHarnessReentryBridge, bridge.recover,
waitForDetachedHarnessReentryState, seedDetachedHarnessRecoveryRunForTest, and
SectionSelector.Select to remain unchanged while moving their calls into the
subtest bodies so behavior is preserved.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The referenced daemon tests were standalone cases even though this repository requires `t.Run("Should...")` subtests in `*_test.go`. I regrouped the affected cases under `TestDetachedHarnessDaemonScenarios` with descriptive `Should...` subtests and preserved their existing behavior and parallelism. Verified with `go test ./internal/daemon -count=1` and `make verify`.
