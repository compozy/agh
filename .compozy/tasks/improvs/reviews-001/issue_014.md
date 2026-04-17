---
status: resolved
file: internal/session/query.go
line: 225
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:a7300ea523df
review_hash: a7300ea523df
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 014: Add volume-name check to block Windows drive-relative paths.
## Review Comment

`filepath.IsAbs("C:foo")` returns false on Windows, and `filepath.Join(base, "C:foo")` can resolve outside the sessions root. The separator check does not catch this case since drive-relative paths contain no slash or backslash. This is a documented behavior in Go's filepath package that creates a path traversal vulnerability.

## Triage

- Decision: `VALID`
- Notes:
  `normalizeStoredSessionID` rejects absolute paths and separators but still
  accepts Windows drive-relative forms like `C:foo`, which can escape the
  sessions directory when joined on Windows. Plan: reject drive-relative
  prefixes explicitly and add unit coverage for that input shape.
