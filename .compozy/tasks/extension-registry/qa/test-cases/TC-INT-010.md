# TC-INT-010: End-to-End Install With Real SQLite

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Integration |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/installer.go` + `internal/store/globaldb/` |

## Objective

Validate the full install pipeline with a real SQLite database: download, extract, register, query back.

## Preconditions

- `t.TempDir()` for SQLite database and extension installation.
- Mock HTTP server for download.
- Valid tar.gz archive with `extension.toml`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Initialize globaldb with schema in temp directory | **Expected:** Database created with extensions table including new columns. |
| 2 | Install extension via `Installer.Install()` | **Expected:** Files extracted to temp dir. |
| 3 | Register extension via `Registry.Install()` with metadata | **Expected:** DB row created. |
| 4 | Query DB for installed extension | **Expected:** Row with `registry_slug`, `registry_name`, `remote_version` populated. |
| 5 | Remove extension via `removeInstalledExtension()` | **Expected:** Directory deleted, DB row removed. |
| 6 | Query DB again | **Expected:** No row found. |

## Edge Cases

- DB schema without new columns (old DB): migration should add columns gracefully.
- Concurrent install of same extension: SQLite UNIQUE constraint prevents duplicate.
