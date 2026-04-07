---
status: resolved
file: internal/store/globaldb/migrate_workspace.go
line: 246
severity: critical
author: claude-code
provider_ref:
---

# Issue 002: Migration _new tables reference old sessions table FK

## Review Comment

In `createMigratedGlobalTables`, `event_summaries_new`, `token_stats_new`, and `permission_log_new` all declare `REFERENCES sessions(id)` instead of `REFERENCES sessions_new(id)`. While foreign keys are disabled during migration (`PRAGMA foreign_keys = OFF`), between the `DROP TABLE sessions` (line ~335) and the `ALTER TABLE sessions_new RENAME TO sessions` (line ~339) there is a window where the FK metadata references a non-existent table. This is technically safe only because FKs are off, but the schema is misleading and fragile — if the migration is ever refactored to run with FKs enabled or the drop/rename order changes, it breaks silently.

```sql
-- line 248: should be sessions_new(id) not sessions(id)
CREATE TABLE event_summaries_new (
    session_id TEXT NOT NULL REFERENCES sessions(id), -- BUG
```

**Fix:** Change FK references in all `_new` tables to `REFERENCES sessions_new(id)`, or omit FK declarations entirely during migration and let `EnsureSchema` establish the correct schema afterward.

## Triage

- Decision: `valid`
- Root cause: The migration creates `_new` child tables with foreign keys that still point at `sessions` instead of `sessions_new`. That leaves the intermediate schema internally inconsistent during the swap sequence and makes the migration fragile to future changes.
- Fix approach: Point the `_new` foreign keys at `sessions_new` and extend migration tests so the generated schema matches the intended temporary table graph.
- Resolution: Implemented and covered by migration helper assertions; full repository verification passed.
