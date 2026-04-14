# TC-FUNC-008: MultiRegistry Close Propagates to All Sources

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 2 min |
| **Module** | `internal/registry/multi.go` |

## Objective

Validate that `MultiRegistry.Close()` calls `Close()` on all registered sources and aggregates any errors.

## Preconditions

- MultiRegistry with 3 stub sources.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Close()` | **Expected:** All 3 sources have `Close()` called. |
| 2 | Configure one source to return error on Close | **Expected:** Close still calls the other sources. Error is returned/logged. |
| 3 | Call Close twice | **Expected:** No panic, idempotent or returns "already closed" error. |

## Edge Cases

- Source Close blocks indefinitely: should not block other sources (or have a timeout).
