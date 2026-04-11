---
status: resolved
file: internal/config/config_test.go
line: 288
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093048586,nitpick_hash:20f43eb8dab7
review_hash: 20f43eb8dab7
source_review_id: "4093048586"
source_review_submitted_at: "2026-04-11T01:15:37Z"
---

# Issue 007: Comprehensive integration test - consider adding t.Parallel().
## Review Comment

This test thoroughly validates the MCP server merging behavior across all four config sources (home TOML, home JSON, workspace TOML, workspace JSON) and correctly asserts both merge strategies:
- TOML overlays use field-level merging (e.g., `partial` retains `Command` from global but gains `Args`/`Env` from workspace)
- JSON sidecars use whole-object replacement (e.g., `sidecar.Args` is cleared)

Consider adding `t.Parallel()` since the test uses `t.TempDir()` and `t.Setenv()` without shared mutable state.

## Triage

- Decision: `invalid`
- Notes:
- This test uses `t.Setenv("AGH_HOME", ...)`, which mutates process-global state. Parallelizing it would make it race with other config tests that also read or override environment variables.
- Go's testing guidance intentionally treats environment mutation as incompatible with parallel execution, so adding `t.Parallel()` here would reduce isolation rather than improve it.
- No code change is warranted.
