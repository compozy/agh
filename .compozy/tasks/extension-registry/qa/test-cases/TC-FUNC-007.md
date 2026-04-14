# TC-FUNC-007: Version Comparison Handles Semantic Versions

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/version.go` |

## Objective

Validate that `VersionIsNewer()` correctly compares semantic version strings across major, minor, patch, and pre-release components.

## Preconditions

- None (pure function, no setup required).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | `VersionIsNewer("1.0.0", "2.0.0")` | **Expected:** Returns `true` (2.0.0 is newer). |
| 2 | `VersionIsNewer("2.0.0", "1.0.0")` | **Expected:** Returns `false`. |
| 3 | `VersionIsNewer("1.0.0", "1.0.0")` | **Expected:** Returns `false` (same version). |
| 4 | `VersionIsNewer("1.0.0", "1.1.0")` | **Expected:** Returns `true`. |
| 5 | `VersionIsNewer("1.0.0", "1.0.1")` | **Expected:** Returns `true`. |
| 6 | `VersionIsNewer("v1.0.0", "v2.0.0")` | **Expected:** Returns `true` (v prefix handled). |
| 7 | `VersionIsNewer("1.0.0-beta", "1.0.0")` | **Expected:** Returns `true` (release > pre-release). |
| 8 | `VersionIsNewer("1.0.0-alpha", "1.0.0-beta")` | **Expected:** Returns `true` (beta > alpha). |

## Edge Cases

- Empty version string: should handle gracefully (error or false).
- Non-semver string (e.g., "latest"): should handle gracefully.
- Versions with build metadata (e.g., `1.0.0+build123`): metadata ignored per semver spec.
