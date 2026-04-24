---
status: resolved
file: internal/store/globaldb/global_db.go
line: 142
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeX,comment:PRRC_kwDOR5y4QM66CAk8
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Add workspace referential integrity to `network_channels`.**

`workspace_id` is free text here, so deleting a workspace can leave orphaned channel rows behind. Since this table is workspace-scoped state, it should reference `workspaces(id)` directly and cascade on delete.

<details>
<summary>Suggested schema change</summary>

```diff
 `CREATE TABLE IF NOT EXISTS network_channels (
 		channel      TEXT PRIMARY KEY,
-		workspace_id TEXT NOT NULL,
+		workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 		purpose      TEXT NOT NULL,
 		created_by   TEXT NOT NULL DEFAULT '',
 		created_at   TEXT NOT NULL,
 		updated_at   TEXT NOT NULL
 	);`,
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	`CREATE TABLE IF NOT EXISTS network_channels (
		channel      TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
		purpose      TEXT NOT NULL,
		created_by   TEXT NOT NULL DEFAULT '',
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL
	);`,
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db.go` around lines 135 - 142, The
network_channels table currently defines workspace_id as free text which can
leave orphaned rows; modify the CREATE TABLE for network_channels to make
workspace_id a proper foreign key referencing workspaces(id) with ON DELETE
CASCADE (i.e., add a FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON
DELETE CASCADE and ensure the column types match the referenced id), and update
any migration/initializer that uses the network_channels schema so the
constraint is applied for new deployments (or provide an ALTER TABLE migration
to add the foreign key for existing DBs).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `network_channels.workspace_id` is currently free text, so deleting a workspace can leave orphaned channel metadata behind. Because this table is workspace-scoped durable state, the schema should enforce the relationship directly.
- Fix plan: add `REFERENCES workspaces(id) ON DELETE CASCADE` to the table definition, update the existing-schema migration path so older databases are rebuilt with the foreign key, and add schema/migration coverage. This requires a minimal migration change outside the listed file set because `CREATE TABLE` only fixes new databases.
- Resolution: added a workspace foreign key with `ON DELETE CASCADE` to `network_channels`, updated the legacy schema migration path in `migrate_workspace.go` to rebuild existing tables safely, and added schema and migration coverage.
- Verification: `go test ./internal/store/globaldb` and `make verify`
