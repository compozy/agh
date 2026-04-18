---
status: resolved
file: magefile.go
line: 534
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575RS5,comment:PRRC_kwDOR5y4QM65BgwO
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap command failures with context in `runRaceEnabledGoCommand`.**

Return the underlying error with command context so failures are diagnosable.



<details>
<summary>Proposed fix</summary>

```diff
 func runRaceEnabledGoCommand(env map[string]string, args ...string) error {
-	return runCommandInDirWithEnv(".", withRaceEnabledEnv(env), "go", args...)
+	if err := runCommandInDirWithEnv(".", withRaceEnabledEnv(env), "go", args...); err != nil {
+		return fmt.Errorf("run race-enabled go command: %w", err)
+	}
+	return nil
 }
```
</details>

As per coding guidelines, use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func runRaceEnabledGoCommand(env map[string]string, args ...string) error {
	if err := runCommandInDirWithEnv(".", withRaceEnabledEnv(env), "go", args...); err != nil {
		return fmt.Errorf("run race-enabled go command: %w", err)
	}
	return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@magefile.go` around lines 532 - 534, The function runRaceEnabledGoCommand
currently returns the raw error from runCommandInDirWithEnv without context;
modify runRaceEnabledGoCommand to capture the error from runCommandInDirWithEnv
and, if non-nil, return a wrapped error using fmt.Errorf that includes a short
context string (e.g., "runRaceEnabledGoCommand: failed to run 'go' with args %v:
%w") and the original error; reference the call to runCommandInDirWithEnv(".",
withRaceEnabledEnv(env), "go", args...) and ensure fmt is imported if not
already.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `runRaceEnabledGoCommand` forwards the raw error from `runCommandInDirWithEnv`, so callers lose context about which race-enabled `go` invocation failed.
- Impact: slower diagnosis when mage-driven test/build commands fail.
- Fix plan: wrap the returned error with a short contextual message including the invoked args. This is the same underlying fix tracked in issue 005.
- Resolution: `runRaceEnabledGoCommand` now returns a wrapped error that includes the race-enabled `go` args. The same change resolves the overlapping requirement in issue 005.
- Verification: `go test -tags mage .`; `make verify`
