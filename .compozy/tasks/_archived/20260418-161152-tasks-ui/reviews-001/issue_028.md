---
status: resolved
file: internal/store/globaldb/migrate_workspace.go
line: 233
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb6,comment:PRRC_kwDOR5y4QM65B8fZ
---

# Issue 028: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Task-table rebuild drops indexes without recreating them.**

`DROP TABLE tasks` removes indexes attached to the old table, and the rebuild statement list only recreates/renames the table data. Indexes are not re-applied after rename, which can regress task list/dashboard/inbox query performance after migration.

<details>
<summary>💡 Suggested fix direction</summary>

```diff
 func taskTableMigrationStatements(spec taskTableMigrationSpec) []string {
-	return []string{
+	statements := []string{
 		taskTableCreateStatement(),
 		taskTableCopyStatement(spec),
 		`DROP TABLE tasks`,
 		`ALTER TABLE tasks_new RENAME TO tasks`,
-	}
+	}
+	// Re-apply the same task indexes used by fresh schema creation.
+	// Keep these statements sourced from the canonical schema/index definitions
+	// to avoid drift between bootstrap and migration paths.
+	statements = append(statements, taskTableIndexStatements()...)
+	return statements
 }
```

</details>


Also applies to: 236-317

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/migrate_workspace.go` around lines 227 - 233, The
migration drops the old tasks table and renames tasks_new but never re-creates
indexes, causing query regressions; update taskTableMigrationStatements (and
related logic around taskTableCreateStatement/taskTableCopyStatement) to collect
the existing index definitions for the original "tasks" table (e.g. via
pg_indexes/pg_get_indexdef) and append corresponding CREATE INDEX / CONCURRENTLY
statements after the `ALTER TABLE tasks_new RENAME TO tasks` step (or
alternatively recreate them before dropping the old table), ensuring unique
constraint/index expressions and names are preserved and any pg-specific options
(CONCURRENTLY) are respected.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the task-table rebuild migration drops `tasks`, renames `tasks_new`, and stops there, so every secondary task index attached to the old table is lost after migration.
- Fix approach: source the canonical task index statements from the schema definitions and append them after the `ALTER TABLE ... RENAME TO tasks` step so migrated databases retain the same indexes as fresh databases.

## Resolution

- Reused the canonical task index statement list in the migration path so rebuilt `tasks` tables reapply the same indexes as fresh schema creation.
- Added migration coverage that asserts the rebuilt task table retains its expected indexes.
- Verification: `go test ./internal/store/globaldb` and `go test -tags integration ./internal/store/globaldb`
