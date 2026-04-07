---
status: resolved
file: internal/store/sql_helpers.go
line: 37
severity: high
author: claude-code
provider_ref:
---

# Issue 003: SQL injection vector in exported SQL helper functions

## Review Comment

`TimeClause` and `Int64Clause` accept an `op string` parameter that is interpolated directly into SQL via `fmt.Sprintf("%s %s ?", column, op)`. Both `column` and `op` are unsanitized. While current callers only pass literal `">="` and `">"`, these are exported functions — any future caller (or imported consumer) could inject arbitrary SQL through the `op` or `column` parameters.

`StringClause` has the same issue with `column` but is lower risk since it only uses `=`.

```go
func TimeClause(column string, op string, value time.Time) Clause {
    // op is interpolated directly into SQL
    return Clause{sql: fmt.Sprintf("%s %s ?", column, op), ...}
}
```

**Fix:** Validate `op` against a whitelist (`>=`, `>`, `<=`, `<`, `=`, `!=`) and validate `column` against `NormalizeSQLiteIdentifier`. Or make these functions unexported to limit the attack surface.

## Triage

- Decision: `valid`
- Root cause: `StringClause`, `TimeClause`, and `Int64Clause` interpolate identifiers and operators directly into SQL. Current callers are safe, but the helpers are shared across packages and do not defend their own SQL surface.
- Fix approach: Validate identifiers via the existing SQLite identifier normalizer, whitelist comparison operators, and add tests for rejected operator/column input.
- Resolution: Implemented with fail-closed clause handling and regression coverage; full repository verification passed.
