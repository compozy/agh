---
status: resolved
file: internal/api/core/channels.go
line: 74
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:595cbba5aa54
review_hash: 595cbba5aa54
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 001: Inconsistent whitespace handling for path parameter.
## Review Comment

`c.Param("id")` is not trimmed here, but lines 133, 196, 208, and 220 use `strings.TrimSpace(c.Param("id"))`. This inconsistency could cause lookup mismatches if IDs contain surrounding whitespace.

## Triage

- Decision: `invalid`
- Notes:
  - The current code at [internal/api/core/channels.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/core/channels.go:74) already trims the path parameter with `strings.TrimSpace(c.Param("id"))` before calling `GetInstance`.
  - This specific finding is stale against the current file contents, so there is no production change to make for this line.
  - Resolution: Closed as invalid after code inspection; `make verify` passed without any required change for this finding.
