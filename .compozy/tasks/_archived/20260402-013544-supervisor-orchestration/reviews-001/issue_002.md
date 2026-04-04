---
status: resolved
file: internal/kernel/prompt_catalog.go
line: 11
severity: medium
author: claude-code
---

# Issue 002: Missing direct unit tests for buildPromptCatalogsForAgent



## Review Comment

`buildPromptCatalogsForAgent` has its own file (`prompt_catalog.go`) but no corresponding test file. The function is tested indirectly through integration tests in `session_manager_test.go` and `api_lifecycle_test.go`, but several edge cases and branches are not exercised:

1. **Nil session** (line 12) - returns empty strings without error
2. **Non-master agent type** (line 12) - returns empty strings without error
3. **Empty workspace** (line 22) - returns only roles catalog
4. **`config.ResolvePaths` failure** (line 27) - returns error
5. **`config.LoadPlaybooks` failure** (line 31) - degrades gracefully

These branches are load-bearing for the graceful-degradation contract the function establishes. A refactoring that changes the error handling could break callers silently.

Add a `prompt_catalog_test.go` with table-driven tests covering each branch. The tests can use `t.TempDir()` for workspace isolation and mock role catalogs for the session.

## Triage

- Decision: `valid`
- Notes:
  The current coverage is only indirect. The function has several branchy behaviors that are part of its contract, including non-master bypass, empty-workspace fallback, and graceful degradation when catalog metadata cannot be loaded. A direct unit test file is warranted so future refactors do not silently change those branches.
  Resolved by adding `internal/kernel/prompt_catalog_test.go` with direct coverage for nil session, non-master bypass, empty workspace, `ResolvePaths` degradation, and malformed playbook degradation.
