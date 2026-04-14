# TC-FUNC-006: MultiRegistry CheckUpdate Detects Available Updates

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that `CheckUpdate()` correctly compares local and remote versions and reports when an update is available.

## Preconditions

- Stub source returns `Detail` with latest version `2.0.0`.
- Local extension installed at version `1.0.0`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `CheckUpdate(ctx, "test/pkg", "1.0.0")` | **Expected:** Returns `UpdateInfo{HasUpdate: true, Current: "1.0.0", Latest: "2.0.0"}`. |
| 2 | Call `CheckUpdate(ctx, "test/pkg", "2.0.0")` | **Expected:** Returns `UpdateInfo{HasUpdate: false}`. |
| 3 | Call `CheckUpdate(ctx, "test/pkg", "3.0.0")` (local ahead) | **Expected:** Returns `UpdateInfo{HasUpdate: false}`. |

## Edge Cases

- Version with `v` prefix (e.g., `v1.0.0` vs `1.0.0`): should normalize and compare correctly.
- Pre-release versions: `1.0.0-beta` < `1.0.0`.
- Invalid version string: should return error or treat as needing update.
