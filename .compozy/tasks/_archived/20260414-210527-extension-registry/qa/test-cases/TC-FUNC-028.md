# TC-FUNC-028: Skill Search Via Migrated Pipeline

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 2 min |
| **Module** | `internal/cli/skill_commands.go` |

## Objective

Validate that `agh skill search` works correctly after migration from legacy `internal/skills/marketplace/` to `MultiRegistry`.

## Preconditions

- ClawHub configured as skill registry source.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh skill search "test"` | **Expected:** Results from ClawHub displayed in table format. |
| 2 | Run `agh skill search "test" --limit 5` | **Expected:** At most 5 results returned. |
| 3 | Verify no references to old `internal/skills/marketplace/` package | **Expected:** Old package deleted; skill commands use `internal/registry/` only. |

## Edge Cases

- Same as TC-FUNC-017 edge cases (empty query, special characters).
