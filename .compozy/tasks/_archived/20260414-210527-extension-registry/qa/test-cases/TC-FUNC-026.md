# TC-FUNC-026: CLI Extension Update Install Mode

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 5 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `agh extension update <name>` and `agh extension update --all` download and install available updates.

## Preconditions

- Marketplace-installed extension at version 1.0.0.
- Registry source returns latest version 2.0.0.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension update ext-name` | **Expected:** Downloads and installs v2.0.0. Success message with new version. |
| 2 | Verify DB `remote_version` updated to "2.0.0" | **Expected:** Database reflects new version. |
| 3 | Verify filesystem has new version's files | **Expected:** Extension directory contains v2.0.0 content. |
| 4 | Run `agh extension update --all` with multiple extensions | **Expected:** All marketplace extensions with updates are updated. |
| 5 | Run `agh extension update ext-name` (already latest) | **Expected:** "Already up to date" message. |

## Edge Cases

- Update fails mid-install: old version should remain intact (backup-on-replace).
- Mixed local and marketplace extensions with `--all`: only marketplace extensions updated.
