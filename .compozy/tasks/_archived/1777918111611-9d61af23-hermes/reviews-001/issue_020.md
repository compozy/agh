---
status: resolved
file: internal/memory/catalog.go
line: 90
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59lV11,comment:PRRC_kwDOR5y4QM67Ri1M
---

# Issue 020: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Add a real migration for the widened `memory_operation_log` table.**

This catalog still boots through `storepkg.EnsureSchema`, so existing databases keep the old five-column table. After this change, `logEvent` inserts `scope/workspace_root/filename` and `listOperations` selects them, which will fail on upgraded installs with missing-column errors.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/catalog.go` around lines 77 - 90, The schema change widened
the memory_operation_log table but existing DBs still have the old five-column
table; update the migration system (the code path used by storepkg.EnsureSchema)
to add a real migration that ALTER TABLE memory_operation_log ADD COLUMN scope
TEXT NOT NULL DEFAULT '' , ADD COLUMN workspace_root TEXT NOT NULL DEFAULT '' ,
ADD COLUMN filename TEXT NOT NULL DEFAULT '' and create the new indexes
(idx_memory_operation_log_scope, idx_memory_operation_log_workspace_root) so
upgraded installs won't fail; register this migration in the same migration
list/registry used by the catalog package so that logEvent and listOperations
(which now reference scope/workspace_root/filename) will run against databases
that have the new columns and indexes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The global DB already has a `memory_operation_log` migration, but `internal/memory/catalog.go` uses its own catalog SQLite schema through `storepkg.EnsureSchema`. Existing catalog DBs with the old operation-log table will not receive `scope`, `workspace_root`, `filename`, or the new indexes. Convert the catalog schema path to real migrations with an idempotent migration for the widened operation-log columns.
