# TC-FUNC-020: CLI Extension Install From GitHub With Version

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 5 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate the full CLI install flow for a GitHub-hosted extension with an explicit version.

## Preconditions

- Valid GitHub repo with tagged releases containing extension archive.
- GitHub accessible (or mock server).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension install owner/repo --version v1.0.0` | **Expected:** Download, extract, verify, register. Success message printed. Exit code 0. |
| 2 | Verify extension directory at `~/.agh/extensions/<name>/` | **Expected:** Contains `extension.toml` and extension files. |
| 3 | Verify DB metadata | **Expected:** `registry_slug = "owner/repo"`, `registry_name = "github"`, `remote_version = "v1.0.0"`. |
| 4 | Verify restart message in output | **Expected:** Contains "Restart the daemon" or Phase 1 equivalent. |

## Edge Cases

- Version tag without `v` prefix: should work (e.g., `1.0.0`).
- Non-existent version: error with "release not found" message.
