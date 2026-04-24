---
status: resolved
file: internal/config/bootstrap_test.go
line: 192
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58o_sg,comment:PRRC_kwDOR5y4QM66AfBM
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use required `t.Run("Should...")` subtest pattern for these new scenarios**

These new cases are added as standalone tests; project test guidelines require table-driven subtests with `t.Run("Should...")` for test cases.

<details>
<summary>♻️ Suggested structure</summary>

```diff
-func TestSaveBootstrapConfigFirstRunKeepsNetworkEnabledByDefault(t *testing.T) {
-	t.Parallel()
-	...
-}
-
-func TestSaveBootstrapConfigPreservesExplicitNetworkDisable(t *testing.T) {
-	t.Parallel()
-	...
-}
+func TestSaveBootstrapConfigNetworkBehavior(t *testing.T) {
+	t.Parallel()
+
+	tests := []struct {
+		name string
+		seed string
+	}{
+		{
+			name: "ShouldKeepNetworkEnabledByDefaultOnFirstRun",
+			seed: "",
+		},
+		{
+			name: "ShouldPreserveExplicitNetworkDisable",
+			seed: "[network]\nenabled = false\ndefault_channel = \"legacy\"\n",
+		},
+	}
+
+	for _, tc := range tests {
+		tc := tc
+		t.Run(tc.name, func(t *testing.T) {
+			t.Parallel()
+			// arrange/act/assert...
+		})
+	}
+}
```
</details>


As per coding guidelines, `**/*_test.go`: “MUST use t.Run("Should...") pattern for ALL test cases” and “Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests”.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/bootstrap_test.go` around lines 110 - 192, The two standalone
tests (TestSaveBootstrapConfigFirstRunKeepsNetworkEnabledByDefault and
TestSaveBootstrapConfigPreservesExplicitNetworkDisable) violate the project
guideline requiring table-driven subtests; refactor them into a single
table-driven test that iterates over cases and calls t.Run("Should ...", func(t
*testing.T){ ... }) for each scenario, referencing the same helpers
(ResolveHomePathsFrom, SaveBootstrapConfig, LoadGlobalConfig, writeFile) inside
each subtest; ensure each subtest calls t.Parallel() as appropriate, uses
descriptive "Should..." names, and asserts the same expectations currently in
the two functions (first-run default network enabled and preserving explicit
network disabled + default_channel) so behavior and file contents checks remain
unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The current file still has two standalone tests, `TestSaveBootstrapConfigFirstRunKeepsNetworkEnabledByDefault` and `TestSaveBootstrapConfigPreservesExplicitNetworkDisable`, rather than a shared table-driven subtest.
  - Root cause: the new network-default scenarios were added incrementally as top-level tests instead of following the repo's default `t.Run("Should...")` pattern used elsewhere in `internal/config`.
  - Fix approach: collapse the two scenarios into one table-driven test with `Should...` subtest names, keep the existing assertions intact, and preserve per-case parallel execution.
  - Implemented: the standalone tests were merged into `TestSaveBootstrapConfigNetworkBehavior` with parallel `Should...` subtests and the original expectations preserved.
  - Verified: focused `go test ./internal/api/httpapi ./internal/api/udsapi ./internal/api/core ./internal/config ./internal/daemon ./internal/testutil/e2e` passed, then `make verify` passed.
