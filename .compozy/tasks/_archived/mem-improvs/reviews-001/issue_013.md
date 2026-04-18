---
status: resolved
file: internal/testutil/e2e/runtime_harness.go
line: 836
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745bC,comment:PRRC_kwDOR5y4QM65BAQW
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Resolve relative `workdir` values against the harness base directory.**

`exec.Cmd.Dir` treats relative paths as relative to the test process cwd, not `c.workdir`. Passing `"."` or another relative value here will therefore behave differently across runners. Joining relative paths to `c.workdir` keeps the harness deterministic.


<details>
<summary>🧭 Proposed fix</summary>

```diff
 	cmd := execabs.CommandContext(ctx, c.binaryPath, args...)
 	cmd.Env = append([]string(nil), c.env...)
-	cmd.Dir = strings.TrimSpace(workdir)
-	if cmd.Dir == "" {
-		cmd.Dir = c.workdir
-	}
+	trimmedDir := strings.TrimSpace(workdir)
+	switch {
+	case trimmedDir == "":
+		cmd.Dir = c.workdir
+	case filepath.IsAbs(trimmedDir):
+		cmd.Dir = trimmedDir
+	default:
+		cmd.Dir = filepath.Join(c.workdir, trimmedDir)
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (c *CLIClient) RunInDir(ctx context.Context, workdir string, args ...string) (string, string, error) {
	// `#nosec` G204 -- test helper intentionally shells out to the current agh test binary.
	cmd := execabs.CommandContext(ctx, c.binaryPath, args...)
	cmd.Env = append([]string(nil), c.env...)
	trimmedDir := strings.TrimSpace(workdir)
	switch {
	case trimmedDir == "":
		cmd.Dir = c.workdir
	case filepath.IsAbs(trimmedDir):
		cmd.Dir = trimmedDir
	default:
		cmd.Dir = filepath.Join(c.workdir, trimmedDir)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/runtime_harness.go` around lines 829 - 836, The
RunInDir behavior uses exec.Cmd.Dir which treats relative paths as relative to
the test process CWD; update CLIClient.RunInDir to resolve relative workdir
values against the harness base c.workdir: after trimming workdir, if it's empty
keep c.workdir; else if !filepath.IsAbs(workdir) set cmd.Dir =
filepath.Join(c.workdir, workdir) otherwise set cmd.Dir = workdir; ensure you
import "path/filepath" and preserve the existing fallback logic around c.workdir
and the use of execabs.CommandContext.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `CLIClient.RunInDir` currently assigns relative `workdir` values directly to `exec.Cmd.Dir`, which makes resolution depend on the test process working directory instead of the harness base directory.
  - I will resolve relative paths against `c.workdir` and add a focused test so harness behavior stays deterministic across runners.

## Resolution

- Updated `CLIClient.RunInDir` to resolve relative `workdir` values against the harness base directory while preserving empty and absolute path behavior.
- Added a focused harness test that proves a relative directory executes under `filepath.Join(c.workdir, workdir)`.
