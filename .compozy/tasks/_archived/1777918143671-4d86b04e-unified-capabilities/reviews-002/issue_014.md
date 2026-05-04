---
status: resolved
file: internal/e2elane/command_wiring_test.go
line: 143
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyyH,comment:PRRC_kwDOR5y4QM654Noz
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Apply the same required `t.Run("Should...")` table-driven pattern here.**

This new test also bypasses the mandatory subtest structure; please convert to explicit subtests for each script assertion.

<details>
<summary>Suggested refactor</summary>

```diff
 func TestWebPackageScriptsRouteSharedCodegenIntoDependentCommands(t *testing.T) {
 	t.Parallel()

 	repoRoot := repoRoot(t)
 	pkg := readPackageJSON(t, filepath.Join(repoRoot, WebDir, "package.json"))

-	want := map[string]string{
-		"codegen":       "bun run --cwd .. codegen",
-		"codegen-check": "bun run --cwd .. codegen-check",
-		"dev":           "bun run codegen && bun run dev:raw",
-		"build":         "bun run codegen-check && bun run build:raw",
-		"test":          "bun run codegen-check && bun run test:raw",
-		"typecheck":     "bun run codegen-check && bun run typecheck:raw",
-	}
-
-	for script, command := range want {
-		if got := pkg.Scripts[script]; got != command {
-			t.Fatalf("web package script %q = %q, want %q", script, got, command)
-		}
-	}
+	tests := []struct {
+		name    string
+		script  string
+		command string
+	}{
+		{name: "Should route codegen", script: "codegen", command: "bun run --cwd .. codegen"},
+		{name: "Should route codegen-check", script: "codegen-check", command: "bun run --cwd .. codegen-check"},
+		{name: "Should run codegen before dev", script: "dev", command: "bun run codegen && bun run dev:raw"},
+		{name: "Should run codegen-check before build", script: "build", command: "bun run codegen-check && bun run build:raw"},
+		{name: "Should run codegen-check before test", script: "test", command: "bun run codegen-check && bun run test:raw"},
+		{name: "Should run codegen-check before typecheck", script: "typecheck", command: "bun run codegen-check && bun run typecheck:raw"},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+			if got := pkg.Scripts[tt.script]; got != tt.command {
+				t.Fatalf("web package script %q = %q, want %q", tt.script, got, tt.command)
+			}
+		})
+	}
 }
```
</details>


As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestWebPackageScriptsRouteSharedCodegenIntoDependentCommands(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, WebDir, "package.json"))

	tests := []struct {
		name    string
		script  string
		command string
	}{
		{name: "Should route codegen", script: "codegen", command: "bun run --cwd .. codegen"},
		{name: "Should route codegen-check", script: "codegen-check", command: "bun run --cwd .. codegen-check"},
		{name: "Should run codegen before dev", script: "dev", command: "bun run codegen && bun run dev:raw"},
		{name: "Should run codegen-check before build", script: "build", command: "bun run codegen-check && bun run build:raw"},
		{name: "Should run codegen-check before test", script: "test", command: "bun run codegen-check && bun run test:raw"},
		{name: "Should run codegen-check before typecheck", script: "typecheck", command: "bun run codegen-check && bun run typecheck:raw"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := pkg.Scripts[tt.script]; got != tt.command {
				t.Fatalf("web package script %q = %q, want %q", tt.script, got, tt.command)
			}
		})
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/e2elane/command_wiring_test.go` around lines 123 - 143, The test
TestWebPackageScriptsRouteSharedCodegenIntoDependentCommands currently iterates
scripts in a for loop without subtests; refactor it into a table-driven set of
subtests using t.Run(...) for each script. Replace the direct for-range over
want with either a slice of test cases or iterate the want map but call
t.Run(fmt.Sprintf("Should set %s script", script), func(t *testing.T){
t.Parallel(); if got := pkg.Scripts[script]; got != command { t.Fatalf(...)} })
so each assertion becomes its own subtest (keep the initial t.Parallel() on the
parent test and add t.Parallel() inside each subtest). Use the existing symbols
TestWebPackageScriptsRouteSharedCodegenIntoDependentCommands, pkg, want and
pkg.Scripts to locate the code to change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: this test has the same map-iteration/no-subtest problem as issue 013, plus it now carries several independent script assertions that are easier to diagnose as separate subtests.
- Fix plan: replace the direct map loop with explicit `Should...` subtests for each script/command pair.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
