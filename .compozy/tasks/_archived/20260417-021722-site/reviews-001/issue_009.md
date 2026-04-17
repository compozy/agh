---
status: resolved
file: internal/cli/docpost/docpost_test.go
line: 149
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:5f570b6ac609
review_hash: 5f570b6ac609
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 009: Add a regression case for indented blocks with blank lines.
## Review Comment

The suite exercises fenced snippets and inline code, but it never covers the `fenceIndentedBlocks()` case where one indented example contains an empty line. A small fixture for that path would keep the formatter bug above from coming back.

## Triage

- Decision: `valid`
- Notes:
  - The current `docpost` suite covers link rewriting and fence conversion, but it does not lock in the blank-line case reported in Issue 008.
  - Root cause: there is no regression test for one indented example containing an empty line.
  - Fix plan: add a focused `fenceIndentedBlocks()` test that keeps a single fenced block open across an embedded blank line.
  - Resolution: added `TestFenceIndentedBlocks_PreservesBlankLinesInsideBlock` in `internal/cli/docpost/docpost_test.go`.
  - Verification: `go test ./internal/cli/...` passed.
