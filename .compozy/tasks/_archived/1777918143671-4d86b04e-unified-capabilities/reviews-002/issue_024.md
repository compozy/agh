---
status: resolved
file: internal/session/manager_clear.go
line: 129
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyym,comment:PRRC_kwDOR5y4QM654Npd
---

# Issue 024: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Rollback partial renames inside `backupSessionDB()`.**

If renaming `session.db` succeeds and `session.db-wal` or `session.db-shm` fails afterward, this function returns with some artifacts already moved to `.clear-backup`. The deferred restore in `ClearConversation()` is not armed until `backupSessionDB()` succeeds, so this leaves the session store partially removed on error.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_clear.go` around lines 102 - 129, backupSessionDB
currently can leave partial .clear-backup files if a later rename fails; modify
backupSessionDB so that on any error during the loop (e.g., failed os.Rename for
one of the paths) it immediately attempts to roll back all previously renamed
files by renaming each entry in the backups slice from backup back to original
(use sessionDBBackup entries you appended), surface the original error after
attempting rollback, and ensure any rollback errors are logged or wrapped but do
not suppress the primary error; keep the rest of the function behavior (paths
list, removing stale backups, returning backups on success) intact so
ClearConversation can safely arm its deferred restore only after backupSessionDB
fully succeeds.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `backupSessionDB()` renames artifacts one by one and only lets `ClearConversation()` arm deferred restoration after the helper fully succeeds. If a later `os.Remove` or `os.Rename` fails, earlier artifacts can remain stranded as `.clear-backup`, leaving the session database partially removed.
  I will make `backupSessionDB()` roll back already-renamed artifacts immediately on loop failure and preserve the original error as the primary failure.
  Fixed and verified with targeted package tests plus `make verify`.
