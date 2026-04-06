---
status: resolved
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

- Decision: `valid`
- Root cause: the global DB migration rewrites legacy `sessions.workspace` rows into `sessions.workspace_id`, but no equivalent rewrite is performed for on-disk `meta.json` files that still encode the legacy `workspace` path field. Current readers then reject those files because `workspace_id` is required.
- Fix plan: extend the global schema migration flow to reconcile legacy session metadata files against the migrated workspace table and rewrite immediate-legacy `meta.json` files with the stable `workspace_id`.
- Scope note: this may require limited supporting test changes outside `internal/store/schema.go`, but the production fix should stay rooted in the migration layer rather than adding a permanent workaround at the read boundary.

## Resolution

- Added post-schema reconciliation that scans sibling session directories, maps legacy `workspace` paths to migrated workspace IDs, and rewrites compatible legacy `meta.json` files with `workspace_id`.
- Added an integration-style regression test covering legacy DB migration plus on-disk metadata rewrite.
- Verified with targeted package tests and `make verify`.
