---
status: resolved
file: internal/testutil/acpmock/registration.go
line: 167
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y10y,comment:PRRC_kwDOR5y4QM644c85
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Fail fast when diagnostics default root is unavailable.**

If `opts.DiagnosticsPath` is empty and `homePaths.LogsDir` is blank, Line 163 falls back to a relative `acpmock/` directory. That can leak artifacts into the process working directory.



<details>
<summary>🩹 Proposed fix</summary>

```diff
 func resolveDiagnosticsPath(homePaths aghconfig.HomePaths, name string, override string) (string, error) {
@@
+	if strings.TrimSpace(homePaths.LogsDir) == "" {
+		return "", errors.New("acpmock: home paths logs directory is required when diagnostics path override is not set")
+	}
 	dir := filepath.Join(homePaths.LogsDir, "acpmock")
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if strings.TrimSpace(homePaths.LogsDir) == "" {
		return "", errors.New("acpmock: home paths logs directory is required when diagnostics path override is not set")
	}
	dir := filepath.Join(homePaths.LogsDir, "acpmock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("acpmock: create diagnostics directory %q: %w", dir, err)
	}
	return filepath.Join(dir, strings.TrimSpace(name)+".jsonl"), nil
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/registration.go` around lines 163 - 167, The
current code silently falls back to creating a relative "acpmock" directory when
opts.DiagnosticsPath is empty and homePaths.LogsDir is blank; change the logic
in the diagnostics path resolver (the code that uses opts.DiagnosticsPath and
homePaths.LogsDir and builds dir with filepath.Join) to detect an empty
homePaths.LogsDir when opts.DiagnosticsPath is not provided and return an
explicit error instead of creating a relative directory; specifically, before
calling os.MkdirAll on dir, add a guard that if opts.DiagnosticsPath == "" &&
strings.TrimSpace(homePaths.LogsDir) == "" then return a clear error indicating
diagnostics root is unavailable so callers fail fast.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: when no diagnostics override is supplied and `homePaths.LogsDir` is blank, `resolveDiagnosticsPath()` falls back to `filepath.Join("", "acpmock")`, which creates a relative directory under the process working directory.
- Fix plan: require a non-empty logs root when no explicit diagnostics override is provided and return a contextual error otherwise.
- Test impact: requires a focused assertion in `internal/testutil/acpmock/fixture_test.go`.
- Resolution: implemented. `resolveDiagnosticsPath()` now fails fast when no override is supplied and `homePaths.LogsDir` is blank, and the acpmock tests cover that guard.
- Verification: `go test ./internal/testutil/acpmock`, `make verify`.
