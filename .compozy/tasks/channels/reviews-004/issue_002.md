---
status: resolved
file: internal/daemon/channels_test.go
line: 30
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093927386,nitpick_hash:67675467b124
review_hash: 67675467b124
source_review_id: "4093927386"
source_review_submitted_at: "2026-04-11T15:47:00Z"
---

# Issue 002: Reshape this new suite into t.Run("Should...") cases.
## Review Comment

The coverage is useful, but this file mostly uses one top-level test per scenario instead of the repo's default subtest style. Grouping related cases into table-driven `t.Run("Should...")` blocks will reduce setup duplication and align the suite with the project's test conventions.

As per coding guidelines, `**/*_test.go`: `Use table-driven tests with subtests (t.Run) as default` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `valid`
- Notes:
  - `internal/daemon/channels_test.go` is mostly a list of single-scenario top-level tests even though this repository’s testing rules call for table/subtest-oriented `t.Run("Should...")` cases by default.
  - The coverage is useful, but the current layout duplicates setup and does not follow the test-shape convention expected in this workspace.
  - Fix approach: regroup the file into suite-style top-level tests that contain `t.Run("Should...")` cases while preserving the same behavior coverage.

## Resolution

- Reworked `internal/daemon/channels_test.go` into grouped suites that use `t.Run("Should...")` for the actual cases while keeping the original behavior coverage intact.
- Verified the reshaped suite still passes through `go test ./internal/daemon` and the full `make verify` gate.
