---
status: resolved
file: internal/store/schema.go
line: 450
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCm,comment:PRRC_kwDOR5y4QM61T6IF
---

# Issue 022: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Foreign key reference in `event_summaries_new` points to wrong table.**

The `REFERENCES sessions(id)` should reference `sessions_new(id)` since this is the migrated table being created. The same issue exists for `token_stats_new` (lines 451-462) and `permission_log_new` (lines 463-472).

However, since foreign keys are disabled during migration (`PRAGMA foreign_keys = OFF`) and the tables are renamed afterward, this may work in practice. Still, it's misleading and could cause issues if the migration logic changes.


<details>
<summary>🛠️ Suggested fix for clarity</summary>

```diff
 		`CREATE TABLE event_summaries_new (
 			id         TEXT PRIMARY KEY,
-			session_id TEXT NOT NULL REFERENCES sessions(id),
+			session_id TEXT NOT NULL REFERENCES sessions_new(id),
 			type       TEXT NOT NULL,
```

Apply similar changes to `token_stats_new` and `permission_log_new`.
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		`CREATE TABLE event_summaries_new (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions_new(id),
			type       TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			summary    TEXT,
			timestamp  TEXT NOT NULL
		);`,
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/schema.go` around lines 443 - 450, The CREATE TABLE statements
for event_summaries_new, token_stats_new, and permission_log_new incorrectly
reference sessions(id) (and any other old table names) in their FOREIGN KEY
clauses; update those REFERENCES to point to the new migrated tables (e.g.,
sessions_new(id)) so the foreign-key definitions match the newly created tables
(modify the REFERENCES in event_summaries_new, token_stats_new, and
permission_log_new to sessions_new or the appropriate *_new table names).
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  The migration intentionally points the child tables at the final table name
  `sessions`. During the swap, `sessions_new` is renamed to `sessions`, so keeping
  `REFERENCES sessions(id)` preserves the final schema without depending on
  SQLite rename-side effects to rewrite foreign-key targets. The current
  migration helper flow already exercises this path successfully. No change.
