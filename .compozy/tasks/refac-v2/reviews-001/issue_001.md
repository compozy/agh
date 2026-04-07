---
status: resolved
file: internal/store/sqlite.go
line: 147
severity: critical
author: claude-code
provider_ref:
---

# Issue 001: SQLite recovery leaves WAL/SHM files causing corruption

## Review Comment

`recoverSQLiteDatabase` renames only the main `.db` file but leaves the `-wal` and `-shm` companion files in place. When `OpenSQLiteDatabase` creates a new database at the same path, SQLite discovers these stale WAL/SHM files and attempts to replay or use them. This can corrupt the newly created database or produce "not a database" errors, defeating the recovery purpose.

```go
// current: only renames the main file
func recoverSQLiteDatabase(path string) (string, error) {
    corruptPath := fmt.Sprintf("%s.corrupt.%s", path, ...)
    if err := os.Rename(path, corruptPath); err != nil {
        return "", err
    }
    return corruptPath, nil
}
```

**Fix:** Also rename (or remove) the WAL and SHM files:

```go
for _, suffix := range []string{"-wal", "-shm"} {
    companion := path + suffix
    if _, err := os.Stat(companion); err == nil {
        _ = os.Rename(companion, corruptPath+suffix)
    }
}
```

## Triage

- Decision: `valid`
- Root cause: `recoverSQLiteDatabase` only renames the primary database file. SQLite WAL mode uses sibling `-wal` and `-shm` files, so leaving those behind can contaminate the replacement database after a corruption recovery.
- Fix approach: Rename the companion WAL/SHM files alongside the main database when they exist, and add a regression test that exercises recovery with WAL sidecar files present.
- Resolution: Implemented and covered by store helper tests; full repository verification passed.
