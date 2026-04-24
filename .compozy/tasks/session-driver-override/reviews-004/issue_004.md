---
status: resolved
file: internal/daemon/task_runtime_test.go
line: 518
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11X,comment:PRRC_kwDOR5y4QM663fBl
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, locate and check the structure of the test file
fd task_runtime_test.go
```

Repository: compozy/agh

Length of output: 92

---

🏁 Script executed:

```shell
# Check the overall line count and specific sections mentioned in the review
wc -l internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 97

---

🏁 Script executed:

```shell
# Examine the test functions at the line ranges mentioned
sed -n '515,530p' internal/daemon/task_runtime_test.go
sed -n '645,655p' internal/daemon/task_runtime_test.go
sed -n '688,700p' internal/daemon/task_runtime_test.go
sed -n '745,760p' internal/daemon/task_runtime_test.go
sed -n '835,845p' internal/daemon/task_runtime_test.go
sed -n '920,930p' internal/daemon/task_runtime_test.go
sed -n '1330,1345p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 2650

---

🏁 Script executed:

```shell
# Get more complete view of TestBootTasksSkipsMissingPrerequisites
sed -n '648,720p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 1851

---

🏁 Script executed:

```shell
# Get TestBootTasksBuildsRuntimeWhenDependenciesAreAvailable
sed -n '690,800p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 3535

---

🏁 Script executed:

```shell
# Get TestBootTasksRecoversPendingRunsOnStartup
sed -n '747,900p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 5040

---

🏁 Script executed:

```shell
# Get TestBootTasksRequiresHarnessResolver through TestRecoverTaskRunsOnBootTracksAllRecoveryOutcomes
sed -n '838,1100p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 9399

---

🏁 Script executed:

```shell
# Get TestTaskRecoveryLivenessHelpers
sed -n '920,1050p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 4584

---

🏁 Script executed:

```shell
# Get TestRecoverTaskRunsOnBootTracksAllRecoveryOutcomes
sed -n '1331,1500p' internal/daemon/task_runtime_test.go
```

Repository: compozy/agh

Length of output: 6458

---



</details>

**Wrap plain test bodies in `t.Run("Should...")` subtests.**

These test functions test logic directly in their body rather than using the required `t.Run()` pattern:
- TestBootTasksBuildsRuntimeWhenDependenciesAreAvailable (line 690)
- TestBootTasksRecoversPendingRunsOnStartup (line 747)
- TestBootTasksRequiresHarnessResolver (line 838)
- TestTaskRecoveryLivenessHelpers (line 920)
- TestRecoverTaskRunsOnBootTracksAllRecoveryOutcomes (line 1331)

Repo guideline: `**/*_test.go`: `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/task_runtime_test.go` around lines 517 - 518, The listed
top-level tests (TestBootTasksBuildsRuntimeWhenDependenciesAreAvailable,
TestBootTasksRecoversPendingRunsOnStartup, TestBootTasksRequiresHarnessResolver,
TestTaskRecoveryLivenessHelpers,
TestRecoverTaskRunsOnBootTracksAllRecoveryOutcomes) must be converted to use the
t.Run("Should...") subtest pattern: for each of these functions, replace the
direct test body with a single t.Run call whose name describes the expectation
(e.g., "Should build runtime when dependencies are available"), move the current
body into the t.Run closure, and call t.Parallel() inside the subtest (or at the
top-level before t.Run if intended) so the test still runs in parallel; ensure
any test-local variables are captured correctly in the closure.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
- The cited tests already exercise distinct runtime/bootstrap branches directly, and the requested change would only wrap each body in a single `t.Run(...)` without changing setup, assertions, or parallelism.
- No missing coverage or defect was identified in these tests, so this is a style-only refactor and not a review blocker for the scoped batch.
