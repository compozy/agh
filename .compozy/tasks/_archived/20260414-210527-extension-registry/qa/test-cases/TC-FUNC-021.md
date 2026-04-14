# TC-FUNC-021: CLI Extension Install Latest Version (No --version)

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that omitting `--version` installs the latest release.

## Preconditions

- GitHub repo with multiple tagged releases.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension install owner/repo` | **Expected:** Installs the latest non-prerelease, non-draft release. |
| 2 | Verify installed version matches the latest release tag | **Expected:** `remote_version` in DB matches latest tag. |

## Edge Cases

- Repo with only pre-release versions: should fail or install nothing (per design).
- Repo with only draft releases: should fail with "no releases found".
