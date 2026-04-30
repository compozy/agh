---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/config_test.go
line: 410
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKJ,comment:PRRC_kwDOR5y4QM680KIp
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restructure this test into `t.Run("Should ...")` subtests for each scenario.**

This block mixes many independent cases in one function, and Line 366 uses a path string as subtest name instead of the required `Should ...` format. Please split show/list/path/parse cases into explicit `Should ...` subtests (each with `t.Parallel()` where safe) so failures are isolated and guideline-compliant.



<details>
<summary>Suggested refactor</summary>

```diff
 func TestConfigRenderingAndMutationHelpers(t *testing.T) {
 	t.Parallel()

-	entries := []configEntry{
+	entries := []configEntry{
 		// ...
 	}
-	showBundle := configShowBundle(...)
-	showHuman, err := showBundle.human()
-	// ...
+	t.Run("Should render config show output", func(t *testing.T) {
+		t.Parallel()
+		showBundle := configShowBundle(...)
+		showHuman, err := showBundle.human()
+		if err != nil {
+			t.Fatalf("configShowBundle.human() error = %v", err)
+		}
+		// assertions...
+	})
+
+	t.Run("Should render config list output", func(t *testing.T) {
+		t.Parallel()
+		// ...
+	})
+
+	t.Run("Should render config path output", func(t *testing.T) {
+		t.Parallel()
+		// ...
+	})

 	for _, tc := range testCases {
-		t.Run(tc.path, func(t *testing.T) {
+		t.Run("Should classify mutation path "+tc.path, func(t *testing.T) {
 			t.Parallel()
 			// ...
 		})
 	}
+
+	t.Run("Should parse string slice from JSON", func(t *testing.T) { t.Parallel(); /* ... */ })
+	t.Run("Should parse string slice from CSV", func(t *testing.T) { t.Parallel(); /* ... */ })
+	t.Run("Should return empty slice for blank input", func(t *testing.T) { t.Parallel(); /* ... */ })
+	t.Run("Should fail for invalid JSON element types", func(t *testing.T) { t.Parallel(); /* ... */ })
 }
```
</details>

As per coding guidelines, "`**/*_test.go`: Use `t.Run(\"Should ...\")` subtests with `t.Parallel` as default" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/config_test.go` around lines 279 - 410, The test
TestConfigRenderingAndMutationHelpers bundles many independent checks; split it
into separate t.Run("Should ...") subtests (e.g., "Should render show bundle
human", "Should render show bundle toon", "Should list bundle human", "Should
list bundle toon", "Should render path bundle human", "Should render path bundle
toon", "Should classify mutation paths", "Should parse string slice values") and
move each logical block that calls configShowBundle.human()/toon(),
configListBundle.human()/toon(), configPathBundle.human()/toon(), the
table-driven calls to configMutationPath, and parseStringSliceValue into its own
subtest; ensure each subtest calls t.Parallel() where safe, and change the
table-driven t.Run names for the mutation path cases to the required "Should
..." pattern (not the raw path string) while keeping the existing assertions and
use of symbols unchanged so failures are isolated and guidelines-compliant.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestConfigRenderingAndMutationHelpers` mixes rendering, mutation-path classification, and string-slice parsing in one top-level body, and its table subtests use raw path strings instead of `Should ...` names. Split the logical blocks into named parallel subtests and fix the table case names.
