---
status: resolved
file: internal/store/globaldb/global_db_automation_scheduler.go
line: 440
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:57648dc154d1
review_hash: 57648dc154d1
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 030: Use structured error checking instead of string matching.
## Review Comment

Line 442 uses `strings.Contains(strings.ToLower(err.Error()), ...)` to detect constraint violations, which violates the coding guideline requiring `errors.Is()` or structured error inspection.

The codebase already uses the correct pattern with `modernc.org/sqlite`: check for `*sqlite.Error` using type assertion and inspect the error code (see `internal/extension/registry.go:755`). Replace the string matching with:

```go
var sqliteErr *sqlite.Error
if errors.As(err, &sqliteErr) && sqliteErr.Code()&0xff == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
return fmt.Errorf(...)
}
```

Note: This same pattern should also be applied to `internal/store/globaldb/global_db_workspace.go` and `internal/store/globaldb/global_db_automation.go`, which use string matching for similar constraint checks.

## Triage

- Decision: `valid`
- Root cause: `insertAutomationRunTx` detects duplicate scheduled fires by lowercasing and substring-matching the SQLite error text. This is brittle and violates the project rule to use structured error matching instead of comparing error strings.
- Fix approach: replace the string check in `internal/store/globaldb/global_db_automation_scheduler.go` with `errors.As` against `*sqlite.Error` and the SQLite extended unique-constraint code, preserving the existing `automation.ErrScheduledFireAlreadyClaimed` behavior. The review note mentions similar patterns in other files, but this batch scope is limited to the listed scheduler file.
