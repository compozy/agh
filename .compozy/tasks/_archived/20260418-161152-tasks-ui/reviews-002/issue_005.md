---
status: resolved
file: internal/daemon/automation_task_e2e_assertions_test.go
line: 126
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM576AUb,comment:PRRC_kwDOR5y4QM65ChGq
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use `t.Run("Should...")` for this new test case.**

Line 120-126 adds a standalone case without the required subtest pattern for Go tests in this repo.

<details>
<summary>Suggested update</summary>

```diff
 func TestFindTaskRunInDetailReturnsMissingForNilDetail(t *testing.T) {
 	t.Parallel()
 
-	if _, ok := findTaskRunInDetail(nil, "task-run-1"); ok {
-		t.Fatal("findTaskRunInDetail(nil) = present, want missing")
-	}
+	t.Run("Should return missing for nil detail", func(t *testing.T) {
+		t.Parallel()
+		if _, ok := findTaskRunInDetail(nil, "task-run-1"); ok {
+			t.Fatal("findTaskRunInDetail(nil) = present, want missing")
+		}
+	})
 }
```
</details>

  
As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/automation_task_e2e_assertions_test.go` around lines 120 -
126, The new test TestFindTaskRunInDetailReturnsMissingForNilDetail is a
standalone test but must follow the repo's subtest pattern; update it to call
t.Run with a descriptive "Should..." name (e.g., t.Run("Should return missing
for nil detail", func(t *testing.T) {...})) and move t.Parallel() inside that
subtest, keeping the assertion that findTaskRunInDetail(nil, "task-run-1")
returns missing; ensure the test function name
(TestFindTaskRunInDetailReturnsMissingForNilDetail) and the target function
findTaskRunInDetail remain unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `TestFindTaskRunInDetailReturnsMissingForNilDetail` is currently a standalone body without the required `t.Run("Should ...")` structure used across Go tests in this repo.
- Root cause analysis: The test was added as a direct assertion instead of following the required subtest pattern.
- Intended fix: Wrap the assertion in a `Should ...` subtest and move `t.Parallel()` to the subtest body.
- Resolution: Wrapped the nil-detail assertion in a `Should return missing for nil detail` subtest and kept the original helper behavior unchanged.
- Verification:
  - `go test ./internal/daemon -run 'TestFindTaskRunInDetailReturnsMissingForNilDetail|TestRequireCompletedSessionAutomationRun|TestRequireDelegatedTaskAutomationRun|TestFindTaskRunHelpers|TestClassifyAutomationRunLinkageRejectsMixedSurfaces'`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
