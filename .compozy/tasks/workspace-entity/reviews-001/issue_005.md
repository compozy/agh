---
status: pending
file: internal/store/schema.go
line: 253
severity: high
author: claude-code
provider_ref:
---

# Issue 005: Legacy session meta.json files not migrated to workspace_id

## Review Comment

The `migrateGlobalSchema` function carefully migrates the SQLite `sessions` table from `workspace` (bare path) to `workspace_id` (FK to workspaces table). However, per-session `meta.json` files on disk are not migrated. These files still contain the old `"workspace": "/path"` field instead of `"workspace_id": "ws_xxx"`.

`SessionMeta` now defines `WorkspaceID string json:"workspace_id,omitempty"` and `Validate()` requires it to be non-empty. When `ReadSessionMeta` deserializes an old `meta.json`, the `"workspace"` key is silently ignored (no matching struct field), leaving `WorkspaceID` empty, and validation fails.

This means:
- `Resume` for pre-upgrade sessions fails at `ReadSessionMeta`
- `Status` for stopped pre-upgrade sessions fails
- `ListAll` scanning session dirs produces inconsistent results vs the (migrated) DB

The DB migration shows clear intent to preserve data, but the on-disk meta files are left behind.

**Suggested fix:** After the DB migration completes, scan `~/.agh/sessions/*/meta.json`, read each file, map the old `workspace` path to the new `workspace_id` using the same `workspaceIDs` map built during migration, and rewrite the meta file. Alternatively, teach `ReadSessionMeta` to accept a legacy `workspace` field and lazily backfill `workspace_id` on read.

## Triage

- Decision: `UNREVIEWED`
- Notes:
