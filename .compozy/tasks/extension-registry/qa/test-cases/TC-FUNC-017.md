# TC-FUNC-017: CLI Extension Search With Default Source

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `agh extension search <query>` queries configured registry sources and displays formatted results.

## Preconditions

- AGH binary built.
- Registry sources configured via TOML config.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension search "oauth"` | **Expected:** Table output with columns: Name, Version, Author, Source. Exit code 0. |
| 2 | Run `agh extension search "nonexistent-xyz-12345"` | **Expected:** "No results found" message. Exit code 0. |
| 3 | Run `agh extension search` (no query) | **Expected:** Error: query argument required. Non-zero exit code. |

## Edge Cases

- Query with special characters: should be URL-encoded before sending to registry.
- Very long query string: should not panic.
