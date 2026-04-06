---
status: pending
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

- Decision: `UNREVIEWED`
- Notes:
