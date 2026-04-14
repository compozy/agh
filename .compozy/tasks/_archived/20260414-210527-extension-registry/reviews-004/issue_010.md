---
status: resolved
file: internal/registry/extract_test.go
line: 57
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM567Hj2,comment:PRRC_kwDOR5y4QM63sxX1
---

# Issue 010: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use `t.Run("Should...")` for each test case, including current single-scenario tests.**

Several top-level tests are still single-case functions instead of `t.Run("Should...")` cases, which breaks the repo’s test-case convention.

<details>
<summary>Suggested pattern (example)</summary>

```diff
 func TestPathWithinRoot(t *testing.T) {
 	t.Parallel()
-
-	root := t.TempDir()
-
-	target, err := PathWithinRoot(root, filepath.Join("review", "SKILL.md"))
-	if err != nil {
-		t.Fatalf("PathWithinRoot(valid) error = %v", err)
-	}
-	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
-		t.Fatalf("PathWithinRoot(valid) = %q, want path under %q", target, root)
-	}
-
-	if _, err := PathWithinRoot(root, filepath.Join("..", "escape")); !errors.Is(err, ErrPathOutsideRoot) {
-		t.Fatalf("PathWithinRoot(escape) error = %v, want %v", err, ErrPathOutsideRoot)
-	}
-	if _, err := PathWithinRoot("   ", "review/SKILL.md"); !errors.Is(err, ErrPathRootRequired) {
-		t.Fatalf("PathWithinRoot(blank root) error = %v, want %v", err, ErrPathRootRequired)
-	}
+	t.Run("ShouldResolvePathInsideRootAndRejectEscapes", func(t *testing.T) {
+		root := t.TempDir()
+
+		target, err := PathWithinRoot(root, filepath.Join("review", "SKILL.md"))
+		if err != nil {
+			t.Fatalf("PathWithinRoot(valid) error = %v", err)
+		}
+		if !strings.HasPrefix(target, root+string(filepath.Separator)) {
+			t.Fatalf("PathWithinRoot(valid) = %q, want path under %q", target, root)
+		}
+
+		if _, err := PathWithinRoot(root, filepath.Join("..", "escape")); !errors.Is(err, ErrPathOutsideRoot) {
+			t.Fatalf("PathWithinRoot(escape) error = %v, want %v", err, ErrPathOutsideRoot)
+		}
+		if _, err := PathWithinRoot("   ", "review/SKILL.md"); !errors.Is(err, ErrPathRootRequired) {
+			t.Fatalf("PathWithinRoot(blank root) error = %v, want %v", err, ErrPathRootRequired)
+		}
+	})
 }
```
</details>



As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases` and `Use table-driven tests with subtests (t.Run) as default in Go tests`.


Also applies to: 180-199, 235-274, 369-439

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract_test.go` around lines 24 - 57, The test function
TestExtractArchive_ValidArchiveProducesDirectoryStructure should be refactored
into a subtest using t.Run with a "Should..." description and use table-driven
subtests for each check; specifically, wrap the overall scenario in
t.Run("Should extract valid tar.gz into expected directory structure", func(t
*testing.T) { ... }) and convert the loop over checks into subtests like
t.Run(fmt.Sprintf("Should have %s", check.path), func(t *testing.T){ ... }),
keeping existing setup (mustTarGz, ExtractArchive) and assertions (os.ReadFile
and content compare) intact; apply the same pattern to other top-level tests
referenced (lines ~180-199, 235-274, 369-439) so all cases follow the
t.Run("Should...") convention.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: several top-level tests in `extract_test.go` still bundle a single scenario directly in the parent test function instead of using the repo’s `t.Run("Should...")` convention.
- Evidence: [`internal/registry/extract_test.go`](internal/registry/extract_test.go) includes multiple single-scenario top-level tests at the referenced ranges.
- Fix plan: wrap those scenarios in named `Should...` subtests while preserving existing assertions and setup.
- Resolution: Refactored the referenced extractor tests into `Should...` subtests without changing behavior. Verified with package tests and `make verify`.
