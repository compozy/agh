---
status: resolved
file: go.mod
line: 25
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:ebda09433582
review_hash: ebda09433582
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 001: Remove the unused github.com/pelletier/go-toml/v2 dependency.
## Review Comment

The v2.2.4 version is not imported anywhere in the codebase, making it an orphaned direct dependency that bloats go.mod. Only the v1 version is actively used (in `persistence.go` and `transport_parity_integration_test.go`). Remove `github.com/pelletier/go-toml/v2 v2.2.4` from go.mod.

As a secondary improvement, consider whether you can consolidate the two TOML libraries (`github.com/BurntSushi/toml` and `github.com/pelletier/go-toml` v1) to reduce the overall dependency footprint.

## Triage

- Decision: `INVALID`
- Notes:
  - Current code imports `github.com/pelletier/go-toml/v2/unstable` in `internal/config/persistence.go`, so `github.com/pelletier/go-toml/v2 v2.2.4` is not an orphaned dependency.
  - Removing the v2 requirement would break the config persistence code that relies on the v2 unstable AST package.
