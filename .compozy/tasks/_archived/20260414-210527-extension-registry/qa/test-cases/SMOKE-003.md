# SMOKE-003: Extension Install from GitHub Completes

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Smoke |
| **Estimated Time** | 5 min |
| **Module** | CLI / Installer |

## Objective

Validate that `agh extension install <slug>` downloads, extracts, verifies, and registers an extension from GitHub Releases.

## Preconditions

- AGH binary built.
- GitHub source configured in `[extensions.marketplace]`.
- A known public GitHub repo with a valid extension release (e.g., a test extension).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension install <owner>/<repo> --version <tag>` | **Expected:** Download progress shown. "Extension installed" message. Exit code 0. |
| 2 | Verify extension directory exists at `~/.agh/extensions/<name>/` | **Expected:** Directory exists with extracted files including `extension.toml`. |
| 3 | Verify database entry: query `extensions` table for the installed name | **Expected:** Row exists with `registry_slug`, `registry_name`, `remote_version` populated. |
| 4 | Verify restart message printed | **Expected:** Output includes "Restart the daemon to activate" or equivalent Phase 1 message. |

## Edge Cases

- Install with `--version` pointing to non-existent tag: should fail with clear error.
- Install same extension twice: should fail with "already installed" or replace with confirmation.
