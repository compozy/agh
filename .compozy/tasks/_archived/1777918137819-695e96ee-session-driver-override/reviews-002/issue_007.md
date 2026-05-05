---
status: resolved
file: internal/session/provider_lifecycle_integration_test.go
line: 14
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPI,comment:PRRC_kwDOR5y4QM6628Dt
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap each test case body in a `t.Run("Should...")` subtest.**

Both new integration tests are top-level bodies without the required `Should...` subtest wrapper.

<details>
<summary>Suggested adjustment</summary>

```diff
func TestManagerIntegrationProviderPersistsAcrossCreateStatusListAndResume(t *testing.T) {
-	h := newHarness(t)
-	// existing body...
+	t.Run("Should persist provider across create/status/list/resume", func(t *testing.T) {
+		h := newHarness(t)
+		// existing body...
+	})
}

func TestManagerIntegrationLegacyProviderRepairPersistsAndResumeStaysDeterministic(t *testing.T) {
-	h := newHarness(t)
-	// existing body...
+	t.Run("Should repair missing provider and keep resume deterministic", func(t *testing.T) {
+		h := newHarness(t)
+		// existing body...
+	})
}
```
</details>


As per coding guidelines `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".


Also applies to: 76-76

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/provider_lifecycle_integration_test.go` at line 14, The test
function TestManagerIntegrationProviderPersistsAcrossCreateStatusListAndResume
is a top-level test body and must be wrapped in a subtest using t.Run with a
descriptive "Should..." name; update the function to call t.Run("Should persist
provider across create/status/list/resume", func(t *testing.T) { ... }) and move
the existing test body into that closure (apply the same t.Run wrapping for the
other test at the indicated location). Ensure you preserve the original
assertions and setup but place them inside the t.Run anonymous func receiving t
*testing.T.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: Both new provider lifecycle integration scenarios are top-level bodies instead of `Should...` subtests. I will wrap each scenario in a descriptive `t.Run` block and keep the existing setup and assertions intact.
