---
status: resolved
file: internal/e2elane/command_wiring_test.go
line: 1
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:ed559e30b907
review_hash: ed559e30b907
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 008: Consider adding integration build tag for tests that execute external commands.
## Review Comment

These tests invoke `make` via `exec.CommandContext`, which requires the repository structure and build tools to be present. While the filename doesn't follow the `*_integration_test.go` pattern, tests that depend on external tooling and repo layout are typically gated behind build tags to avoid CI failures in constrained environments.

If these tests should run with regular unit tests, the current approach is acceptable. Otherwise, consider:
1. Renaming to `command_wiring_integration_test.go` with `//go:build integration`
2. Or adding a lighter build tag like `//go:build e2elane_wiring`

## Triage

- Decision: `invalid`
- Notes:
  - These tests intentionally guard the repository’s command-surface wiring by reading `package.json` and running `make -n` / `make help`; they do not execute the expensive E2E lanes themselves.
  - Keeping them in the default unit suite is desirable because it prevents silent regressions in the repo entrypoints that `make verify` is expected to protect.
  - Adding an integration tag here would weaken coverage without addressing a concrete failure mode in the current implementation.
