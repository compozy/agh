---
status: resolved
file: Makefile
line: 50
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:6d2f9182ace8
review_hash: 6d2f9182ace8
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 001: Avoid duplicating the CLI default output path in Make target.
## Review Comment

Since `agh doc` already has a default output dir, consider removing `--output-dir ...` here (or centralizing this path in one variable) to reduce drift risk.

## Triage

- Decision: `invalid`
- Notes:
  - The `cli-docs` Make target intentionally pins the site output path as the contract for `make cli-docs`.
  - Removing `--output-dir` would make the Makefile silently follow any future CLI default-path change instead of keeping the site path explicit at the entrypoint people actually use.
  - Drift risk is already covered by the existing `defaultCLIDocsDir` command-flag test in `internal/cli/doc_test.go`, so no code change is warranted here.
