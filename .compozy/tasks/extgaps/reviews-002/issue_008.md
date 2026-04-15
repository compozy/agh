---
status: resolved
file: internal/extension/install_managed.go
line: 155
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57AO3v,comment:PRRC_kwDOR5y4QM63zyQA
---

# Issue 008: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Reject symlink targets that escape the extension root.**

`EvalSymlinks` now dereferences the link and copies whatever it points to, even if that target is outside `sourceDir`. A package can therefore smuggle arbitrary host files/directories into the managed install via a symlink. Please carry the canonical source root through the traversal helpers and reject any resolved target whose `filepath.Rel(sourceRoot, resolvedPath)` escapes that root. 

<details>
<summary>Suggested direction</summary>

```diff
-func copyInstallSymlink(sourcePath string, targetPath string, activeDirs map[string]struct{}) error {
+func copyInstallSymlink(sourceRoot string, sourcePath string, targetPath string, activeDirs map[string]struct{}) error {
 	resolvedPath, err := filepath.EvalSymlinks(sourcePath)
 	if err != nil {
 		return fmt.Errorf("extension: resolve source symlink %q: %w", sourcePath, err)
 	}
+
+	relToRoot, err := filepath.Rel(sourceRoot, resolvedPath)
+	if err != nil {
+		return fmt.Errorf("extension: relate symlink target %q to source root %q: %w", resolvedPath, sourceRoot, err)
+	}
+	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
+		return fmt.Errorf("extension: symlink target %q escapes source root %q", sourcePath, sourceRoot)
+	}
```

Plumb `sourceRoot` from `copyInstallTree` through `copyInstallDirectoryContents` and `copyInstallEntry`, and seed it from the canonicalized root path.
</details>


Also applies to: 175-197, 238-263

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/install_managed.go` around lines 153 - 155,
copyInstallTree currently canonicalizes the source root but does not pass it
down, so EvalSymlinks in copyInstallDirectoryContents/copyInstallEntry can
dereference links outside the extension root; plumb the canonical source root
from copyInstallTree into copyInstallDirectoryContents and copyInstallEntry
(seed it with the canonicalized absSourceRoot) and, after resolving any symlink
target, call filepath.Rel(sourceRoot, resolvedPath) and reject the entry if the
result escapes (starts with ".." or returns an error). Ensure all traversal
sites mentioned (the calls around lines 175-197 and 238-263) use the new
sourceRoot parameter and perform the same relative-root check before copying.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `copyInstallSymlink(...)` dereferences symlink targets and copies their contents after `EvalSymlinks(...)`, but it does not verify that the resolved target still lives under the canonical extension source root.
- Why this is valid: that allows a crafted extension payload to smuggle arbitrary host files or directories into the managed install via symlinks that point outside the package tree.
- Fix approach: carry the canonical source root through the recursive copy helpers, reject any resolved target whose relative path escapes that root, and add regression coverage for both file and directory symlink escapes in `internal/extension/install_managed_test.go`.
- Resolution: `internal/extension/install_managed.go` now canonicalizes the source root, carries it through the recursive copy helpers, rejects escaped symlink targets, and keeps cycle detection on canonical paths. `internal/extension/install_managed_test.go` now covers both allowed in-root symlinks and rejected out-of-root file/directory escapes.
- Verification: targeted `go test ./internal/extension -run 'TestCopyInstallTreeMaterializesSymlinkTargets|TestInstallLocalManagedUsesInstalledChecksumForMaterializedSymlinks|TestInstallLocalManagedNormalizesProvidedChecksum|TestCopyInstallTreeRejectsSymlinkDirectoryCycles|TestCopyInstallTreeRejectsSymlinkTargetsOutsideSourceRoot|TestInstallLocalManagedWrapsPhaseErrors' -count=1` passed, followed by a clean `make verify`.
