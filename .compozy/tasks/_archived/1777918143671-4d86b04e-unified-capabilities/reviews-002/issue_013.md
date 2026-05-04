---
status: resolved
file: internal/e2elane/command_wiring_test.go
line: 98
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyx-,comment:PRRC_kwDOR5y4QM654Non
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use required `t.Run("Should...")` subtests in this test case.**

This assertion loop should be table-driven with explicit subtests; current structure violates the repo’s required Go test pattern and gives nondeterministic failure ordering due to map iteration.

<details>
<summary>Suggested refactor</summary>

```diff
 func TestRootPackageScriptsExposeSharedCodegenEntryPoints(t *testing.T) {
 	t.Parallel()

 	repoRoot := repoRoot(t)
 	pkg := readPackageJSON(t, filepath.Join(repoRoot, "package.json"))

-	want := map[string]string{
-		"codegen":       "make codegen",
-		"codegen-check": "make codegen-check",
-	}
-
-	for script, command := range want {
-		if got := pkg.Scripts[script]; got != command {
-			t.Fatalf("package.json script %q = %q, want %q", script, got, command)
-		}
-	}
+	tests := []struct {
+		name    string
+		script  string
+		command string
+	}{
+		{name: "Should expose codegen script", script: "codegen", command: "make codegen"},
+		{name: "Should expose codegen-check script", script: "codegen-check", command: "make codegen-check"},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+			if got := pkg.Scripts[tt.script]; got != tt.command {
+				t.Fatalf("package.json script %q = %q, want %q", tt.script, got, tt.command)
+			}
+		})
+	}
 }
```
</details>


As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/e2elane/command_wiring_test.go` around lines 82 - 98, The test
TestRootPackageScriptsExposeSharedCodegenEntryPoints currently iterates a map
directly and must be converted to a table-driven test with explicit
t.Run("Should ...") subtests; create a slice (or sorted keys) from the want map
and for each entry call t.Run with a descriptive "Should expose <script> as
<command>" name and perform the assertion inside the subtest so failures are
deterministic and adhere to the repo pattern; keep using repoRoot(t),
readPackageJSON(t, ...), and compare pkg.Scripts[script] to the expected command
within each t.Run.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: this test iterates a map directly, which gives nondeterministic assertion ordering and bypasses the repo's required `Should...` subtest structure.
- Fix plan: convert the expectations to an explicit test-case slice and assert each script in its own parallel `Should...` subtest.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
