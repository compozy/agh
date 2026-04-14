# TC-FUNC-018: CLI Extension Search With --from Flag

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/extension.go` |

## Objective

Validate that `--from` flag filters search to a specific registry source.

## Preconditions

- Multiple registry sources configured.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh extension search "test" --from clawhub` | **Expected:** Results only from ClawHub source. |
| 2 | Run `agh extension search "test" --from github` | **Expected:** Error or message: GitHub does not support search. |
| 3 | Run `agh extension search "test" --from nonexistent` | **Expected:** Error: unknown registry source "nonexistent". |

## Edge Cases

- Case sensitivity in `--from` value: should match case-insensitively or document expected case.
