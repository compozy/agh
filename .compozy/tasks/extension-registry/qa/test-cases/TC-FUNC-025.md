# TC-FUNC-025: CLI Extension Update Check Mode

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `agh extension update --check` reports available updates without installing them.

## Preconditions

- Marketplace-installed extension at version 1.0.0.
- Registry source returns latest version 2.0.0.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension update --check` | **Expected:** Lists extensions with available updates: "ext-name: 1.0.0 -> 2.0.0". |
| 2 | Verify extension is NOT updated (still at 1.0.0) | **Expected:** DB `remote_version` still "1.0.0". Filesystem unchanged. |
| 3 | Run `agh extension update --check` with no updates available | **Expected:** "All extensions up to date" message. |

## Edge Cases

- No marketplace-installed extensions: should report "no marketplace extensions found".
- Registry unreachable during check: should report error per extension, not crash.
