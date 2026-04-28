---
status: resolved
file: internal/extension/host_api_test.go
line: 3615
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57znpC,comment:PRRC_kwDOR5y4QM645il0
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use `t.Run("Should...")` subtests for the two newly added scenarios.**

These are distinct failure-path cases and should be isolated as named subtests per test guidelines.



<details>
<summary>💡 Suggested refactor</summary>

```diff
-	_, _, err = (&HostAPIHandler{}).taskManagerAndActor(testutil.Context(t))
-	assertErrorContains(t, err, "resolve task manager")
-	assertErrorContains(t, err, "task manager is not configured")
-
-	env := newHostAPITestEnv(t)
-	_, _, err = env.handler.taskManagerAndActor(testutil.Context(t))
-	assertErrorContains(t, err, "derive task actor context")
-	assertErrorContains(t, err, "extension name is not available")
+	t.Run("ShouldWrapTaskManagerResolutionError", func(t *testing.T) {
+		t.Parallel()
+		_, _, err := (&HostAPIHandler{}).taskManagerAndActor(testutil.Context(t))
+		assertErrorContains(t, err, "resolve task manager")
+		assertErrorContains(t, err, "task manager is not configured")
+	})
+
+	t.Run("ShouldWrapTaskActorContextErrorWhenExtensionNameMissing", func(t *testing.T) {
+		t.Parallel()
+		env := newHostAPITestEnv(t)
+		_, _, err := env.handler.taskManagerAndActor(testutil.Context(t))
+		assertErrorContains(t, err, "derive task actor context")
+		assertErrorContains(t, err, "extension name is not available")
+	})
+
+	env := newHostAPITestEnv(t)
```
</details>

As per coding guidelines, "**Use table-driven tests with subtests (`t.Run`) as default**" and "**MUST use `t.Run(\"Should...\")` pattern for ALL test cases**".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	t.Run("ShouldWrapTaskManagerResolutionError", func(t *testing.T) {
		t.Parallel()
		_, _, err := (&HostAPIHandler{}).taskManagerAndActor(testutil.Context(t))
		assertErrorContains(t, err, "resolve task manager")
		assertErrorContains(t, err, "task manager is not configured")
	})

	t.Run("ShouldWrapTaskActorContextErrorWhenExtensionNameMissing", func(t *testing.T) {
		t.Parallel()
		env := newHostAPITestEnv(t)
		_, _, err := env.handler.taskManagerAndActor(testutil.Context(t))
		assertErrorContains(t, err, "derive task actor context")
		assertErrorContains(t, err, "extension name is not available")
	})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 3608 - 3615, The two newly
added assertions around taskManagerAndActor should be split into named subtests
using t.Run to isolate failure paths; update the test block so the call to
(&HostAPIHandler{}).taskManagerAndActor(testutil.Context(t)) and its asserts are
wrapped in t.Run("Should return error when task manager is not configured",
func(t *testing.T) { ... }), and wrap the
env.handler.taskManagerAndActor(testutil.Context(t)) case in t.Run("Should
return error when extension name is not available", func(t *testing.T) { ... });
keep the same assertions (assertErrorContains) and contexts but place each
scenario inside its own t.Run closure to follow the test guidelines.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: two distinct failure-path checks live inline in a larger test body, which blurs the failing scenario and weakens the repo's preferred subtest structure for isolated cases.
- Fix plan: wrap each `taskManagerAndActor` failure path in its own named `t.Run("Should...")` subtest while keeping the assertions unchanged.
- Resolution: the two `taskManagerAndActor` failure paths now run as isolated named subtests, keeping the original assertions while making failures attributable to a single scenario.
- Verification: `go test ./internal/bundles ./internal/sandbox/daytona ./internal/extension ./internal/tools` and `make verify` passed on 2026-04-17.
