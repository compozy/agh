---
status: resolved
file: internal/store/globaldb/global_db_task_aux.go
line: 503
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb5,comment:PRRC_kwDOR5y4QM65B8fY
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`For SQLite tables without an explicit INTEGER PRIMARY KEY, are `rowid` values guaranteed to remain stable across VACUUM or table rebuilds?`

💡 Result:

No, for SQLite tables without an explicit INTEGER PRIMARY KEY, rowid values are not guaranteed to remain stable across VACUUM or table rebuilds. The official SQLite documentation explicitly states that the VACUUM command will change rowids for tables that do not declare an INTEGER PRIMARY KEY, as the rowid is not persistent in such cases. This applies to rowid tables (those with an implicit rowid, not WITHOUT ROWID). Table rebuilds, such as those performed by VACUUM or other operations that reorganize the B-tree, can similarly reassign rowids since they are not fixed without an INTEGER PRIMARY KEY alias.

Citations:

- 1: https://www.sqlite.com/rowidtable.html
- 2: https://www.sqlite.org/rowidtable.html
- 3: https://www2.sqlite.org/rowidtable.html
- 4: https://sqlite.org/lang_vacuum.html

---

🏁 Script executed:

```shell
# Find table schema definition for task_events
rg "task_events" --type go -B 5 -A 5
```

Repository: compozy/agh

Length of output: 10155

---

🏁 Script executed:

```shell
# Check the actual schema or migration files
fd -e sql -e go | head -20
```

Repository: compozy/agh

Length of output: 737

---

🏁 Script executed:

```shell
# Look for the table definition or schema
rg "CREATE TABLE.*task_events" -B 2 -A 10
```

Repository: compozy/agh

Length of output: 1195

---

🏁 Script executed:

```shell
# Check rows.Close() error handling in the file
rg "rows\.Close\(\)" internal/store/globaldb/global_db_task_aux.go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 286

---



</details>

**Don't use SQLite `rowid` as the external event sequence, and fix ignored error closures.**

The `task_events` table has `id TEXT PRIMARY KEY` without an explicit `INTEGER PRIMARY KEY` alias, making its `rowid` unstable. Operations like `VACUUM` or table rebuilds can renumber it, so reconnecting clients can skip or duplicate events once this sequence escapes the database boundary. Persist an explicit monotonic sequence column instead of exposing `rowid`.

Additionally, `rows.Close()` errors are ignored throughout this code (e.g., lines 141-143, 544-546). Replace `defer func() { _ = rows.Close() }()` with proper error handling or a documented justification per the coding guidelines.

Also applies to: 528-537

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_task_aux.go` around lines 497 - 503, The
code currently exposes SQLite's internal rowid (see the SELECT that returns
rowid) which is unstable; add and use an explicit monotonic sequence column
(e.g., sequence INTEGER PRIMARY KEY AUTOINCREMENT or a separate sequence field
like event_seq INTEGER NOT NULL UNIQUE with DB-side auto-increment behavior) in
the task_events schema and replace all uses of "rowid" in queries and ordering
with that new sequence column (update INSERTs, SELECTs, and any
ordering/pagination logic that referenced rowid). Also replace the ignored close
patterns (defer func() { _ = rows.Close() }() and similar) with proper error
handling: capture the error from rows.Close(), propagate or log it alongside the
primary error (e.g., if scan returns err and closeErr != nil, wrap/return a
combined error), or explicitly document why close errors are safe to ignore per
coding guidelines; update the routines referencing rows.Close() (the
QueryRowContext/rows handling around the SELECT of event by id and the other
rows loops) to implement this change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `GetTaskEventRecord` and `ListTaskEventRecords` expose SQLite `rowid` as the external event sequence. That sequence is not stable across `VACUUM`/table rebuilds for this schema because `task_events` uses `id TEXT PRIMARY KEY` rather than `INTEGER PRIMARY KEY`.
- Root cause: the same code path also contains ignored `rows.Close()` handling, which needs fixing alongside the sequencing bug.
- Fix approach: add an explicit durable event sequence column to `task_events`, backfill/persist it through schema creation and migration, and switch the sequence reads/order/filter logic from `rowid` to the new persisted sequence. Also fix the scoped close-error handling in the same file.

## Resolution

- Added a persisted `event_seq` column plus canonical indexes to the `task_events` schema and switched event-record reads/orderings to use it instead of SQLite `rowid`.
- Added migration logic to backfill `event_seq` for legacy databases and retain stable replay ordering after rebuilds or `VACUUM`.
- Replaced the ignored `rows.Close()` paths in the same file with explicit close/error joining.
- Verification: `go test ./internal/store/globaldb` and `go test -tags integration ./internal/store/globaldb`
