---
status: resolved
file: internal/store/schema.go
line: 627
severity: medium
author: claude-code
provider_ref:
---

# Issue 004: SQL string interpolation in tableColumns PRAGMA query

## Review Comment

The `tableColumns` function builds a SQL statement via `fmt.Sprintf`:

```go
rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", strings.TrimSpace(table)))
```

While the `table` parameter currently only comes from hardcoded internal calls (`"sessions"`, `"workspaces"`), directly interpolating values into SQL strings is an anti-pattern that makes the codebase fragile if future callers pass untrusted input. CLAUDE.md explicitly calls out SQL injection as an OWASP concern to avoid.

**Suggested fix:** Since PRAGMA statements don't support parameterized queries in SQLite, validate the table name against a known allowlist:

```go
func tableColumns(ctx context.Context, exec sqlQueryExecutor, table string) (map[string]struct{}, error) {
    name := strings.TrimSpace(table)
    if !isValidTableName(name) {
        return nil, fmt.Errorf("store: invalid table name %q", name)
    }
    // ... proceed with fmt.Sprintf
}
```

## Triage

- Decision: `valid`
- Root cause: `tableColumns()` interpolates the table name directly into a `PRAGMA table_info(...)` statement without validating the identifier first. The current callers are internal and trusted, but the helper itself accepts arbitrary input and should enforce a safe identifier contract.
- Fix plan: validate the table name against a strict SQLite identifier rule before constructing the PRAGMA statement, then add tests covering invalid identifiers.

## Resolution

- Added strict SQLite identifier validation before constructing the `PRAGMA table_info(...)` query.
- Added regression coverage for invalid identifiers in the store helper tests.
- Verified with targeted package tests and `make verify`.
