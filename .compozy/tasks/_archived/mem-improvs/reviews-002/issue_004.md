---
status: resolved
file: internal/testutil/e2e/runtime_harness_lifecycle_test.go
line: 135
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575RS0,comment:PRRC_kwDOR5y4QM65BgwJ
---

# Issue 004: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap this test case in a `t.Run("Should...")` subtest.**

Line 113 adds a standalone case, but the test policy requires each case to use `t.Run("Should...")`. Please nest this scenario in a named subtest (and keep `t.Parallel()` inside it if independent).

<details>
<summary>♻️ Proposed refactor</summary>

```diff
 func TestCLIClientRunInDirResolvesRelativePathsAgainstBaseWorkdir(t *testing.T) {
     t.Parallel()
-
-	baseDir := t.TempDir()
-	targetDir := filepath.Join(baseDir, "nested")
-	if err := os.MkdirAll(targetDir, 0o755); err != nil {
-		t.Fatalf("os.MkdirAll(%q) error = %v", targetDir, err)
-	}
-
-	client := &CLIClient{
-		binaryPath: writeCLIScript(t, "#!/bin/sh\npwd\n"),
-		workdir:    baseDir,
-	}
-
-	stdout, stderr, err := client.RunInDir(context.Background(), "nested", "ignored")
-	if err != nil {
-		t.Fatalf("RunInDir() error = %v; stderr=%s", err, strings.TrimSpace(stderr))
-	}
-	if got, want := strings.TrimSpace(stdout), targetDir; got != want {
-		t.Fatalf("RunInDir() stdout = %q, want %q", got, want)
-	}
+	t.Run("Should resolve relative workdir against base workdir", func(t *testing.T) {
+		t.Parallel()
+
+		baseDir := t.TempDir()
+		targetDir := filepath.Join(baseDir, "nested")
+		if err := os.MkdirAll(targetDir, 0o755); err != nil {
+			t.Fatalf("os.MkdirAll(%q) error = %v", targetDir, err)
+		}
+
+		client := &CLIClient{
+			binaryPath: writeCLIScript(t, "#!/bin/sh\npwd\n"),
+			workdir:    baseDir,
+		}
+
+		stdout, stderr, err := client.RunInDir(context.Background(), "nested", "ignored")
+		if err != nil {
+			t.Fatalf("RunInDir() error = %v; stderr=%s", err, strings.TrimSpace(stderr))
+		}
+		if got, want := strings.TrimSpace(stdout), targetDir; got != want {
+			t.Fatalf("RunInDir() stdout = %q, want %q", got, want)
+		}
+	})
 }
```
</details>

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/runtime_harness_lifecycle_test.go` around lines 113 -
135, The test function
TestCLIClientRunInDirResolvesRelativePathsAgainstBaseWorkdir should be converted
to use a named subtest: wrap the existing body in t.Run("Should resolve relative
paths against base workdir", func(t *testing.T) { ... }), keep the t.Parallel()
call inside that subtest (not at top level), and leave all references to
CLIClient, writeCLIScript, RunInDir and the assertions unchanged so behavior
remains identical; this satisfies the requirement to use the t.Run("Should...")
pattern for each case.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestCLIClientRunInDirResolvesRelativePathsAgainstBaseWorkdir` is a standalone case and does not follow the required `t.Run("Should...")` test-case shape.
- Impact: style/policy drift only; runtime behavior is already correct.
- Fix plan: wrap the existing body in one named subtest and keep `t.Parallel()` inside that subtest.
- Resolution: wrapped the relative-workdir CLI test in a single `t.Run("Should ...")` case and kept `t.Parallel()` inside the subtest body.
- Verification: `go test ./internal/testutil/e2e`; `make verify`
