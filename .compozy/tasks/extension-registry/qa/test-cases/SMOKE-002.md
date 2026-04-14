# SMOKE-002: Extension Search Returns Results

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Smoke |
| **Estimated Time** | 3 min |
| **Module** | CLI / Registry |

## Objective

Validate that `agh extension search` queries available registry sources and returns formatted results.

## Preconditions

- AGH binary built and available in PATH.
- At least one searchable registry source configured (ClawHub for skills).
- Network access to configured registry (or mock server running).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension search "test"` | **Expected:** Results displayed with name, version, author, source columns. Or "no results" message if no matches. Exit code 0. |
| 2 | Run `agh extension search "test" --limit 1` | **Expected:** At most 1 result returned. |
| 3 | Run `agh extension search "test" --from github` | **Expected:** Error or empty results (GitHub search not supported). Clear message about capability. |

## Edge Cases

- Empty query string: should return an error or usage message.
- No network: should fail gracefully with timeout/connection error.
