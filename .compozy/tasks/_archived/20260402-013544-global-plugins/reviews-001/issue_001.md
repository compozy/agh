---
status: resolved
file: internal/plugins/install.go
line: 818
severity: high
author: claude-code
provider_ref:
---

# Issue 001: backupFile loop has dead error check causing infinite loop

## Review Comment

The `backupFile` function at line 814-822 contains a variable scoping bug that makes the error check on line 818 dead code, potentially causing an infinite loop.

```go
for i := 1; ; i++ {
    if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
        break
    }
    if err != nil {  // BUG: this `err` is the OUTER scope's err (always nil here)
        return "", fmt.Errorf("stat backup destination %q: %w", backupPath, err)
    }
    backupPath = fmt.Sprintf("%s.bak.%d", path, i)
}
```

The `err` declared in the `if _, err := os.Stat(...)` is scoped to that `if` statement. The subsequent `if err != nil` references the outer `err` from line 808 (`info, err := os.Stat(path)`), which is guaranteed to be `nil` at this point. If `os.Stat(backupPath)` returns a non-`ErrNotExist` error (e.g., `EACCES`), the loop never breaks and iterates unboundedly.

**Fix:** Use a regular variable declaration to unify the scope:

```go
for i := 1; ; i++ {
    _, statErr := os.Stat(backupPath)
    if errors.Is(statErr, os.ErrNotExist) {
        break
    }
    if statErr != nil {
        return "", fmt.Errorf("stat backup destination %q: %w", backupPath, statErr)
    }
    backupPath = fmt.Sprintf("%s.bak.%d", path, i)
}
```

## Triage

- Decision: `valid`
- Notes:
  - Confirmed in `internal/plugins/install.go`: the loop uses `if _, err := os.Stat(backupPath); ...` and then checks `if err != nil` outside that `if` initializer scope.
  - The second `err` reference resolves to the earlier `info, err := os.Stat(path)` binding, which is already known to be `nil` on this path.
  - Result: non-`ErrNotExist` failures while probing `backupPath` are ignored instead of being surfaced, and the function falls through to alternate suffixes. This is a real control-flow bug in backup collision handling and needs a production fix plus regression coverage.
  - Fixed by moving the backup-destination probe to a dedicated `statErr` variable and returning that error immediately when it is not `os.ErrNotExist`.
  - Added regression coverage in `internal/plugins/install_test.go` using a self-referential symlink at `path.bak` so `os.Stat` returns a real non-`ErrNotExist` error and the function now fails instead of silently skipping to `.bak.1`.
  - Verification: `go test ./internal/plugins` and `make verify` both passed after the fix.
