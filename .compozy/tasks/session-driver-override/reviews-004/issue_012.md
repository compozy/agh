---
status: resolved
file: internal/session/provider_lifecycle_test.go
line: 229
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11k,comment:PRRC_kwDOR5y4QM663fB2
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/session/provider_lifecycle_test.go | head -250
```

Repository: compozy/agh

Length of output: 9397

---

🏁 Script executed:

```shell
find internal -name "*_test.go" -type f | head -10
```

Repository: compozy/agh

Length of output: 412

---

🏁 Script executed:

```shell
rg -l "t\.Run\(" internal --type go | head -5
```

Repository: compozy/agh

Length of output: 224

---

🏁 Script executed:

```shell
wc -l internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 103

---

🏁 Script executed:

```shell
grep -c "^func Test" internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 57

---

🏁 Script executed:

```shell
grep "^func Test" internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 421

---

🏁 Script executed:

```shell
ls -la internal/session/*_test.go
```

Repository: compozy/agh

Length of output: 1579

---

🏁 Script executed:

```shell
head -50 internal/session/manager_test.go | grep -E "^func Test|t\.Run"
```

Repository: compozy/agh

Length of output: 125

---

🏁 Script executed:

```shell
head -100 internal/session/manager_test.go | tail -50
```

Repository: compozy/agh

Length of output: 2152

---

🏁 Script executed:

```shell
grep -A 10 "t\.Run(" internal/session/manager_test.go | head -40
```

Repository: compozy/agh

Length of output: 1187

---

🏁 Script executed:

```shell
grep "t\.Run(" internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
head -150 internal/session/manager_test.go | grep -E "func Test|t\.Run"
```

Repository: compozy/agh

Length of output: 269

---

🏁 Script executed:

```shell
cat -n internal/session/manager_test.go | grep -B 2 "t\.Run.*Should" | head -20
```

Repository: compozy/agh

Length of output: 685

---

🏁 Script executed:

```shell
grep -c "t\.Run(" internal/session/manager_test.go
```

Repository: compozy/agh

Length of output: 58

---

🏁 Script executed:

```shell
cat internal/session/manager_test.go | grep -E "^func Test" | wc -l
```

Repository: compozy/agh

Length of output: 58

---

🏁 Script executed:

```shell
grep -E "^func Test" internal/session/manager_test.go | head -10
```

Repository: compozy/agh

Length of output: 780

---



</details>

**Refactor all test cases to use `t.Run("Should...")` table-driven subtests.**

The five test functions in this file use standalone function pattern instead of the required `t.Run("Should...")` pattern with descriptive names. Repo testing standards require table-driven tests with subtests for all test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/provider_lifecycle_test.go` around lines 15 - 229, Refactor
the five standalone tests
(TestCreateWithProviderOverridePropagatesToSessionRuntime,
TestCreateWithInvalidProviderFailsBeforePersistenceAndLogs,
TestStatusRepairsLegacyProviderAndLogs,
TestStatusFailsWhenLegacyProviderRepairCannotResolveAgent,
TestResumeFailsWhenPersistedProviderUnavailable) into a table-driven test that
uses t.Run("Should...") subtests for each scenario; for each case include a
descriptive Name, setup closure reusing helpers like newHarness, createSession,
readMeta, store.WriteSessionMeta, and assertions from the original functions,
invoke t.Parallel() inside each subtest (not at top-level), preserve cleanup
(t.Cleanup/Stop), and ensure log capture setup (newCaptureLogHandler/WithLogger)
and driver/notifier checks are executed per-case so behavior remains identical.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
## Triage

- Decision: `invalid`
- Notes:
- The five tests in `provider_lifecycle_test.go` cover distinct create/status/resume flows with materially different setup, log expectations, and metadata mutations.
- Converting them into a single table-driven function would be a stylistic consolidation only; it would not fix a defect or improve regression sensitivity for this batch.
