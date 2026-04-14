---
status: resolved
file: internal/registry/extract.go
line: 250
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564r_M,comment:PRRC_kwDOR5y4QM63phdr
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't report the install as failed after the replacement already succeeded.**

By the time `os.RemoveAll(backupDir)` runs, both `os.Rename` calls have already committed the update. Returning an error here makes `internal/registry/installer.go` abort before checksum/provenance handling even though `targetDir` already contains the new package, which can leave on-disk state ahead of persisted state. Treat backup deletion as post-install cleanup instead of a fatal install failure.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract.go` around lines 249 - 250, The removal of
backupDir via os.RemoveAll(backupDir) is currently returned as an install
failure even though the replacement and os.Rename calls already succeeded;
change this to treat backup deletion as post-install cleanup by not returning
the error to callers—replace the returning fmt.Errorf with a non-fatal handling
(e.g., log a warning/error including the backupDir and err, and continue/return
nil) so installer logic in installer.go doesn't abort after a successful
replacement; optionally capture or emit the cleanup error for telemetry but do
not propagate it as a fatal install error.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `MoveInstalledDir(..., replaceExisting=true)` returns a fatal error after the target directory has already been successfully replaced if only backup cleanup fails.
- Evidence: [`internal/registry/extract.go`](internal/registry/extract.go) lines 238-251 complete both renames before `os.RemoveAll(backupDir)` can abort the operation.
- Fix plan: treat backup deletion as best-effort post-commit cleanup and add regression coverage so install success is not reported as failure after a committed replacement.
- Resolution: Backup cleanup is now best-effort after a committed replacement, with regression coverage around the non-fatal cleanup path. Verified with package tests and `make verify`.
