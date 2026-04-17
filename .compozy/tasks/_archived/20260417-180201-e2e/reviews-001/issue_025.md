---
status: resolved
file: internal/testutil/e2e/config_seed.go
line: 189
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcG,comment:PRRC_kwDOR5y4QM640q0j
---

# Issue 025: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject workspace file paths that escape the seeded root.**

`filepath.Join(root, relativePath)` allows entries like `../outside.txt`, so one malformed fixture key can write outside the temporary workspace and break test isolation. Normalize the candidate path and fail if it does not stay under `root`. As per coding guidelines, Use `t.TempDir()` for filesystem isolation in Go tests.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/config_seed.go` around lines 182 - 189, Reject file
paths that escape the seeded root by validating the normalized target before
creating directories or writing files: after computing targetPath :=
filepath.Join(root, relativePath) call cleaned := filepath.Clean(targetPath)
(and cleanedRoot := filepath.Clean(root)) then ensure cleaned == cleanedRoot or
strings.HasPrefix(cleaned, cleanedRoot+string(os.PathSeparator)) (or alternately
use filepath.Rel and fail if it returns a path starting with ".."); if the check
fails call t.Fatalf with a clear error. Also ensure the temporary workspace root
is created with t.TempDir() (use that value for root) so tests get proper
filesystem isolation. Apply these checks around the existing
os.MkdirAll/os.WriteFile calls that iterate over opts.Files.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `SeedWorkspace` joins fixture keys directly under the workspace root, so a
  malformed relative path like `../outside.txt` can escape the seeded root and
  break test isolation. The fix is to validate each normalized target path
  before creating directories or writing fixture files.

## Resolution

- `SeedWorkspace` now rejects blank and escaping file paths before writing, and
  regression coverage exercises the path guard.
