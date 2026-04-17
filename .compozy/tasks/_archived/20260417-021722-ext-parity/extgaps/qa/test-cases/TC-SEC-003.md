# TC-SEC-003: Path traversal prevented in bundle file loading

**Priority:** P1 (High)
**Type:** Security
**Component:** `internal/extension/bundle.go` — `collectBundleFiles()`, `resolveResourcePath()`

## Objective

Validate that bundle file loading does not allow path traversal to read files outside the extension's root directory.

## Preconditions

- Extension with declared bundle resource paths

## Test Steps

1. Bundle resource path `"../../etc/passwd"` in manifest
   **Expected:** Error during resource path resolution or file not found within extension root

2. Bundle resource path `"bundles/../../../secret.toml"`
   **Expected:** Error or resolved to safe path within extension root

3. Symlink in bundle directory pointing outside extension root
   **Expected:** WalkDir follows symlink — verify this is documented as a known behavior or mitigated

4. Absolute path in bundle resource `"/etc/passwd"`
   **Expected:** Behavior depends on resolveResourcePath — verify it either rejects or safely resolves

5. Bundle path with null bytes `"bundles\x00.toml"`
   **Expected:** OS rejects null byte in path

## Edge Cases

- Resource path "." → collects all bundle files from extension root
- Resource path to specific file → only that file loaded
- Resource path to directory → recursively collects .toml and .json files
