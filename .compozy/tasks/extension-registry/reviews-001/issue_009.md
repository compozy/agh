---
status: resolved
file: internal/registry/installer.go
line: 460
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUx,comment:PRRC_kwDOR5y4QM63maeB
---

# Issue 009: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Reject manifest symlinks at the archive root.**

`fileExists()` uses `os.Stat`, so `extension.toml` / `SKILL.md` can be a symlink. That lets an archive satisfy the root-manifest check with a link that resolves outside the extracted payload, and `parseInstalledPackageMetadata()` will then read that external target. Require a regular file here instead of following links.  


<details>
<summary>🔒 Proposed fix</summary>

```diff
 func fileExists(path string) (bool, error) {
-	info, err := os.Stat(path)
+	info, err := os.Lstat(path)
 	switch {
 	case err == nil:
-		return !info.IsDir(), nil
+		return info.Mode().IsRegular(), nil
 	case errors.Is(err, os.ErrNotExist):
 		return false, nil
 	default:
 		return false, err
 	}
 }
```
</details>


Also applies to: 629-638

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/installer.go` around lines 439 - 460, manifestPathAtRoot
currently accepts manifests via fileExists (which uses os.Stat), allowing
symlinks that can point outside the extracted archive; change the check to
require a regular non-symlink file: replace or supplement fileExists usage in
manifestPathAtRoot (and the other similar manifest-check block) with a function
that uses os.Lstat and validates that the path exists and its FileMode indicates
a regular file (not ModeSymlink, dir, or other non-regular types) and return an
error when a manifest is a symlink or non-regular file; ensure errors reference
installerExtensionManifestName and installerSkillManifestName and preserve
existing error-wrapping behavior so parseInstalledPackageMetadata cannot read
outside targets.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `manifestPathAtRoot()` relies on `fileExists()`, which uses `os.Stat()` and therefore treats a symlinked `extension.toml` or `SKILL.md` as a valid root manifest. That allows the installer to follow an external target during metadata parsing. I will require a regular non-symlink manifest file in `internal/registry/installer.go` and add regression coverage in the in-scope `internal/registry/installer_test.go`.
- Resolution: Replaced the manifest existence check with an `os.Lstat()`-based regular-file guard in `internal/registry/installer.go` and added a symlinked-manifest regression test in `internal/registry/installer_test.go`.
- Verification: `go test ./internal/registry/...`; `make verify`
