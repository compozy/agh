---
status: resolved
file: internal/config/config_test.go
line: 346
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCH,comment:PRRC_kwDOR5y4QM61T6He
---

# Issue 011: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Tests using `os.Chdir` should not run in parallel.**

The test changes the process working directory, which is global state. While `t.Cleanup` restores it, this can interfere with other tests running concurrently. These tests lack `t.Parallel()`, which is correct, but consider adding a comment explaining why.


<details>
<summary>📝 Add clarifying comment</summary>

```diff
 func TestLoadWithoutWorkspaceRootIgnoresCurrentDirectoryWorkspaceFiles(t *testing.T) {
+	// Note: Does not use t.Parallel() because os.Chdir affects global process state.
 	homeRoot := filepath.Join(t.TempDir(), "home")
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/config_test.go` around lines 285 - 346, The test
TestLoadWithoutWorkspaceRootIgnoresCurrentDirectoryWorkspaceFiles mutates global
process state via os.Chdir, so it must not be run in parallel; add a short
clarifying comment above the test (near the function declaration) explaining
that the test intentionally omits t.Parallel() because it changes the working
directory with os.Chdir and uses t.Cleanup to restore it, which would race with
other tests if run concurrently.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The test intentionally avoids `t.Parallel()` because it changes the process working directory with `os.Chdir`.
  - That behavior is already correct, but the constraint is implicit and easy to forget during future cleanup.
  - I will add a short comment documenting why the test must remain non-parallel.
