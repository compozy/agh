# TC-REG-003: Existing Extension Local Install Still Works

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Regression |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |
| **Changed In** | Task 04 — Extension CLI Commands |

## Objective

Validate that local-path extension installation (`PrepareLocalExtensionInstallIfPresent`) is unaffected by the new marketplace install path.

## Preconditions

- Local directory with valid `extension.toml`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension install /path/to/local/ext` | **Expected:** Extension installed from local path (no registry query). |
| 2 | Verify no registry metadata in DB | **Expected:** `registry_slug`, `registry_name`, `remote_version` are all NULL. |
| 3 | Run `agh extension update --check` | **Expected:** Local extension skipped (no registry to check). |

## Regression Risk

Medium — the install command now has two code paths (local vs marketplace). Path detection logic could misroute.
