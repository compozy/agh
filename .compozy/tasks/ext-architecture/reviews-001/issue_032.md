---
status: resolved
file: internal/extension/registry.go
line: 527
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAax,comment:PRRC_kwDOR5y4QM62zls-
---

# Issue 032: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Hashing symlinks by target path does not protect the target contents.**

A payload can include a symlink whose target lives outside the extension directory; changing that target later changes runtime behavior without changing the stored checksum, because only the link text is hashed here. That undermines the integrity check for executable/resource files. Reject symlinks entirely, or resolve and hash only in-tree targets.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry.go` around lines 518 - 527, The current code
hashes only the symlink text (os.Readlink) which allows external targets to
change without invalidating the checksum; change the symlink handling in the
checksum routine (the branch that checks info.Mode()&os.ModeSymlink) to either
(A) reject symlinks outright by returning a clear error (e.g., "symlinks are not
allowed in extensions") or (B) resolve the symlink target and only accept it if
it resolves inside the extension root, then replace the existing
writeChecksumString call with logic that resolves the target path, stat/resolve
the target file, and include the target's canonical path and its actual
content/metadata into the hasher (using the same writeChecksumString/hasher
flow) instead of hashing the raw link text; use symbols shown (absPath,
normalizedPath, os.Readlink, writeChecksumString, hasher, filepath.ToSlash,
info.Mode()) to locate and update the code, and return an error if the target is
outside the extension tree.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The checksum helper currently hashes symlink metadata, not the target content, so a symlink pointing outside the artifact can change runtime behavior without invalidating the checksum. That weakens the integrity guarantee.
  Fix approach: reject symlinks from extension payload checksums instead of hashing only the link text.
  Additional test scope needed: `internal/extension/registry_test.go` is outside the batch file list but is the minimal place to lock down checksum behavior.
