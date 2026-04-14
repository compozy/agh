# TC-INT-004: ClawHub Filters Non-Skill Package Types

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 2 min |
| **Module** | `internal/registry/clawhub/client.go` |

## Objective

Validate that ClawHub adapter returns empty results when searching for extension-type packages (ClawHub is skills-only).

## Preconditions

- Mock server with valid search endpoint.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Search(ctx, "test", SearchOpts{PackageType: PackageTypeExtension})` | **Expected:** Returns empty results (ClawHub does not serve extensions). |
| 2 | Call `Search(ctx, "test", SearchOpts{PackageType: PackageTypeSkill})` | **Expected:** Returns results normally. |
| 3 | Call `Search(ctx, "test", SearchOpts{})` (no type filter) | **Expected:** Returns skill results (default behavior). |

## Edge Cases

- Unknown package type: should return empty or error.
