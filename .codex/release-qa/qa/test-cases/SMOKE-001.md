## SMOKE-001: Repository Verification Gate

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

### Objective

Verify the canonical AGH release gate passes from the current workspace.

### Preconditions

- `go`, `bun`, and repository dependencies are available.
- Worktree state has been reviewed.

### Test Steps

1. Run `make verify`.
   **Expected:** Formatting, codegen check, web lint/typecheck/test/build, Go lint, race unit tests, build, and boundary checks complete with exit code 0.

2. Inspect output for warnings and failures.
   **Expected:** No warnings, errors, or failed tests remain.

### Edge Cases & Variations

| Variation           | Input                        | Expected Result                              |
| ------------------- | ---------------------------- | -------------------------------------------- |
| Missing web bundle  | `web/dist/index.html` absent | Gate rebuilds or reports actionable failure. |
| Generated API drift | Stale OpenAPI artifacts      | `CodegenCheck` fails before build claim.     |
