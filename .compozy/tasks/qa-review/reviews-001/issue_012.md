---
status: resolved
file: internal/store/sessiondb/session_db.go
line: 760
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQy,comment:PRRC_kwDOR5y4QM67VX7Q
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don’t make `OpenSessionDB` fail on best-effort compaction.**

`VACUUM` is maintenance work, but it now sits in the critical open path. A transient failure here (`database is locked`, low disk space, etc.) will prevent the session DB from opening even though the schema/data are already usable after `RunMigrations` succeeds.



Also applies to: 792-804

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/sessiondb/session_db.go` around lines 756 - 760, OpenSessionDB
currently runs vacuumSessionSQLite inside the store.OpenSQLiteDatabase critical
path and returns its error, causing open to fail on transient VACUUM issues;
change the logic so that after RunMigrations(ctx, db, sessionSchemaMigrations)
succeeds you call vacuumSessionSQLite(ctx, db) but do not propagate its
error—log the error (with context) and continue returning nil so the DB opens
even if vacuum fails. Apply the same change for the other similar block that
calls vacuumSessionSQLite (the one around the 792-804 region) so both
OpenSQLiteDatabase callbacks never fail due to vacuumSessionSQLite errors.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `openSessionSQLite()` currently propagates `vacuumSessionSQLite()` failures after successful migrations, so best-effort compaction can block a usable session database from opening.
  - Root cause: the maintenance step runs inside the open callback as a hard failure path instead of as non-blocking cleanup.
  - Fix plan: keep migrations as the blocking gate, downgrade vacuum failures to logged warnings, and add a focused regression using an injected vacuum function so the open path can be tested without depending on a flaky SQLite lock.
