# SMOKE-001: Make Verify Gate Passes

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Smoke |
| **Estimated Time** | 5 min |
| **Module** | Build System |

## Objective

Validate that the full verification gate (`make verify`) passes after extension registry changes, confirming no regressions in formatting, linting, tests, or build.

## Preconditions

- Extension registry branch checked out with all 5 tasks merged.
- Go toolchain and `golangci-lint` installed.
- Dependencies resolved (`make deps`).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `make fmt` | **Expected:** No files modified; exit code 0. |
| 2 | Run `make lint` | **Expected:** Zero warnings, zero errors; exit code 0. |
| 3 | Run `make test` | **Expected:** All unit tests pass with `-race`; exit code 0. |
| 4 | Run `make build` | **Expected:** Binary compiles successfully; exit code 0. |
| 5 | Run `make verify` (combines all above) | **Expected:** All stages pass sequentially; exit code 0. |

## Edge Cases

- Stale build cache: run `go clean -testcache` before step 5 to verify clean pass.

## Related Tests

- All other test cases depend on this gate passing first.
