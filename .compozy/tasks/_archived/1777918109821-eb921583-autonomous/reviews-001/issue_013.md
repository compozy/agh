---
status: resolved
file: internal/api/core/tasks_surface_integration_test.go
line: 347
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsM,comment:PRRC_kwDOR5y4QM67YHCm
---

# Issue 013: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap each mutation case in a named subtest.**

The inline table loop makes route failures harder to isolate and doesn't follow the repo's required `t.Run("Should...")` pattern. Converting these cases into named subtests will make the new `/start` and `approve=201` expectations much easier to diagnose.



As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default pattern` and `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_integration_test.go` around lines 331 - 347,
The table-driven loop over mutation routes should be converted into named
subtests using t.Run so failures are isolated and follow the "Should..."
pattern; for each case in the slice (paths like "/tasks/task-1/publish",
"/tasks/task-1/start", "/tasks/task-1/approve", etc.) replace the anonymous
iteration with t.Run("Should <action or path>", func(t *testing.T){ resp :=
performRequest(t, fixture.Engine, http.MethodPost, tc.path, nil); if resp.Code
!= tc.want { t.Fatalf("%s status = %d, want %d; body=%s", tc.path, resp.Code,
tc.want, resp.Body.String()) } }) ensuring each test name describes the
expectation (e.g., "Should return 201 for /tasks/task-1/start") and keep the
same performRequest call and assertion logic inside each subtest.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestExpandedTaskMutationHandlersDelegateIntegration` loops over mutation endpoints without subtests, so route-specific failures are reported only from the shared loop body. Fix by wrapping each route case in a named `t.Run("Should ...")` subtest while preserving the final aggregate call/origin assertions.
