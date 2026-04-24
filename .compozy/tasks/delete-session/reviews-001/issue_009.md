---
status: resolved
file: internal/session/manager_delete_test.go
line: 70
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6Z,comment:PRRC_kwDOR5y4QM66Aomu
---

# Issue 009: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Restructure to `t.Run("Should...")` table-driven subtests.**

The assertions are useful, but this file currently skips the required subtest pattern.


As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use `t.Run(\"Should...\")` pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_delete_test.go` around lines 11 - 70, Convert the
two top-level tests into table-driven subtests using t.Run with names that begin
"Should...": create a single TestDeleteBehavior (or similar) that calls
t.Parallel() once and iterates over a cases slice where each case has a name
("Should remove stopped session from history", "Should stop active session
before removing artifacts") and a test func. Move the existing bodies of
TestDeleteRemovesStoppedSessionFromHistory and
TestDeleteStopsActiveSessionBeforeRemovingArtifacts into the case handlers,
preserving calls to h.manager.Stop, h.manager.Delete, h.manager.Status,
h.manager.ListAll, h.manager.Get, session.SessionDir(), and assertions on
h.driver.stopCalls; invoke each case with t.Run(case.name, func(t *testing.T) {
t.Parallel(); ... }) so each subtest runs in parallel and keeps original
assertions and setup (newHarness, createSession) unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `internal/session/manager_delete_test.go` still uses two separate top-level tests instead of the repository's required table-driven `Should...` subtest pattern. I will consolidate these delete scenarios into one parallel table-driven test and keep the existing assertions intact.
