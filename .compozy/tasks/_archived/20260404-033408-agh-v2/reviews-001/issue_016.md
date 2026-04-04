---
status: resolved
file: internal/store/global_db.go
line: 113
severity: medium
author: claude-code
provider_ref:
---

# Issue 016: UpdateSessionState silently succeeds on missing session

## Review Comment

The UPDATE statement at line 113 executes without checking `RowsAffected()`. If the session ID does not exist in the database, the UPDATE affects zero rows and returns nil error. The caller has no way to know the update was a no-op, which could mask bugs where sessions are not properly registered before state updates.

**Suggested fix:** Check `result.RowsAffected()` and return an error when zero rows are affected:

```go
result, err := g.db.ExecContext(ctx, query, args...)
if err != nil { return err }
affected, _ := result.RowsAffected()
if affected == 0 {
    return fmt.Errorf("store: session %q not found", update.ID)
}
```

## Triage

- Decision: `valid`
- Notes: `UpdateSessionState()` ignores `RowsAffected()`, so updating a missing session is silently treated as success. That masks registry consistency bugs and makes state-sync failures harder to detect.
