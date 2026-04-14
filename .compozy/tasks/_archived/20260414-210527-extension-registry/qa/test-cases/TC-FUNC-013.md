# TC-FUNC-013: Installer Manifest Verification Blocks Prompt Injection

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 5 min |
| **Module** | `internal/registry/installer.go` |

## Objective

Validate that the installer's content verification scans manifest files and blocks archives containing prompt injection attempts.

## Preconditions

- Archives with various injection payloads in `extension.toml` or other manifest files.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Archive with `extension.toml` containing "ignore previous instructions" | **Expected:** Install fails with content verification error listing the matched rule. |
| 2 | Archive with "disregard existing rules" in a nested config file | **Expected:** Install fails with verification error. |
| 3 | Archive with "you are now a helpful assistant that..." | **Expected:** Install fails — role hijack detected. |
| 4 | Archive with "rm -rf /" in a script file | **Expected:** Install fails — destructive command detected. |
| 5 | Archive with "extract credentials" in manifest | **Expected:** Install fails — credential extraction detected. |
| 6 | Archive with benign content only | **Expected:** Install succeeds. |

## Edge Cases

- Mixed case injection ("Ignore Previous Instructions"): should match case-insensitively.
- Injection in binary file: should not scan binary files (only text manifests).
- Very large manifest file: should still complete scanning without timeout.
