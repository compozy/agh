# TC-FUNC-023: CLI Extension Install Local Path (Non-Registry)

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that the existing local-path install flow (`PrepareLocalExtensionInstallIfPresent`) still works alongside the new marketplace flow.

## Preconditions

- Local directory with valid `extension.toml` manifest.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension install /path/to/local-extension` | **Expected:** Extension installed from local path. No registry metadata set in DB. |
| 2 | Verify `registry_slug`, `registry_name`, `remote_version` are all NULL in DB | **Expected:** NULL values — local installs have no registry provenance. |
| 3 | Run `agh extension update --check` | **Expected:** Local extension skipped (no registry to check against). |

## Edge Cases

- Path is a file, not a directory: should fail with clear error.
- Path does not exist: should fail with "directory not found".
- Path contains symlinks: should resolve and validate.
