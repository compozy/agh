---
status: resolved
file: internal/store/globaldb/migrate_workspace.go
line: 806
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59ReBO,comment:PRRC_kwDOR5y4QM662-hC
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
python - <<'PY'
import sqlite3

db = sqlite3.connect(":memory:")
cur = db.cursor()

cur.executescript("""
PRAGMA foreign_keys = ON;

CREATE TABLE workspaces (
  id TEXT PRIMARY KEY
);
INSERT INTO workspaces (id) VALUES ('ws-alpha');

CREATE TABLE network_channels (
  channel      TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  purpose      TEXT NOT NULL,
  created_by   TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);
INSERT INTO network_channels (
  channel, workspace_id, purpose, created_by, created_at, updated_at
) VALUES (
  'coord.core', ' ws-alpha ', 'Coordination', 'codex',
  '2026-04-11T12:00:00Z', '2026-04-11T12:00:00Z'
);

PRAGMA foreign_keys = OFF;
CREATE TABLE network_channels_new (
  channel      TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  purpose      TEXT NOT NULL,
  created_by   TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);
INSERT INTO network_channels_new (
  channel, workspace_id, purpose, created_by, created_at, updated_at
) SELECT
  channel, workspace_id, purpose, created_by, created_at, updated_at
FROM network_channels
WHERE TRIM(workspace_id) IN (SELECT id FROM workspaces);
DROP TABLE network_channels;
ALTER TABLE network_channels_new RENAME TO network_channels;
PRAGMA foreign_keys = ON;
""")

print("migrated rows:", cur.execute(
    "SELECT channel, quote(workspace_id) FROM network_channels"
).fetchall())

cur.execute("DELETE FROM workspaces WHERE id = 'ws-alpha'")
print("remaining rows after workspace delete:", cur.execute(
    "SELECT channel, quote(workspace_id) FROM network_channels"
).fetchall())
PY
```

Repository: compozy/agh

Length of output: 174

---

🏁 Script executed:

```shell
# Check file exists and examine the migration code at lines 801-806
if [ -f internal/store/globaldb/migrate_workspace.go ]; then
  echo "File exists. Showing lines 801-810:"
  sed -n '801,810p' internal/store/globaldb/migrate_workspace.go
else
  echo "File not found. Searching for migration files..."
  find . -name "*migrate*workspace*" -type f
fi
```

Repository: compozy/agh

Length of output: 499

---



</details>

**Trim `workspace_id` in the copied value, not only in the filter.**

A row like `" ws-alpha "` passes the `WHERE TRIM(workspace_id) IN (...)` predicate, but this migration still inserts the untrimmed `workspace_id`. That leaves a non-matching FK value in the rebuilt table, so the migrated row will not participate in the new `ON DELETE CASCADE` relationship.

<details>
<summary>Suggested fix</summary>

```diff
 		`INSERT INTO network_channels_new (
 			channel, workspace_id, purpose, created_by, created_at, updated_at
 		) SELECT
-			channel, workspace_id, purpose, created_by, created_at, updated_at
+			channel, TRIM(workspace_id), purpose, created_by, created_at, updated_at
 		FROM network_channels
 		WHERE TRIM(workspace_id) IN (SELECT id FROM workspaces)`,
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		`INSERT INTO network_channels_new (
			channel, workspace_id, purpose, created_by, created_at, updated_at
		) SELECT
			channel, TRIM(workspace_id), purpose, created_by, created_at, updated_at
		FROM network_channels
		WHERE TRIM(workspace_id) IN (SELECT id FROM workspaces)`,
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/migrate_workspace.go` around lines 801 - 806, The
INSERT selects untrimmed workspace_id but only trims it in the WHERE clause,
causing inserted FK values to include whitespace; update the SELECT to insert
TRIM(workspace_id) (e.g., SELECT channel, TRIM(workspace_id) AS workspace_id,
purpose, created_by, created_at, updated_at FROM network_channels WHERE
TRIM(workspace_id) IN (SELECT id FROM workspaces)) so the stored workspace_id
matches workspace ids and the new FK/ON DELETE CASCADE will work correctly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: The migration rebuild filters rows with `WHERE TRIM(workspace_id) IN (...)` but still copies the raw `workspace_id` into `network_channels_new`. A legacy row like `" ws-alpha "` therefore survives the filter yet keeps a foreign-key value that no longer matches `workspaces(id)`, which breaks the intended `ON DELETE CASCADE` relationship after migration.
- Fix plan: Update the `INSERT ... SELECT` in `internal/store/globaldb/migrate_workspace.go` to store `TRIM(workspace_id)` and extend the migration test to cover a whitespace-padded `workspace_id`.
- Outcome: The migration now writes `TRIM(workspace_id)`, and the migration test now proves whitespace-padded rows are preserved with a normalized FK and still cascade on workspace delete. Verified with `go test ./internal/store/globaldb -count=1` and `make verify`.
