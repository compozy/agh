---
status: resolved
file: internal/store/globaldb/global_db.go
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg3-,comment:PRRC_kwDOR5y4QM63ZMHs
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
python - <<'PY'
import sqlite3

conn = sqlite3.connect(":memory:")
conn.execute("""
CREATE TABLE t (
  message_id TEXT PRIMARY KEY,
  text TEXT NOT NULL
);
""")

conn.execute("INSERT INTO t(message_id, text) VALUES(NULL, 'first')")
try:
    conn.execute("INSERT INTO t(message_id, text) VALUES(NULL, 'second')")
    print("RESULT: second NULL insert allowed")
except Exception as e:
    print("RESULT: second NULL insert rejected:", e)

rows = list(conn.execute("SELECT rowid, message_id, text FROM t"))
print("ROWS:", rows)
PY
```

Repository: compozy/agh

Length of output: 137

---

🏁 Script executed:

```shell
cat -n internal/store/globaldb/global_db.go | sed -n '85,110p'
```

Repository: compozy/agh

Length of output: 1219

---

🏁 Script executed:

```shell
# Check if there are any other constraints, triggers, or enforcement around message_id in this table
rg -A 20 "network_message_log" internal/store/globaldb/global_db.go | head -40
```

Repository: compozy/agh

Length of output: 1252

---



</details>

**Add `NOT NULL` constraint to `message_id` PRIMARY KEY.**

Line 93 defines `message_id TEXT PRIMARY KEY` without an explicit `NOT NULL` constraint. SQLite allows `NULL` values in `TEXT PRIMARY KEY` columns for rowid tables, which violates the deduplication and identity guarantees for this log table. This permits multiple rows with `NULL` message_id, breaking the uniqueness contract.

<details>
<summary>Proposed schema fix</summary>

```diff
-		message_id TEXT PRIMARY KEY,
+		message_id TEXT PRIMARY KEY NOT NULL,
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	`CREATE TABLE IF NOT EXISTS network_message_log (
		message_id TEXT PRIMARY KEY NOT NULL,
		session_id TEXT,
		channel    TEXT NOT NULL,
		peer_from  TEXT NOT NULL,
		kind       TEXT NOT NULL,
		intent     TEXT,
		text       TEXT NOT NULL,
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_net_msg_channel_ts ON network_message_log(channel, timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_msg_peer_ts ON network_message_log(peer_from, timestamp);`,
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db.go` around lines 92 - 103, The CREATE TABLE
for network_message_log uses "message_id TEXT PRIMARY KEY" which can still
accept NULLs in SQLite; update the CREATE TABLE statement for
network_message_log to declare message_id as NOT NULL (e.g., message_id TEXT NOT
NULL PRIMARY KEY) to enforce uniqueness, and add a migration/repair step in the
global DB initialization (the code that runs this DDL in global_db.go) to
deduplicate or backfill any existing rows with NULL message_id before applying
the NOT NULL constraint so the schema change won't fail at runtime.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: the reported duplicate-`NULL` scenario is not reachable through the product code path. `GlobalDB.WriteNetworkMessage` calls `store.NetworkMessageEntry.Validate()`, which requires a non-empty `MessageID`, and then inserts a concrete Go string value rather than a nullable parameter.
- Reasoning: reproducing the review comment requires bypassing the store API with ad hoc SQL. Adding a migration/repair path for unreachable legacy rows is not warranted for this batch and conflicts with the repo's zero-legacy-tolerance guidance.
