---
status: resolved
file: internal/registry/extract.go
line: 129
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUc,comment:PRRC_kwDOR5y4QM63madi
---

# Issue 006: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Reject symlinked paths inside an existing destination root.**

`CleanArchiveEntryPath` and `PathWithinRoot` only validate the lexical path. If `destRoot` already contains something like `review -> /outside`, extracting `review/SKILL.md` passes these checks and `os.OpenFile` follows the symlink outside the extraction root. Since `ExtractArchive` accepts pre-existing directories, this is still an extraction escape.



Also applies to: 229-247

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract.go` around lines 91 - 129, Reject extraction paths
that would traverse existing symlinks under destRoot by verifying the real
filesystem components after PathWithinRoot: in ExtractArchive, after computing
targetPath (using CleanArchiveEntryPath and PathWithinRoot), walk the path from
destRoot to targetPath and use os.Lstat on each intermediate component to detect
any symlink; if any component exists and is a symlink, return an error and do
not create directories or open files. Apply the same check before creating
directories (case tar.TypeDir) and before creating/opening files (case
tar.TypeReg/0), and duplicate the check at the other extraction block mentioned
(the similar logic around the other occurrence) so existing symlinked parents
cannot cause extraction escapes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `ExtractArchive()` only validates the lexical path. If an existing component under `destRoot` is a symlink, `os.MkdirAll()` and `os.OpenFile()` can follow it outside the extraction tree. I will add a filesystem-component check that rejects symlink traversal before directory or file creation and add regression coverage in `internal/registry/extract_test.go`; that test file is outside the listed scope but is the minimal place to validate the extractor hardening.
- Resolution: Added real filesystem-component validation in `internal/registry/extract.go` so extraction rejects existing symlink traversal before creating directories or files, with regression coverage in `internal/registry/extract_test.go`.
- Verification: `go test ./internal/registry/...`; `make verify`
