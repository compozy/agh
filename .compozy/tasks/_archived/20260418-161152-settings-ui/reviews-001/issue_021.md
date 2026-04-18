---
status: resolved
file: internal/daemon/restart_test.go
line: 1216
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kR8,comment:PRRC_kwDOR5y4QM65B60p
---

# Issue 021: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**This wrapper test is Unix-only today.**

The temporary `agh-helper.sh` depends on `/bin/sh` and executable-bit semantics, so this test will fail on Windows even though the restart code has platform-specific process launchers. Either gate this case to Unix or generate a platform-specific stub.


<details>
<summary>💡 One portable approach</summary>

```diff
-	scriptPath := filepath.Join(t.TempDir(), "agh-helper.sh")
-	script := "#!/bin/sh\nexit 0\n"
-	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
+	dir := t.TempDir()
+	scriptPath := filepath.Join(dir, "agh-helper.sh")
+	script := "#!/bin/sh\nexit 0\n"
+	mode := os.FileMode(0o755)
+	if runtime.GOOS == "windows" {
+		scriptPath = filepath.Join(dir, "agh-helper.cmd")
+		script = "@echo off\r\nexit /b 0\r\n"
+		mode = 0o600
+	}
+	if err := os.WriteFile(scriptPath, []byte(script), mode); err != nil {
 		t.Fatalf("os.WriteFile(script) error = %v", err)
 	}
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "agh-helper.sh")
	script := "#!/bin/sh\nexit 0\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(dir, "agh-helper.cmd")
		script = "@echo off\r\nexit /b 0\r\n"
		mode = 0o600
	}
	if err := os.WriteFile(scriptPath, []byte(script), mode); err != nil {
		t.Fatalf("os.WriteFile(script) error = %v", err)
	}

	err = RunRelaunchHelper(testutil.Context(t), RelaunchHelperConfig{
		HomePaths:      homePaths,
		OperationID:    operation.OperationID,
		Executable:     func() (string, error) { return scriptPath, nil },
		Environment:    []string{"PATH=" + os.Getenv("PATH")},
		PollInterval:   10 * time.Millisecond,
		ReleaseTimeout: 200 * time.Millisecond,
		ReadyTimeout:   200 * time.Millisecond,
	})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/restart_test.go` around lines 1202 - 1216, The test in
restart_test.go creates a Unix-only helper script and will fail on Windows;
update the test around the RunRelaunchHelper invocation (and the
scriptPath/script variables) to either skip this case on non-Unix platforms (use
runtime.GOOS or build tags) or create a platform-specific stub: for Unix keep
the /bin/sh executable script, and for Windows provide an .exe/PowerShell/batch
stub or return a fake executable path from the Executable func; ensure the
change is applied to the Test that calls RunRelaunchHelper so the test only uses
the shell script on Unix and uses the Windows-specific stub or skip logic
otherwise.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestRunRelaunchHelperReplacementFailurePersistsFailedOperation` currently writes a Unix shell script with a `#!/bin/sh` shebang and `0o755` mode, then returns that path from the helper executable hook.
  - That setup is not portable to Windows, where `/bin/sh` is unavailable and executable-bit semantics do not make a `.sh` file runnable.
  - Root cause: the test hard-codes a Unix-only process stub even though the production restart path has platform-specific launch behavior.
  - Fix approach: generate a platform-specific temporary helper stub in the test so Unix keeps the shell script path and Windows uses a `.cmd` file with Windows line endings and non-executable file mode.
  - Implemented in `internal/daemon/restart_test.go` by selecting the helper filename, contents, and file mode based on `runtime.GOOS`.
  - Verification: `go test ./internal/daemon ./internal/settings` and `make verify` both passed after the change.
