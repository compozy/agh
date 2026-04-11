---
status: resolved
file: internal/extension/registry.go
line: 488
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAat,comment:PRRC_kwDOR5y4QM62zls5
---

# Issue 031: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Identify the SQLite driver in use and any existing typed constraint handling.
rg -n --type=go 'modernc\.org/sqlite|github\.com/mattn/go-sqlite3|zombiezen\.com/go/sqlite'
rg -n --type=go 'unique constraint failed: extensions\.name|ErrConstraint|ExtendedCode|sqlite.*Error|SQLITE_CONSTRAINT'
```

Repository: compozy/agh

Length of output: 218

---

🏁 Script executed:

```shell
#!/bin/bash
# Check the full context of the function and related error types
cat -n internal/extension/registry.go | sed -n '475,500p'
```

Repository: compozy/agh

Length of output: 901

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for ExtensionExistsError definition
rg -n --type=go 'type ExtensionExistsError' -A 3
```

Repository: compozy/agh

Length of output: 241

---

🏁 Script executed:

```shell
#!/bin/bash
# Check for any existing constraint error handling in the codebase
rg -n --type=go 'errors\.As\(|errors\.Is\(' internal/
```

Repository: compozy/agh

Length of output: 30662

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for modernc.org/sqlite imports and usage patterns
rg -n --type=go -B 2 -A 5 'sqlite\.' internal/store/
```

Repository: compozy/agh

Length of output: 37

---

🌐 Web query:

`modernc.org/sqlite constraint error handling golang errors.As`

💡 Result:

In modernc.org/sqlite (pure Go SQLite driver), constraint violations (e.g., UNIQUE, PRIMARY KEY, CHECK, FOREIGN KEY) return an error of type *sqlite.Error with Code == sqlite3.SQLITE_CONSTRAINT (19). To handle these errors in Go: 1. Type assert the error: import ( "database/sql" "modernc.org/sqlite" "modernc.org/sqlite/lib" // for sqlite3.SQLITE_CONSTRAINT ) err := db.ExecContext(ctx, "INSERT INTO users (id) VALUES (?)", existingID) if err != nil { if sqliteErr, ok := err.(*sqlite.Error); ok { if sqliteErr.Code == lib.SQLITE_CONSTRAINT { // Handle constraint violation return errors.New("duplicate key") } } return err } 2. Note: Unlike github.com/mattn/go-sqlite3 (which supports errors.As with its sqlite3.Error), modernc.org/sqlite uses type assertion on *sqlite.Error, as no sources confirm errors.As support. Official docs show (*Error).Code and ErrorCodeString mapping SQLITE_CONSTRAINT to "Abort due to constraint violation (SQLITE_CONSTRAINT)". For specific sub-constraints (e.g., PRIMARY KEY), check extended codes via sqliteErr.ExtendedCode == lib.SQLITE_CONSTRAINT_PRIMARYKEY (275), etc., if available (similar to standard SQLite). Sources confirm *sqlite.Error type, Code method, and SQLITE_CONSTRAINT handling via type check.

Citations:

- 1: http://godoc.org/modernc.org/sqlite
- 2: https://dev.to/bitecode/catch-error-when-using-sqlite-in-golang-58nn
- 3: https://golang-tools.github.io/sqlhelper/pkg/modernc.org/sqlite.html

---



</details>

**Use type assertion with `*sqlite.Error` instead of string matching for constraint detection.**

String-based constraint error matching is brittle across driver versions and can allow duplicate installs to silently fail. Since the codebase uses `modernc.org/sqlite`, use type assertion and check the error code instead:

```go
if sqliteErr, ok := err.(*sqlite.Error); ok && sqliteErr.Code == lib.SQLITE_CONSTRAINT {
    return &ExtensionExistsError{Name: name}
}
```

This aligns with the coding guideline to use explicit typed error handling rather than string comparison. Note: `modernc.org/sqlite` requires type assertion with `*sqlite.Error`, not `errors.As()`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry.go` around lines 479 - 488, The function
mapRegistryConstraintError currently matches constraint failures via string
matching on err.Error(); change this to a type assertion against *sqlite.Error
and check the error code (lib.SQLITE_CONSTRAINT) to detect unique-constraint
violations and return &ExtensionExistsError{Name: name} accordingly; update
mapRegistryConstraintError to assert err as (*sqlite.Error) and compare
sqliteErr.Code == lib.SQLITE_CONSTRAINT instead of using strings.Contains,
keeping the existing fallback fmt.Errorf wrapping for other errors.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The repository uses `modernc.org/sqlite`, so string-matching on the error text is brittle and unnecessary. Constraint detection should use the typed SQLite error code instead.
  Fix approach: switch `mapRegistryConstraintError` to a typed `*sqlite.Error` check against `SQLITE_CONSTRAINT`, with tests updated accordingly.
  Additional test scope needed: `internal/extension/registry_test.go` is outside the batch file list but is the minimal place to exercise the mapper directly.
