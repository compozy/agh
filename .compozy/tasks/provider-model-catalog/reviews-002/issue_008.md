---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/api/testutil/model_catalog_parity_test.go
line: 168
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYapw,comment:PRRC_kwDOR5y4QM6-7HYF
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't hardcode `/tmp` in this test helper.**

This will fail on Windows and other non-Unix environments. `t.TempDir()` already gives you an isolated path plus automatic cleanup.

 

<details>
<summary>Suggested change</summary>

```diff
 func newShortParityHomePaths(t *testing.T) aghconfig.HomePaths {
 	t.Helper()
 
-	root, err := os.MkdirTemp("/tmp", "agh-model-parity-*")
-	if err != nil {
-		t.Fatalf("MkdirTemp() error = %v", err)
-	}
-	t.Cleanup(func() {
-		if err := os.RemoveAll(root); err != nil {
-			t.Errorf("RemoveAll(%q) error = %v", root, err)
-		}
-	})
+	root := t.TempDir()
 	homePaths, err := aghconfig.ResolveHomePathsFrom(root)
 	if err != nil {
 		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
 	}
 	return homePaths
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func newShortParityHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	root := t.TempDir()
	homePaths, err := aghconfig.ResolveHomePathsFrom(root)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	return homePaths
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/testutil/model_catalog_parity_test.go` around lines 152 - 168,
The helper newShortParityHomePaths hardcodes /tmp with os.MkdirTemp which breaks
on Windows; replace the os.MkdirTemp and manual cleanup with t.TempDir() (e.g.
root := t.TempDir()), remove the explicit t.Cleanup RemoveAll block, then call
aghconfig.ResolveHomePathsFrom(root) as before and keep t.Fatalf on error so
newShortParityHomePaths uses the portable temp directory provided by the test
framework.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/testutil/model_catalog_parity_test.go` still hardcodes `os.MkdirTemp("/tmp", ...)` in `newShortParityHomePaths`.
  - That is non-portable and unnecessary because `t.TempDir()` already provides isolation and cleanup.
  - Fix plan: replace the hardcoded `/tmp` allocation and manual cleanup with `t.TempDir()`.
  - Fixed by removing the `/tmp` dependency while keeping the temp root short enough for UDS sockets, then verified with focused package tests plus `make verify`.
