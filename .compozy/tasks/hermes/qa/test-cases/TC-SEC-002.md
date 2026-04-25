## TC-SEC-002: Skill And Managed Extension Symlink Escape Rejection

**Priority:** P0 (Critical)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that skill loading and managed extension loading canonicalize paths and reject symlinks that resolve outside approved roots.

### Traceability

- Task: task_05, MCP Auth and Skill Security.
- TechSpec: issue 28; Testing Approach skill symlink escape rejection.
- ADR: ADR-003 security boundary for MCP/tool auth and managed extension paths.
- Surfaces: `internal/skills`, `internal/extension`, managed extension install paths, site skill/extension docs where security behavior is operator-visible.

### Preconditions

- Isolated temp skill root and managed extension root.
- One valid in-root symlink fixture and one out-of-root symlink escape fixture.
- No real user skill or extension directories are touched.

### Test Steps

1. Load a valid skill directory with normal files and in-root links.
   - **Expected:** Skill metadata and resources load successfully, and provenance hashing stays inside the root.

2. Load a skill whose declared files traverse a symlink to a path outside the skill root.
   - **Expected:** Loader rejects or safely ignores the escaped path and records a validation error without reading external content.

3. Install or inspect a managed extension containing a symlink escape.
   - **Expected:** Managed extension path validation fails before copying, executing, hashing, or exposing escaped content.

4. Confirm diagnostics contain paths but no external file contents.
   - **Expected:** Error output is bounded and does not leak contents from the escaped target.

5. Re-run with a symlink that resolves within the approved root.
   - **Expected:** In-root symlink behavior remains accepted where the implementation permits it.

### Evidence To Capture

- `qa/logs/TC-SEC-002/go-test-symlink-security.log`
- Fixture layout notes under `qa/logs/TC-SEC-002/fixtures.md`
- Any discrepancy in `qa/issues/BUG-*.md`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Relative symlink escape | `../outside/secret` | Rejected before read |
| Absolute symlink escape | `/tmp/outside/secret` | Rejected before read |
| Nested escape | Link inside subdir | Rejected after canonicalization |
| In-root symlink | Link target inside root | Accepted or safely processed per loader rules |

### Related Test Cases

- TC-FUNC-003: Extension environment diagnostics after safe loading.
