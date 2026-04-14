# TC-FUNC-029: Skill Update Via Migrated Pipeline

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/skill_commands.go` |

## Objective

Validate that `agh skill update` works correctly through the migrated pipeline using `.agh-meta.json` for version tracking.

## Preconditions

- Skill installed via marketplace with `.agh-meta.json` containing version info.
- Registry returns a newer version.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh skill update --check` | **Expected:** Shows skills with available updates and their version delta. |
| 2 | Run `agh skill update <skill-name>` | **Expected:** Downloads and installs new version. `.agh-meta.json` updated. |
| 3 | Verify old skill files replaced with new version | **Expected:** Directory contents match new version. |

## Edge Cases

- `.agh-meta.json` missing or corrupt: should handle gracefully (treat as unknown version).
- Skill installed locally (no `.agh-meta.json`): skipped during update check.
