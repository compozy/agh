# TC-FUNC-009: Installer Full Pipeline — Download, Extract, Verify, Move

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/installer.go` |

## Objective

Validate the complete install pipeline: download archive, extract to temp dir, verify manifest content, compute checksum, and move to target directory.

## Preconditions

- Valid tar.gz archive created via `mustTarGz()` containing `extension.toml` manifest.
- Stub downloader returning the archive as a reader.
- Target directory in `t.TempDir()`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Create `NewInstaller()` with default options | **Expected:** Installer created without error. |
| 2 | Call `Install(ctx, "test/ext", dlOpts, targetDir)` | **Expected:** Returns `InstallResult` with slug, version, checksum, install path. |
| 3 | Verify target directory contains extracted files | **Expected:** `extension.toml` and other files present at `targetDir/`. |
| 4 | Verify temp directory cleaned up | **Expected:** No `.agh-install-*` directories remain in parent of target. |
| 5 | Verify checksum is a valid SHA256 hex string | **Expected:** 64-character hex string. |

## Edge Cases

- Archive with single root directory: extraction should walk into it.
- Empty archive: should fail with "no manifest found" error.
- Archive with no `extension.toml`: should fail with manifest validation error.
