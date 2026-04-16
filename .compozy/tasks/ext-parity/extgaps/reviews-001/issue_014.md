---
status: resolved
file: internal/extension/install_managed.go
line: 249
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__B6,comment:PRRC_kwDOR5y4QM63zbyb
---

# Issue 014: _⚠️ Potential issue_ | _🔴 Critical_

## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Guard against symlink directory cycles before recursing.**

A symlink like `a -> .` or `a -> ..` now resolves to a directory already on the current traversal path, and `copyInstallSymlink` immediately walks it again. That can recurse indefinitely and blow up the install on cyclic links. Please track the resolved-directory stack for the current copy and reject/recover when a symlink resolves back into an ancestor.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/install_managed.go` around lines 230 - 249,
copyInstallSymlink currently follows symlinks and immediately calls
copyInstallDirectoryContents without guarding against directory cycles; modify
copyInstallSymlink to detect and reject cycles by tracking a visited set of
resolved directory paths (e.g., map[string]struct{}) for the current copy
operation, check if resolvedPath is already in that set and return a clear error
if so, and pass this visited set down into copyInstallDirectoryContents (and any
other recursive helpers) so every recursive entry first records the resolved
path and checks for repeats before recursing.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: symlinked directories are materialized by recursively copying the resolved target, but the current recursion does not track ancestor directories. A symlink that resolves back to the current path or one of its parents can recurse indefinitely.
- Fix plan: thread an ancestor-directory stack through the install-tree copy helpers, detect when a resolved directory target re-enters the current traversal path, and fail with a clear cycle error before recursing.
- Resolution: threaded an active-directory stack through the install copy recursion and added explicit cycle detection before descending into resolved symlinked directories.
- Verification: added the cycle regression in `internal/extension/install_managed_test.go` and passed `go test ./internal/extension` plus `make verify`.
